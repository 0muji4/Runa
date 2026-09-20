package com.runa.android.navigation

import androidx.annotation.StringRes
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.systemBarsPadding
import androidx.compose.material3.Icon
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.Scaffold
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.runa.android.ui.components.LocalReauthenticate
import com.runa.android.ui.components.RunaIcons
import com.runa.android.ui.theme.RunaColors
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.res.stringResource
import androidx.navigation.NavController
import androidx.navigation.NavDestination.Companion.hierarchy
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import com.runa.android.R
import com.runa.android.ui.screens.GalleryScreen
import com.runa.android.ui.screens.HomeScreen
import com.runa.android.ui.screens.TodaysMoonScreen
import com.runa.android.ui.screens.calendar.CalendarScreen
import com.runa.android.ui.screens.calendar.DayRecordsScreen
import com.runa.android.ui.screens.diary.DiaryDetailScreen
import com.runa.android.ui.screens.diary.DiaryEditorScreen
import com.runa.android.ui.screens.diary.DiaryListScreen
import com.runa.android.ui.screens.insight.InsightScreen
import com.runa.android.ui.screens.AccountScreen
import com.runa.android.ui.screens.NotificationSettingsScreen
import com.runa.android.ui.screens.PrivacyLockScreen
import com.runa.android.ui.screens.SettingsScreen
import com.runa.android.ui.screens.ThemeScreen
import com.runa.android.ui.screens.SongArchiveScreen
import com.runa.android.ui.screens.SplashScreen
import com.runa.android.ui.screens.TodaysSongScreen
import com.runa.android.ui.screens.auth.AuthFlow
import com.runa.shared.feature.auth.AuthState
import com.runa.shared.feature.auth.AuthViewModel
import com.runa.shared.feature.push.PendingRoute
import com.runa.shared.feature.push.PendingRouteKind
import org.koin.compose.koinInject

/** App routes. Tab routes live inside the tab shell; everything else is pushed. */
object Routes {
    const val TABS = "tabs"

    const val HOME = "home"
    const val TODAYS_SONG = "todays_song"
    const val DIARY = "diary"
    const val GALLERY = "gallery"

    const val SONG_ARCHIVE = "song_archive"
    const val DIARY_EDITOR_NEW = "diary/editor"
    const val DIARY_EDITOR_EDIT = "diary/editor/{clientId}"
    const val DIARY_DETAIL = "diary/detail/{clientId}"
    const val DIARY_WRITE_ON = "diary/write-on/{date}"
    const val CALENDAR = "calendar"
    const val DAY_RECORDS = "calendar/day/{date}"
    const val INSIGHT = "insight"
    const val TODAYS_MOON = "todays_moon"
    const val SETTINGS = "settings"
    const val THEME = "settings/theme"
    const val NOTIFICATIONS = "settings/notifications"
    const val PRIVACY_LOCK = "settings/privacy_lock"
    const val ACCOUNT = "settings/account"
}

/** clientId is a UUID (path-safe). */
fun diaryEditorRoute(clientId: String): String = "diary/editor/$clientId"
fun diaryDetailRoute(clientId: String): String = "diary/detail/$clientId"

/** date is ISO yyyy-MM-dd (path-safe). */
fun dayRecordsRoute(isoDate: String): String = "calendar/day/$isoDate"
fun diaryWriteOnRoute(isoDate: String): String = "diary/write-on/$isoDate"

/** Root auth gate: splash while restoring, tab body when authenticated, else the sign-in flow. */
@Composable
fun RunaApp(
    authViewModel: AuthViewModel = koinInject(),
    pendingRoute: PendingRoute = koinInject(),
) {
    val state by authViewModel.state.collectAsStateWithLifecycle()
    // Stable identity: a new lambda per recomposition would recompose the whole tab tree.
    val reauthenticate = remember(authViewModel) { { authViewModel.logout() } }

    // A notification tap must not fire after a later sign-in.
    LaunchedEffect(state) {
        if (state is AuthState.Unauthenticated) pendingRoute.consume()
    }

    when (val current = state) {
        is AuthState.Restoring -> SplashScreen()
        is AuthState.Authenticated ->
            CompositionLocalProvider(LocalReauthenticate provides reauthenticate) {
                RunaAuthenticatedApp(
                    displayName = current.user.displayName,
                    onSignOut = reauthenticate,
                )
            }
        else -> AuthFlow(state = current, authViewModel = authViewModel)
    }
}

/** Outer [NavHost] without a bottom bar: pushed screens cover the tab bar entirely. */
@Composable
fun RunaAuthenticatedApp(
    displayName: String,
    onSignOut: () -> Unit,
    pendingRoute: PendingRoute = koinInject(),
) {
    val rootNav = rememberNavController()

    NavHost(navController = rootNav, startDestination = Routes.TABS) {
        composable(Routes.TABS) {
            RunaTabs(displayName = displayName, rootNav = rootNav)
        }

        composable(Routes.SONG_ARCHIVE) {
            Pushed {
                SongArchiveScreen(
                    onPlayAndReturn = { rootNav.popBackStack() },
                    onBack = { rootNav.popBackStack() },
                )
            }
        }
        composable(Routes.DIARY_EDITOR_NEW) {
            Pushed { DiaryEditorScreen(clientId = null, onDone = { rootNav.popBackStack() }) }
        }
        composable(
            Routes.DIARY_EDITOR_EDIT,
            arguments = listOf(navArgument("clientId") { type = NavType.StringType }),
        ) { entry ->
            Pushed {
                DiaryEditorScreen(
                    clientId = entry.arguments?.getString("clientId"),
                    onDone = { rootNav.popBackStack() },
                )
            }
        }
        composable(
            Routes.DIARY_DETAIL,
            arguments = listOf(navArgument("clientId") { type = NavType.StringType }),
        ) { entry ->
            Pushed {
                DiaryDetailScreen(
                    clientId = entry.arguments?.getString("clientId").orEmpty(),
                    onEdit = { clientId -> rootNav.navigate(diaryEditorRoute(clientId)) },
                    onDeleted = { rootNav.popBackStack() },
                    onBack = { rootNav.popBackStack() },
                )
            }
        }
        composable(Routes.CALENDAR) {
            Pushed {
                CalendarScreen(
                    onOpenDayRecords = { iso -> rootNav.navigate(dayRecordsRoute(iso)) },
                    onWriteOnDay = { iso -> rootNav.navigate(diaryWriteOnRoute(iso)) },
                    onBack = { rootNav.popBackStack() },
                )
            }
        }
        composable(Routes.INSIGHT) {
            Pushed { InsightScreen(onBack = { rootNav.popBackStack() }) }
        }
        composable(
            Routes.DAY_RECORDS,
            arguments = listOf(navArgument("date") { type = NavType.StringType }),
        ) { entry ->
            val date = entry.arguments?.getString("date").orEmpty()
            Pushed {
                DayRecordsScreen(
                    isoDate = date,
                    onOpenEntry = { clientId -> rootNav.navigate(diaryDetailRoute(clientId)) },
                    onWrite = { rootNav.navigate(diaryWriteOnRoute(date)) },
                    onBack = { rootNav.popBackStack() },
                )
            }
        }
        composable(
            Routes.DIARY_WRITE_ON,
            arguments = listOf(navArgument("date") { type = NavType.StringType }),
        ) { entry ->
            Pushed {
                DiaryEditorScreen(
                    clientId = null,
                    onDone = { rootNav.popBackStack() },
                    backdateIsoDate = entry.arguments?.getString("date"),
                )
            }
        }
        composable(Routes.TODAYS_MOON) {
            Pushed { TodaysMoonScreen(onBack = { rootNav.popBackStack() }) }
        }
        composable(Routes.SETTINGS) {
            Pushed {
                SettingsScreen(
                    onBack = { rootNav.popBackStack() },
                    onOpenTheme = { rootNav.navigate(Routes.THEME) },
                    onOpenNotifications = { rootNav.navigate(Routes.NOTIFICATIONS) },
                    onOpenPrivacyLock = { rootNav.navigate(Routes.PRIVACY_LOCK) },
                    onOpenAccount = { rootNav.navigate(Routes.ACCOUNT) },
                )
            }
        }
        composable(Routes.THEME) {
            Pushed { ThemeScreen(onBack = { rootNav.popBackStack() }) }
        }
        composable(Routes.NOTIFICATIONS) {
            Pushed { NotificationSettingsScreen(onBack = { rootNav.popBackStack() }) }
        }
        composable(Routes.PRIVACY_LOCK) {
            Pushed { PrivacyLockScreen(onBack = { rootNav.popBackStack() }) }
        }
        composable(Routes.ACCOUNT) {
            Pushed {
                AccountScreen(
                    onBack = { rootNav.popBackStack() },
                    onSignOut = onSignOut,
                )
            }
        }
    }

    // After NavHost so the graph is set before the first navigate.
    val pending by pendingRoute.route.collectAsStateWithLifecycle()
    LaunchedEffect(pending) {
        if (pending == PendingRouteKind.DIARY_EDITOR_NEW) {
            rootNav.navigate(Routes.DIARY_EDITOR_NEW) { launchSingleTop = true }
            pendingRoute.consume()
        }
    }
}

/** Pushed screens have no [Scaffold], so system-bar insets are applied here. */
@Composable
private fun Pushed(content: @Composable () -> Unit) {
    Box(
        Modifier
            .fillMaxSize()
            .background(RunaColors.Background)
            .systemBarsPadding(),
    ) {
        content()
    }
}

/** Tab shell. Screens that should cover the tab bar navigate on [rootNav], not [tabNav]. */
@Composable
private fun RunaTabs(
    displayName: String,
    rootNav: NavController,
) {
    val tabNav = rememberNavController()

    Scaffold(
        containerColor = RunaColors.Background,
        bottomBar = {
            val backStackEntry by tabNav.currentBackStackEntryAsState()
            val currentDestination = backStackEntry?.destination
            NavigationBar(containerColor = RunaColors.Background) {
                RunaTab.entries.forEach { tab ->
                    val selected = currentDestination?.hierarchy?.any { it.route == tab.route } == true
                    NavigationBarItem(
                        selected = selected,
                        onClick = {
                            tabNav.navigate(tab.route) {
                                popUpTo(tabNav.graph.findStartDestination().id) {
                                    saveState = true
                                }
                                launchSingleTop = true
                                restoreState = true
                            }
                        },
                        icon = { Icon(tab.icon, contentDescription = stringResource(tab.labelRes)) },
                        colors = NavigationBarItemDefaults.colors(
                            selectedIconColor = RunaColors.Accent,
                            unselectedIconColor = RunaColors.Subtle,
                            indicatorColor = androidx.compose.ui.graphics.Color.Transparent,
                        ),
                    )
                }
            }
        },
    ) { innerPadding ->
        NavHost(
            navController = tabNav,
            startDestination = Routes.HOME,
            modifier = androidx.compose.ui.Modifier.padding(innerPadding),
        ) {
            composable(Routes.HOME) {
                HomeScreen(
                    displayName = displayName,
                    onSettingsClick = { rootNav.navigate(Routes.SETTINGS) },
                    onOpenTodaysMoon = { rootNav.navigate(Routes.TODAYS_MOON) },
                )
            }
            composable(Routes.TODAYS_SONG) {
                TodaysSongScreen(
                    onOpenArchive = { rootNav.navigate(Routes.SONG_ARCHIVE) },
                )
            }
            composable(Routes.DIARY) {
                DiaryListScreen(
                    onOpenEntry = { clientId -> rootNav.navigate(diaryDetailRoute(clientId)) },
                    onNewEntry = { rootNav.navigate(Routes.DIARY_EDITOR_NEW) },
                    onOpenCalendar = { rootNav.navigate(Routes.CALENDAR) },
                    onOpenInsight = { rootNav.navigate(Routes.INSIGHT) },
                )
            }
            composable(Routes.GALLERY) { GalleryScreen() }
        }
    }
}

/** Bottom tabs. No visible text labels; [labelRes] is the content description only. */
private enum class RunaTab(
    val route: String,
    @StringRes val labelRes: Int,
    val icon: ImageVector,
) {
    HOME(Routes.HOME, R.string.tab_home, RunaIcons.Moon),
    TODAYS_SONG(Routes.TODAYS_SONG, R.string.tab_todays_song, RunaIcons.MusicNote),
    DIARY(Routes.DIARY, R.string.tab_diary, RunaIcons.Document),
    GALLERY(Routes.GALLERY, R.string.tab_gallery, RunaIcons.Image),
}
