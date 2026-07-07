package com.runa.shared.di

import com.runa.shared.feature.auth.AuthRepository
import com.runa.shared.feature.auth.AuthViewModel
import com.runa.shared.feature.auth.DefaultAuthRepository
import com.runa.shared.feature.health.HealthzViewModel
import com.runa.shared.network.ApiClient
import com.runa.shared.network.HttpClientFactory
import com.runa.shared.network.KtorApiClient
import com.runa.shared.network.auth.TokenRefresher
import com.runa.shared.network.auth.TokenStore
import com.runa.shared.platform.httpClientEngine
import com.runa.shared.platform.platformModule
import org.koin.core.context.startKoin
import org.koin.core.module.Module
import org.koin.core.qualifier.named
import org.koin.dsl.module
import org.koin.mp.KoinPlatform

// Koin qualifiers for the two HTTP clients (bare vs. authenticated).
internal val BARE_CLIENT = named("bareClient")
internal val AUTH_CLIENT = named("authClient")

/**
 * DI entry point for iOS (via SKIE as `doInitKoin(baseUrl:)`) and the fallback
 * for platforms that need no Context. Android uses its own overload that also
 * supplies an `androidContext(...)` (see Koin.android.kt).
 *
 * @param baseUrl host+port only (e.g. http://localhost:8080), no /api/v1 suffix.
 */
fun initKoin(baseUrl: String) {
    startKoin {
        modules(sharedModule(baseUrl), platformModule())
    }
}

/**
 * The platform-agnostic bindings. Marked `internal` (not `private`) so the
 * Android `initKoin(context, baseUrl)` overload in androidMain can reuse it.
 *
 * Dependency order is acyclic: bareClient → tokenStore → refresher → authClient →
 * apiClient → repository. Forced logout flows the other way as an event through
 * [TokenStore.sessionExpired], so nothing needs a back-reference.
 */
internal fun sharedModule(baseUrl: String): Module = module {
    single(BARE_CLIENT) { HttpClientFactory.createBase(httpClientEngine()) }

    single { TokenStore(store = get()) }
    single { TokenRefresher(bareClient = get(BARE_CLIENT), baseUrl = baseUrl, tokenStore = get()) }

    single(AUTH_CLIENT) {
        HttpClientFactory.createAuthenticated(
            engine = httpClientEngine(),
            tokenStore = get(),
            refresher = get(),
        )
    }

    single<ApiClient> { KtorApiClient(httpClient = get(AUTH_CLIENT), baseUrl = baseUrl) }

    single<AuthRepository> { DefaultAuthRepository(apiClient = get(), tokenStore = get()) }

    // `single` (not `factory`): these view models own long-lived CoroutineScopes,
    // so one shared instance avoids leaking a scope per resolution.
    single { AuthViewModel(repository = get()) }
    single { HealthzViewModel(apiClient = get()) }
}

/**
 * Resolve [HealthzViewModel] from the started Koin graph (iOS entry point,
 * exported by SKIE as `KoinKt.resolveHealthzViewModel()`).
 */
fun resolveHealthzViewModel(): HealthzViewModel = KoinPlatform.getKoin().get()

/**
 * Resolve [AuthViewModel] from the started Koin graph (iOS entry point, exported
 * by SKIE as `KoinKt.resolveAuthViewModel()`).
 */
fun resolveAuthViewModel(): AuthViewModel = KoinPlatform.getKoin().get()
