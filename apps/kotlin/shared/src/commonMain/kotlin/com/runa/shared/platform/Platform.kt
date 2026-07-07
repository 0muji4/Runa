package com.runa.shared.platform

import app.cash.sqldelight.db.SqlDriver
import io.ktor.client.engine.HttpClientEngine
import org.koin.core.module.Module

/**
 * Platform seams (expect/actual).
 *
 * These declare the platform-specific frames the shared code needs. Most bodies
 * are still TODO stubs; the auth slice fills in secure storage (via
 * [platformModule], which binds a
 * [com.runa.shared.network.auth.SecureKeyValueStore]) and the HTTP engine.
 */

/** Ktor engine per platform: OkHttp on Android, Darwin on iOS. */
expect fun httpClientEngine(): HttpClientEngine

/** SQLDelight driver per platform. Schema is empty for now; DB is unused. */
expect fun createSqlDriver(): SqlDriver

/**
 * Platform-specific Koin bindings, merged into the graph by `initKoin`. Today it
 * provides the secure key-value store (Android: EncryptedSharedPreferences —
 * needs a Context; iOS: Keychain). Keeping this per-platform is how an Android
 * Context is threaded in without widening the common `initKoin(baseUrl)`.
 */
expect fun platformModule(): Module

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
