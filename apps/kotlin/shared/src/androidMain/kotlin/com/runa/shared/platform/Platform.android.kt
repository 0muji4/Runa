package com.runa.shared.platform

import android.content.Context
import android.content.SharedPreferences
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import java.io.IOException
import java.security.GeneralSecurityException
import java.security.KeyStore
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
class EncryptedPrefsStore(private val context: Context) : SecureKeyValueStore {

    private val prefs: SharedPreferences by lazy {
        try {
            createEncryptedPrefs()
        } catch (e: GeneralSecurityException) {
            // The Tink keyset can no longer be decrypted by the AndroidKeyStore
            // master key — e.g. the key was rotated/invalidated across a reinstall
            // or a lock-screen change. Left unhandled this throws AEADBadTagException
            // and crashes at startup (restoreSession reads tokens here). Recovery:
            // discard the corrupted store + master key so a fresh keyset is created.
            // Secrets are lost; the app falls back to unauthenticated and re-login.
            recoverFromCorruptedKeystore()
            createEncryptedPrefs()
        } catch (e: IOException) {
            recoverFromCorruptedKeystore()
            createEncryptedPrefs()
        }
    }

    private fun createEncryptedPrefs(): SharedPreferences {
        val masterKey = MasterKey.Builder(context)
            .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
            .build()
        return EncryptedSharedPreferences.create(
            context,
            PREFS_NAME,
            masterKey,
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
        )
    }

    /** Drops the encrypted prefs file (holding the wrapped keyset) and the master key. */
    private fun recoverFromCorruptedKeystore() {
        context.deleteSharedPreferences(PREFS_NAME)
        try {
            KeyStore.getInstance("AndroidKeyStore").apply { load(null) }
                .deleteEntry(MasterKey.DEFAULT_MASTER_KEY_ALIAS)
        } catch (e: GeneralSecurityException) {
            // Master key already absent/unusable — the prefs deletion above is enough.
        } catch (e: IOException) {
            // Keystore unavailable; nothing more we can safely do here.
        }
    }

    override fun get(key: String): String? = prefs.getString(key, null)

    override fun set(key: String, value: String) {
        prefs.edit().putString(key, value).apply()
    }

    override fun remove(key: String) {
        prefs.edit().remove(key).apply()
    }

    private companion object {
        const val PREFS_NAME = "runa_secure_prefs"
    }
}

/** TODO: back with Firebase Cloud Messaging token retrieval. */
actual class PushTokenProvider {
    actual suspend fun currentToken(): String? =
        TODO("PushTokenProvider.currentToken not implemented")
}

/** TODO: back with Google Play Billing. Placeholder for now. */
actual class BillingClient
