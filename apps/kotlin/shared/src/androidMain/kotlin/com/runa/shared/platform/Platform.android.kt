package com.runa.shared.platform

import app.cash.sqldelight.db.SqlDriver
import io.ktor.client.engine.HttpClientEngine
import io.ktor.client.engine.okhttp.OkHttp

/** Android uses the OkHttp Ktor engine. */
actual fun httpClientEngine(): HttpClientEngine = OkHttp.create()

/**
 * TODO: return an `AndroidSqliteDriver(RunaDatabase.Schema, context, "runa.db")`
 * once local persistence is needed. No feature uses the DB yet, so this stays a
 * stub to keep the DI graph honest.
 */
actual fun createSqlDriver(): SqlDriver =
    TODO("Android SQLDelight driver not wired yet — no feature uses the DB")

/** TODO: back with EncryptedSharedPreferences (androidx.security.crypto). */
actual class SecureStorage {
    actual fun get(key: String): String? =
        TODO("SecureStorage.get not implemented")

    actual fun set(key: String, value: String): Unit =
        TODO("SecureStorage.set not implemented")

    actual fun remove(key: String): Unit =
        TODO("SecureStorage.remove not implemented")
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
