package com.runa.shared.platform

import android.content.Context
import android.content.SharedPreferences
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import app.cash.sqldelight.db.SqlDriver
import com.russhwolf.settings.Settings
import com.russhwolf.settings.SharedPreferencesSettings
import app.cash.sqldelight.driver.android.AndroidSqliteDriver
import com.runa.shared.db.RunaDatabase
import com.runa.shared.feature.lock.AndroidBiometricAuthenticator
import com.runa.shared.feature.lock.BiometricAuthenticator
import com.runa.shared.feature.notification.AndroidLocalNotificationScheduler
import com.runa.shared.feature.notification.LocalNotificationScheduler
import com.runa.shared.feature.today.player.AudioPlayer
import com.runa.shared.feature.today.player.ExoAudioPlayer
import com.runa.shared.network.NetworkMonitor
import com.runa.shared.network.auth.SecureKeyValueStore
import io.ktor.client.engine.HttpClientEngine
import io.ktor.client.engine.okhttp.OkHttp
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import org.koin.android.ext.koin.androidContext
import org.koin.core.module.Module
import org.koin.dsl.module

/** Android uses the OkHttp Ktor engine. */
actual fun httpClientEngine(): HttpClientEngine = OkHttp.create()

/**
 * Android Koin bindings: the encrypted secure store, the SQLDelight driver
 * (persisted to `runa.db`), the connectivity monitor, and the ExoPlayer-backed
 * audio player. All pull the Context from Koin's androidContext.
 */
actual fun platformModule(): Module = module {
    single<SecureKeyValueStore> { EncryptedPrefsStore(androidContext()) }
    // Non-sensitive preferences (the app theme). Plain SharedPreferences — no need
    // for the encrypted store used by tokens.
    single<Settings> {
        SharedPreferencesSettings(
            androidContext().getSharedPreferences("runa_settings", Context.MODE_PRIVATE),
        )
    }
    single<SqlDriver> { AndroidSqliteDriver(RunaDatabase.Schema, androidContext(), "runa.db") }
    single<NetworkMonitor> { AndroidNetworkMonitor(androidContext()) }
    single<AudioPlayer> { ExoAudioPlayer(androidContext()) }
    // Nightly-reminder scheduling (AlarmManager + notification channel) and the
    // biometric gate (androidx.biometric BiometricPrompt). Both need a Context.
    single<LocalNotificationScheduler> { AndroidLocalNotificationScheduler(androidContext()) }
    single<BiometricAuthenticator> { AndroidBiometricAuthenticator(androidContext()) }
}

/**
 * [NetworkMonitor] over [ConnectivityManager]. Registers a default-network
 * callback and publishes whether a validated, internet-capable network exists.
 */
class AndroidNetworkMonitor(context: Context) : NetworkMonitor {
    private val connectivity =
        context.getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager

    private val _isOnline = MutableStateFlow(hasInternet())
    override val isOnline: StateFlow<Boolean> = _isOnline.asStateFlow()

    init {
        connectivity.registerDefaultNetworkCallback(object : ConnectivityManager.NetworkCallback() {
            override fun onAvailable(network: Network) { _isOnline.value = hasInternet() }
            override fun onLost(network: Network) { _isOnline.value = hasInternet() }
            override fun onCapabilitiesChanged(network: Network, caps: NetworkCapabilities) {
                _isOnline.value = caps.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
            }
        })
    }

    private fun hasInternet(): Boolean {
        val caps = connectivity.getNetworkCapabilities(connectivity.activeNetwork) ?: return false
        return caps.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
    }
}

/**
 * EncryptedSharedPreferences-backed [SecureKeyValueStore]. Values are encrypted
 * at rest with a hardware-backed master key.
 */
class EncryptedPrefsStore(context: Context) : SecureKeyValueStore {

    private val prefs: SharedPreferences by lazy {
        val masterKey = MasterKey.Builder(context)
            .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
            .build()
        EncryptedSharedPreferences.create(
            context,
            "runa_secure_prefs",
            masterKey,
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
        )
    }

    override fun get(key: String): String? = prefs.getString(key, null)

    override fun set(key: String, value: String) {
        prefs.edit().putString(key, value).apply()
    }

    override fun remove(key: String) {
        prefs.edit().remove(key).apply()
    }
}

/** TODO: back with Firebase Cloud Messaging token retrieval. */
actual class PushTokenProvider {
    actual suspend fun currentToken(): String? =
        TODO("PushTokenProvider.currentToken not implemented")
}

/** TODO: back with Google Play Billing. Placeholder for now. */
actual class BillingClient
