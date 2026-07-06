package com.runa.shared.platform

import app.cash.sqldelight.db.SqlDriver
import io.ktor.client.engine.HttpClientEngine

/**
 * Platform seams (expect/actual).
 *
 * These declare the platform-specific frames the shared code needs. Bodies are
 * intentionally TODO stubs on each platform for now — the walking skeleton only
 * exercises [httpClientEngine]. The rest exist so feature slices can fill them in
 * without reshaping the DI graph later.
 */

/** Ktor engine per platform: OkHttp on Android, Darwin on iOS. */
expect fun httpClientEngine(): HttpClientEngine

/** SQLDelight driver per platform. Schema is empty for now; DB is unused. */
expect fun createSqlDriver(): SqlDriver

/** Keychain / EncryptedSharedPreferences-backed key-value storage. */
expect class SecureStorage {
    fun get(key: String): String?
    fun set(key: String, value: String)
    fun remove(key: String)
}

/** Provider for the current push notification token (FCM / APNs). */
expect class PushTokenProvider {
    suspend fun currentToken(): String?
}

/** In-app billing entry point. Placeholder until monetization lands. */
expect class BillingClient

/** Biometric (Face ID / fingerprint) gate. */
expect class BiometricAuthenticator {
    suspend fun authenticate(): Boolean
}
