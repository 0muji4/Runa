package com.runa.shared.platform

import io.ktor.client.engine.HttpClientEngine
import org.koin.core.module.Module

/**
 * Platform seams (expect/actual).
 *
 * These declare the platform-specific frames the shared code needs. The auth
 * slice fills in secure storage and the HTTP engine; the diary slice adds the
 * SQLDelight [app.cash.sqldelight.db.SqlDriver] and
 * [com.runa.shared.network.NetworkMonitor]; the today slice adds the
 * [com.runa.shared.feature.today.player.AudioPlayer]. All are bound in [platformModule].
 */

/** Ktor engine per platform: OkHttp on Android, Darwin on iOS. */
expect fun httpClientEngine(): HttpClientEngine

/**
 * Platform-specific Koin bindings, merged into the graph by `initKoin`. Provides
 * the secure key-value store (Android: EncryptedSharedPreferences — needs a
 * Context; iOS: Keychain), the SQLDelight [app.cash.sqldelight.db.SqlDriver], the
 * [com.runa.shared.network.NetworkMonitor], and the
 * [com.runa.shared.feature.today.player.AudioPlayer]. Keeping this per-platform is
 * how an Android Context is threaded in without widening the common `initKoin(baseUrl)`.
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
