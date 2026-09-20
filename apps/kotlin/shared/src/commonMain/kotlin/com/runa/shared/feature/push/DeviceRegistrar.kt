package com.runa.shared.feature.push

import com.runa.shared.feature.auth.AuthRepository
import com.runa.shared.feature.auth.AuthState
import com.runa.shared.feature.notification.NotificationSettingsRepository
import com.runa.shared.feature.notification.ReminderTime
import com.runa.shared.network.ApiClient
import com.runa.shared.network.NetworkMonitor
import com.runa.shared.network.dto.RegisterDeviceRequest
import com.russhwolf.settings.Settings
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant
import kotlinx.datetime.TimeZone

/**
 * Keeps this install's server-side device row (PUT /devices) in step with the signed-in user, the
 * push token, the reminder preference and the zone. Token and auth may arrive in either order.
 */
class DeviceRegistrar(
    private val authRepository: AuthRepository,
    private val pushTokenStore: PushTokenStore,
    private val notificationSettings: NotificationSettingsRepository,
    private val installIdProvider: InstallIdProvider,
    private val networkMonitor: NetworkMonitor,
    private val apiClient: ApiClient,
    private val settings: Settings,
    private val platform: String,
    private val clock: () -> Instant = Clock.System::now,
    private val timeZoneId: () -> String = { TimeZone.currentSystemDefault().id },
    private val scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Default),
) {
    private val mutex = Mutex()
    private val refreshTick = MutableStateFlow(0)
    private var job: Job? = null

    private data class Inputs(
        val auth: AuthState,
        val token: String?,
        val enabled: Boolean,
        val time: ReminderTime,
        val online: Boolean,
    )

    /** Idempotent; begins observing. */
    fun start() {
        if (job != null) return
        job = scope.launch {
            combine(
                authRepository.authState,
                pushTokenStore.token,
                notificationSettings.observeReminderEnabled(),
                notificationSettings.observeReminderTime(),
                networkMonitor.isOnline,
            ) { auth, token, enabled, time, online -> Inputs(auth, token, enabled, time, online) }
                .combine(refreshTick) { inputs, _ -> inputs }
                .collect { evaluate(it) }
        }
    }

    /** Re-evaluate with the current values (foreground, zone change). */
    fun refresh() {
        refreshTick.value = refreshTick.value + 1
    }

    private suspend fun evaluate(inputs: Inputs) {
        mutex.withLock {
            val auth = inputs.auth
            if (auth is AuthState.Unauthenticated) {
                settings.remove(KEY_FINGERPRINT)
                settings.remove(KEY_REGISTERED_AT)
                return
            }
            if (auth !is AuthState.Authenticated) return
            val token = inputs.token ?: return
            if (!inputs.online) return

            val zone = timeZoneId()
            val fingerprint = listOf(auth.user.id, token, inputs.enabled, inputs.time.label, zone).joinToString("|")
            val now = clock().toEpochMilliseconds()
            val registeredAt = settings.getLongOrNull(KEY_REGISTERED_AT)
            val fresh = registeredAt != null && now - registeredAt < REREGISTER_AFTER_MS
            if (fresh && settings.getStringOrNull(KEY_FINGERPRINT) == fingerprint) return

            val request = RegisterDeviceRequest(
                installId = installIdProvider.get(),
                pushToken = token,
                platform = platform,
                notifyTime = inputs.time.label,
                timeZone = zone,
                enabled = inputs.enabled,
            )
            try {
                apiClient.registerDevice(request)
            } catch (e: CancellationException) {
                throw e
            } catch (_: Exception) {
                // Nothing persisted, so the next trigger (online edge, foreground, daily) retries.
                return
            }
            settings.putString(KEY_FINGERPRINT, fingerprint)
            settings.putLong(KEY_REGISTERED_AT, now)
        }
    }

    private companion object {
        const val KEY_FINGERPRINT = "notif.push.registered_fingerprint"
        const val KEY_REGISTERED_AT = "notif.push.registered_at"
        const val REREGISTER_AFTER_MS = 24L * 60 * 60 * 1000
    }
}
