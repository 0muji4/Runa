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
import com.runa.shared.feature.gallery.DefaultGalleryRepository
import com.runa.shared.feature.gallery.GalleryRepository
import com.runa.shared.feature.gallery.GalleryViewModel
import com.runa.shared.feature.gallery.ImageDetailViewModel
import com.runa.shared.feature.health.HealthzViewModel
import com.runa.shared.feature.insight.DefaultInsightRepository
import com.runa.shared.feature.insight.InsightRepository
import com.runa.shared.feature.insight.InsightViewModel
import com.runa.shared.feature.lock.AppLockRepository
import com.runa.shared.feature.lock.AppLockViewModel
import com.runa.shared.feature.lock.DefaultAppLockRepository
import com.runa.shared.feature.notification.DefaultNotificationSettingsRepository
import com.runa.shared.feature.notification.NotificationSettingsRepository
import com.runa.shared.feature.notification.NotificationSettingsViewModel
import com.runa.shared.feature.push.DeviceRegistrar
import com.runa.shared.feature.push.InstallIdProvider
import com.runa.shared.feature.push.PendingRoute
import com.runa.shared.feature.push.PushTokenStore
import com.runa.shared.feature.settings.AccountViewModel
import com.runa.shared.feature.settings.DefaultLocalDataCleaner
import com.runa.shared.feature.settings.DefaultSettingsRepository
import com.runa.shared.feature.settings.DefaultThemeRepository
import com.runa.shared.feature.settings.LocalDataCleaner
import com.runa.shared.feature.settings.SettingsRepository
import com.runa.shared.feature.settings.SettingsViewModel
import com.runa.shared.feature.settings.ThemeRepository
import com.runa.shared.feature.settings.ThemeViewModel
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
import com.runa.shared.network.KtorStorageClient
import com.runa.shared.network.StorageClient
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

internal val BARE_CLIENT = named("bareClient")
internal val AUTH_CLIENT = named("authClient")
internal val STORAGE_CLIENT = named("storageClient")
/** "android" / "ios", bound by each platformModule; sent as RegisterDeviceRequest.platform. */
internal val PLATFORM_NAME = named("platformName")

/**
 * DI entry point for platforms that need no Context (iOS: `doInitKoin(baseUrl:)`); Android has its own overload.
 * @param baseUrl host+port only (e.g. http://localhost:8080), no /api/v1 suffix.
 */
fun initKoin(baseUrl: String) {
    startKoin {
        modules(sharedModule(baseUrl), platformModule())
    }
}

/**
 * The platform-agnostic bindings. Keep the dependency order acyclic: bareClient → tokenStore →
 * refresher → authClient → apiClient → repository; forced logout flows back only as [TokenStore.sessionExpired].
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

    // Must NOT be the auth client: it would attach the Runa token to the object-storage host.
    single(STORAGE_CLIENT) { HttpClientFactory.createStorage(httpClientEngine()) }
    single<StorageClient> { KtorStorageClient(client = get(STORAGE_CLIENT)) }

    single { PushTokenStore(settings = get()) }
    single { InstallIdProvider(settings = get()) }
    single { PendingRoute() }

    single<AuthRepository> {
        DefaultAuthRepository(apiClient = get(), tokenStore = get(), installIdProvider = get())
    }

    single { RunaDatabase(driver = get()) }
    single<DiaryRepository> {
        DefaultDiaryRepository(database = get(), apiClient = get(), networkMonitor = get())
    }
    single<TodayRepository> { DefaultTodayRepository(apiClient = get(), database = get()) }
    single<SongRepository> { DefaultSongRepository(apiClient = get(), database = get()) }

    single<CalendarRepository> { DefaultCalendarRepository(diaryRepository = get(), apiClient = get()) }
    single<TodayMoonRepository> { DefaultTodayMoonRepository() }

    single<InsightRepository> { DefaultInsightRepository(diaryRepository = get()) }

    single<GalleryRepository> {
        DefaultGalleryRepository(
            database = get(),
            apiClient = get(),
            storageClient = get(),
            networkMonitor = get(),
        )
    }

    single<ThemeRepository> { DefaultThemeRepository(settings = get()) }
    single<LocalDataCleaner> { DefaultLocalDataCleaner(database = get()) }
    single<SettingsRepository> {
        DefaultSettingsRepository(
            apiClient = get(),
            authRepository = get(),
            localDataCleaner = get(),
        )
    }

    single<NotificationSettingsRepository> { DefaultNotificationSettingsRepository(settings = get()) }
    single<AppLockRepository> { DefaultAppLockRepository(settings = get()) }

    single(createdAtStart = true) {
        DeviceRegistrar(
            authRepository = get(),
            pushTokenStore = get(),
            notificationSettings = get(),
            installIdProvider = get(),
            networkMonitor = get(),
            apiClient = get(),
            settings = get(),
            platform = get(PLATFORM_NAME),
        ).also { it.start() }
    }

    // `single` のまま: :androidApp は koinInject() で解決し ViewModelStore に載っていないため、
    // `factory` にすると解決のたびに clear() されないインスタンスが増える。
    single { AuthViewModel(repository = get()) }
    single { HealthzViewModel(apiClient = get()) }
    single { DiaryListViewModel(repository = get()) }

    // Params are matched by type: an optional clientId (null = new entry) and an optional createdAt epoch-ms.
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

    factory { CalendarViewModel(repository = get()) }
    single { TodayMoonViewModel(repository = get()) }

    factory { InsightViewModel(repository = get()) }

    factory { params -> DayRecordsViewModel(repository = get(), isoDate = params.get()) }

    single { GalleryViewModel(repository = get()) }

    factory { params -> ImageDetailViewModel(repository = get(), startClientId = params.get()) }

    single { ThemeViewModel(repository = get()) }
    single { SettingsViewModel(themeRepository = get()) }
    single { AccountViewModel(repository = get()) }

    // The lock view model must outlive any screen so the gate keeps its state across foreground/background.
    single { NotificationSettingsViewModel(repository = get()) }
    single { AppLockViewModel(repository = get(), authenticator = get()) }
}

/** Resolve [HealthzViewModel] from the started Koin graph (iOS entry point). */
fun resolveHealthzViewModel(): HealthzViewModel = KoinPlatform.getKoin().get()

/** Resolve [AuthViewModel] from the started Koin graph (iOS entry point). */
fun resolveAuthViewModel(): AuthViewModel = KoinPlatform.getKoin().get()

/** Resolve [DiaryListViewModel] (iOS entry point). */
fun resolveDiaryListViewModel(): DiaryListViewModel = KoinPlatform.getKoin().get()

/** Resolve a [DiaryEditorViewModel] for an entry (iOS entry point); null [clientId] starts a new entry. */
fun resolveDiaryEditorViewModel(clientId: String?): DiaryEditorViewModel =
    KoinPlatform.getKoin().get { parametersOf(clientId) }

/** Resolve a [DiaryEditorViewModel] for a NEW entry backdated to [createdAtEpochMs] (iOS entry point). */
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

/** Resolve a fresh [InsightViewModel] (iOS entry point; starts at the current month). */
fun resolveInsightViewModel(): InsightViewModel = KoinPlatform.getKoin().get()

/** Resolve a [DayRecordsViewModel] for the day [isoDate] (`yyyy-MM-dd`) (iOS entry point). */
fun resolveDayRecordsViewModel(isoDate: String): DayRecordsViewModel =
    KoinPlatform.getKoin().get { parametersOf(isoDate) }

/** Resolve the [GalleryViewModel] from the started Koin graph (iOS entry point). */
fun resolveGalleryViewModel(): GalleryViewModel = KoinPlatform.getKoin().get()

/** Resolve an [ImageDetailViewModel] for the lightbox, focused on [startClientId] (iOS entry point). */
fun resolveImageDetailViewModel(startClientId: String): ImageDetailViewModel =
    KoinPlatform.getKoin().get { parametersOf(startClientId) }

/** Resolve the [ThemeViewModel] (iOS entry point; drives the picker and the root). */
fun resolveThemeViewModel(): ThemeViewModel = KoinPlatform.getKoin().get()

/** Resolve the [SettingsViewModel] for the settings top (iOS entry point). */
fun resolveSettingsViewModel(): SettingsViewModel = KoinPlatform.getKoin().get()

/** Resolve the [AccountViewModel] for the account-data screen (iOS entry point). */
fun resolveAccountViewModel(): AccountViewModel = KoinPlatform.getKoin().get()

/** Resolve the [NotificationSettingsViewModel] (iOS entry point). */
fun resolveNotificationSettingsViewModel(): NotificationSettingsViewModel =
    KoinPlatform.getKoin().get()

/** Resolve the [AppLockViewModel] for the privacy-lock gate (iOS entry point). */
fun resolveAppLockViewModel(): AppLockViewModel = KoinPlatform.getKoin().get()

/** Resolve the [PushTokenStore]; the AppDelegate feeds the APNs token into it (iOS entry point). */
fun resolvePushTokenStore(): PushTokenStore = KoinPlatform.getKoin().get()

/** Resolve the [PendingRoute] the notification tap sets and the root view consumes (iOS entry point). */
fun resolvePendingRoute(): PendingRoute = KoinPlatform.getKoin().get()

/** Resolve the [DeviceRegistrar] to [DeviceRegistrar.refresh] on foreground (iOS entry point). */
fun resolveDeviceRegistrar(): DeviceRegistrar = KoinPlatform.getKoin().get()
