package com.runa.shared.platform

import android.content.Context
import android.content.SharedPreferences
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import app.cash.sqldelight.db.SqlDriver
import com.runa.shared.network.auth.SecureKeyValueStore
import io.ktor.client.engine.HttpClientEngine
import io.ktor.client.engine.okhttp.OkHttp
import org.koin.android.ext.koin.androidContext
import org.koin.core.module.Module
import org.koin.dsl.module

/** Android uses the OkHttp Ktor engine. */
actual fun httpClientEngine(): HttpClientEngine = OkHttp.create()

/**
 * TODO: return an `AndroidSqliteDriver(RunaDatabase.Schema, context, "runa.db")`
 * once local persistence is needed. No feature uses the DB yet, so this stays a
 * stub to keep the DI graph honest.
 */
actual fun createSqlDriver(): SqlDriver =
    TODO("Android SQLDelight driver not wired yet — no feature uses the DB")

/** Provides the EncryptedSharedPreferences-backed secure store, pulling the
 *  Context from Koin's androidContext. */
actual fun platformModule(): Module = module {
    single<SecureKeyValueStore> { EncryptedPrefsStore(androidContext()) }
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

/** TODO: back with androidx.biometric BiometricPrompt. */
actual class BiometricAuthenticator {
    actual suspend fun authenticate(): Boolean =
        TODO("BiometricAuthenticator.authenticate not implemented")
}
