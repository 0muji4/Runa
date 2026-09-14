package com.runa.shared.platform

import app.cash.sqldelight.db.SqlDriver
import app.cash.sqldelight.driver.native.NativeSqliteDriver
import com.russhwolf.settings.ExperimentalSettingsImplementation
import com.russhwolf.settings.KeychainSettings
import com.russhwolf.settings.NSUserDefaultsSettings
import com.russhwolf.settings.Settings
import com.runa.shared.db.RunaDatabase
import com.runa.shared.feature.lock.BiometricAuthenticator
import com.runa.shared.feature.lock.IosBiometricAuthenticator
import com.runa.shared.feature.notification.IosLocalNotificationScheduler
import com.runa.shared.feature.notification.LocalNotificationScheduler
import com.runa.shared.feature.today.player.AudioPlayer
import com.runa.shared.feature.today.player.AvAudioPlayer
import com.runa.shared.network.NetworkMonitor
import com.runa.shared.network.auth.SecureKeyValueStore
import io.ktor.client.engine.HttpClientEngine
import io.ktor.client.engine.darwin.Darwin
import kotlinx.cinterop.ExperimentalForeignApi
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import org.koin.core.module.Module
import org.koin.dsl.module
import platform.Network.nw_path_get_status
import platform.Network.nw_path_monitor_create
import platform.Network.nw_path_monitor_set_queue
import platform.Network.nw_path_monitor_set_update_handler
import platform.Network.nw_path_monitor_start
import platform.Network.nw_path_status_satisfied
import platform.Foundation.NSUserDefaults
import platform.darwin.dispatch_queue_create

actual fun httpClientEngine(): HttpClientEngine = Darwin.create()

actual fun platformModule(): Module = module {
    single<SecureKeyValueStore> { KeychainSecureStore() }
    // Non-sensitive preferences (the app theme); tokens stay in the Keychain store above.
    single<Settings> { NSUserDefaultsSettings(NSUserDefaults.standardUserDefaults) }
    single<SqlDriver> { NativeSqliteDriver(RunaDatabase.Schema, "runa.db") }
    single<NetworkMonitor> { IosNetworkMonitor() }
    single<AudioPlayer> { AvAudioPlayer() }
    single<LocalNotificationScheduler> { IosLocalNotificationScheduler() }
    single<BiometricAuthenticator> { IosBiometricAuthenticator() }
}

/** [NetworkMonitor] over `NWPathMonitor`; "satisfied" is published as online. */
@OptIn(ExperimentalForeignApi::class)
class IosNetworkMonitor : NetworkMonitor {
    private val _isOnline = MutableStateFlow(true)
    override val isOnline: StateFlow<Boolean> = _isOnline.asStateFlow()

    init {
        val monitor = nw_path_monitor_create()
        nw_path_monitor_set_update_handler(monitor) { path ->
            _isOnline.value = nw_path_get_status(path) == nw_path_status_satisfied
        }
        nw_path_monitor_set_queue(monitor, dispatch_queue_create("com.runa.network.monitor", null))
        nw_path_monitor_start(monitor)
    }
}

/** Keychain-backed [SecureKeyValueStore] via multiplatform-settings' [KeychainSettings]. */
@OptIn(ExperimentalSettingsImplementation::class)
class KeychainSecureStore(
    private val settings: KeychainSettings = KeychainSettings(),
) : SecureKeyValueStore {

    override fun get(key: String): String? = settings.getStringOrNull(key)

    override fun set(key: String, value: String) {
        settings.putString(key, value)
    }

    override fun remove(key: String) {
        settings.remove(key)
    }
}

/** TODO: back with APNs device-token retrieval. */
actual class PushTokenProvider {
    actual suspend fun currentToken(): String? =
        TODO("PushTokenProvider.currentToken not implemented")
}

/** TODO: back with StoreKit. Placeholder for now. */
actual class BillingClient
