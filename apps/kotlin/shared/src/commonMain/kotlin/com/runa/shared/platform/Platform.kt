package com.runa.shared.platform

import io.ktor.client.engine.HttpClientEngine
import org.koin.core.module.Module

/** Ktor engine per platform: OkHttp on Android, Darwin on iOS. */
expect fun httpClientEngine(): HttpClientEngine

/** Platform-specific Koin bindings merged into the graph by `initKoin` (this is how an Android Context gets in). */
expect fun platformModule(): Module

/** Provider for the current push notification token (FCM / APNs). */
expect class PushTokenProvider {
    suspend fun currentToken(): String?
}

/** In-app billing entry point. Placeholder until monetization lands. */
expect class BillingClient
