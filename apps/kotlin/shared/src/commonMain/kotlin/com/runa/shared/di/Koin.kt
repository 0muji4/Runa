package com.runa.shared.di

import com.runa.shared.feature.health.HealthzViewModel
import com.runa.shared.network.ApiClient
import com.runa.shared.network.HttpClientFactory
import com.runa.shared.network.KtorApiClient
import org.koin.core.context.startKoin
import org.koin.core.module.Module
import org.koin.dsl.module
import org.koin.mp.KoinPlatform

/**
 * DI entry point. Each platform calls this once at startup, passing its own
 * build-config base URL (host+port only, no /api/v1 suffix).
 *
 * Kept to a single `String` parameter so the signature is identical for Android
 * (called from [Application.onCreate]) and Swift (via SKIE). Nothing in the
 * walking skeleton needs an Android `Context`, so none is threaded through here;
 * when a feature slice needs one, add a platform-specific `androidContext(...)`
 * declaration inside a platform overload rather than widening this signature.
 *
 * @param baseUrl e.g. http://10.0.2.2:8080 (Android emulator) or
 * http://localhost:8080 (iOS simulator).
 */
fun initKoin(baseUrl: String) {
    startKoin {
        modules(sharedModule(baseUrl))
    }
}

private fun sharedModule(baseUrl: String): Module = module {
    single { HttpClientFactory.create() }
    single<ApiClient> { KtorApiClient(httpClient = get(), baseUrl = baseUrl) }
    // TODO: feature slices register their own view models / repositories here.
    // `single` (not `factory`): HealthzViewModel owns a long-lived CoroutineScope,
    // so one shared instance avoids leaking a scope per resolution. Revisit when
    // view models gain per-screen lifecycles.
    single { HealthzViewModel(apiClient = get()) }
}

/**
 * Resolve [HealthzViewModel] from the started Koin graph.
 *
 * Android obtains view models with `koinInject()` inside Compose; Swift has no such
 * helper, so this top-level function (exported by SKIE as
 * `KoinKt.resolveHealthzViewModel()`) is the iOS-facing entry point. Resolving
 * through Koin — rather than calling the constructor directly — keeps the injected
 * base URL wiring intact on both platforms.
 */
fun resolveHealthzViewModel(): HealthzViewModel = KoinPlatform.getKoin().get()
