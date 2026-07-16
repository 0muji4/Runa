package com.runa.shared.di

import com.runa.shared.db.RunaDatabase
import com.runa.shared.feature.auth.AuthRepository
import com.runa.shared.feature.auth.AuthViewModel
import com.runa.shared.feature.auth.DefaultAuthRepository
import com.runa.shared.feature.calendar.CalendarRepository
import com.runa.shared.feature.calendar.CalendarViewModel
import com.runa.shared.feature.calendar.DayRecordsViewModel
import com.runa.shared.feature.calendar.DefaultCalendarRepository
import com.runa.shared.feature.diary.DefaultDiaryRepository
import com.runa.shared.feature.diary.DiaryEditorViewModel
import com.runa.shared.feature.diary.DiaryListViewModel
import com.runa.shared.feature.diary.DiaryRepository
import com.runa.shared.feature.health.HealthzViewModel
import com.runa.shared.feature.todaymoon.DefaultTodayMoonRepository
import com.runa.shared.feature.todaymoon.TodayMoonRepository
import com.runa.shared.feature.todaymoon.TodayMoonViewModel
import com.runa.shared.feature.today.DefaultSongRepository
import com.runa.shared.feature.today.DefaultTodayRepository
import com.runa.shared.feature.today.HomeViewModel
import com.runa.shared.feature.today.SongArchiveViewModel
import com.runa.shared.feature.today.SongRepository
import com.runa.shared.feature.today.TodayRepository
import com.runa.shared.feature.today.player.SongPlayerViewModel
import com.runa.shared.network.ApiClient
import com.runa.shared.network.HttpClientFactory
import com.runa.shared.network.KtorApiClient
import com.runa.shared.network.auth.TokenRefresher
import com.runa.shared.network.auth.TokenStore
import com.runa.shared.platform.httpClientEngine
import com.runa.shared.platform.platformModule
import org.koin.core.context.startKoin
import org.koin.core.module.Module
import org.koin.core.parameter.parametersOf
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

    // Local persistence. The SqlDriver + NetworkMonitor come from platformModule();
    // the database and repositories are shared singletons.
    single { RunaDatabase(driver = get()) }
    single<DiaryRepository> {
        DefaultDiaryRepository(database = get(), apiClient = get(), networkMonitor = get())
    }
    single<TodayRepository> { DefaultTodayRepository(apiClient = get(), database = get()) }
    single<SongRepository> { DefaultSongRepository(apiClient = get(), database = get()) }

    // Calendar composes the local diary stream with the shared moon calc; the moon
    // screen is a pure offline computation (no store needed).
    single<CalendarRepository> { DefaultCalendarRepository(diaryRepository = get(), apiClient = get()) }
    single<TodayMoonRepository> { DefaultTodayMoonRepository() }

    // `single` (not `factory`): these view models own long-lived CoroutineScopes,
    // so one shared instance avoids leaking a scope per resolution.
    single { AuthViewModel(repository = get()) }
    single { HealthzViewModel(apiClient = get()) }
    single { DiaryListViewModel(repository = get()) }

    // The editor is per-entry: `factory` so each open gets a fresh scope/state.
    // Params are matched by type: an optional clientId (null = new entry) and an
    // optional createdAt epoch-ms (calendar "write on this day" backdate).
    factory { params ->
        DiaryEditorViewModel(
            repository = get(),
            clientId = params.getOrNull<String>(),
            createdAtEpochMs = params.getOrNull<Long>(),
        )
    }

    single { HomeViewModel(repository = get()) }
    single { SongPlayerViewModel(audioPlayer = get(), songRepository = get()) }
    single { SongArchiveViewModel(repository = get()) }

    // Calendar view model is a `factory` so each open starts at today's month.
    factory { CalendarViewModel(repository = get()) }
    single { TodayMoonViewModel(repository = get()) }

    // Per-day records: `factory` keyed by the tapped ISO date (yyyy-MM-dd).
    factory { params -> DayRecordsViewModel(repository = get(), isoDate = params.get()) }
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

/**
 * Resolve [DiaryListViewModel] (iOS entry point, `KoinKt.resolveDiaryListViewModel()`).
 */
fun resolveDiaryListViewModel(): DiaryListViewModel = KoinPlatform.getKoin().get()

/**
 * Resolve a [DiaryEditorViewModel] for an entry (iOS entry point). Pass null to
 * start a new entry, or an existing entry's clientId to edit it.
 */
fun resolveDiaryEditorViewModel(clientId: String?): DiaryEditorViewModel =
    KoinPlatform.getKoin().get { parametersOf(clientId) }

/**
 * Resolve a [DiaryEditorViewModel] for a NEW entry backdated to [createdAtEpochMs]
 * (the calendar's "write on this day" flow). iOS entry point.
 */
fun resolveNewDiaryEditorViewModelOn(createdAtEpochMs: Long): DiaryEditorViewModel =
    KoinPlatform.getKoin().get { parametersOf(createdAtEpochMs) }

/** Resolve [HomeViewModel] from the started Koin graph (iOS entry point). */
fun resolveHomeViewModel(): HomeViewModel = KoinPlatform.getKoin().get()

/** Resolve [SongPlayerViewModel] from the started Koin graph (iOS entry point). */
fun resolveSongPlayerViewModel(): SongPlayerViewModel = KoinPlatform.getKoin().get()

/** Resolve [SongArchiveViewModel] from the started Koin graph (iOS entry point). */
fun resolveSongArchiveViewModel(): SongArchiveViewModel = KoinPlatform.getKoin().get()

/** Resolve a fresh [CalendarViewModel] (iOS entry point; starts at today's month). */
fun resolveCalendarViewModel(): CalendarViewModel = KoinPlatform.getKoin().get()

/** Resolve [TodayMoonViewModel] from the started Koin graph (iOS entry point). */
fun resolveTodayMoonViewModel(): TodayMoonViewModel = KoinPlatform.getKoin().get()

/** Resolve a [DayRecordsViewModel] for a tapped calendar day (iOS entry point).
 *  [isoDate] is the day as `yyyy-MM-dd`. */
fun resolveDayRecordsViewModel(isoDate: String): DayRecordsViewModel =
    KoinPlatform.getKoin().get { parametersOf(isoDate) }
