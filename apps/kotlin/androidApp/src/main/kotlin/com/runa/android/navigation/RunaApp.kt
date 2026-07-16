package com.runa.android.navigation

import androidx.annotation.StringRes
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import com.runa.android.ui.theme.RunaColors
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.res.stringResource
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
import com.runa.android.ui.screens.SettingsScreen
import com.runa.android.ui.screens.SongArchiveScreen
import com.runa.android.ui.screens.SplashScreen
import com.runa.android.ui.screens.TodaysSongScreen
import com.runa.android.ui.screens.auth.AuthFlow
import com.runa.shared.feature.auth.AuthState
import com.runa.shared.feature.auth.AuthViewModel
import org.koin.compose.koinInject

/** App routes. Tab routes appear in the bottom bar; settings is a pushed screen. */
object Routes {
    const val HOME = "home"
    const val TODAYS_SONG = "todays_song"
    const val SONG_ARCHIVE = "song_archive"
    const val DIARY = "diary"
    const val DIARY_EDITOR_NEW = "diary/editor"
    const val DIARY_EDITOR_EDIT = "diary/editor/{clientId}"
    const val DIARY_DETAIL = "diary/detail/{clientId}"
    const val DIARY_WRITE_ON = "diary/write-on/{date}"
    const val CALENDAR = "calendar"
    const val DAY_RECORDS = "calendar/day/{date}"
    const val TODAYS_MOON = "todays_moon"
    const val GALLERY = "gallery"
    const val SETTINGS = "settings"
}

/** Route builders for the diary sub-screens (clientId is a UUID, path-safe). */
fun diaryEditorRoute(clientId: String): String = "diary/editor/$clientId"
fun diaryDetailRoute(clientId: String): String = "diary/detail/$clientId"

/** Route builders for the calendar sub-screens (date is ISO yyyy-MM-dd, path-safe). */
fun dayRecordsRoute(isoDate: String): String = "calendar/day/$isoDate"
fun diaryWriteOnRoute(isoDate: String): String = "diary/write-on/$isoDate"

/**
 * Root auth gate. Subscribes to the shared [AuthViewModel] and switches the whole
 * app between the startup splash, the unauthenticated flow, and the tab body:
 *  - [AuthState.Restoring]      → splash (checking the stored session)
 *  - [AuthState.Authenticated]  → the tabbed app, greeting the /me display name
 *  - anything else              → onboarding → sign-in
 *
 * Signing out from Settings flips the state back to unauthenticated, so this gate
 * returns to the sign-in flow automatically.
 */
@Composable
fun RunaApp(authViewModel: AuthViewModel = koinInject()) {
    val state by authViewModel.state.collectAsState()

    when (val current = state) {
        is AuthState.Restoring -> SplashScreen()
        is AuthState.Authenticated -> RunaTabs(
            displayName = current.user.displayName,
            onSignOut = { authViewModel.logout() },
        )
        else -> AuthFlow(state = current, authViewModel = authViewModel)
    }
}

/**
 * The authenticated tab shell (the former app root). Four bottom tabs plus a
 * pushed Settings screen, which now also hosts sign-out.
 */
@Composable
fun RunaTabs(
    displayName: String,
    onSignOut: () -> Unit,
) {
    val navController = rememberNavController()

    Scaffold(
        bottomBar = {
            val backStackEntry by navController.currentBackStackEntryAsState()
            val currentDestination = backStackEntry?.destination
            NavigationBar(containerColor = RunaColors.Background) {
                RunaTab.entries.forEach { tab ->
                    val selected = currentDestination?.hierarchy?.any { it.route == tab.route } == true
                    NavigationBarItem(
                        selected = selected,
                        onClick = {
                            navController.navigate(tab.route) {
                                popUpTo(navController.graph.findStartDestination().id) {
                                    saveState = true
                                }
                                launchSingleTop = true
                                restoreState = true
                            }
                        },
                        icon = { Text(tab.glyph) },
                        label = { Text(stringResource(tab.labelRes)) },
                        colors = NavigationBarItemDefaults.colors(
                            selectedIconColor = RunaColors.Accent,
                            selectedTextColor = RunaColors.Accent,
                            unselectedIconColor = RunaColors.Subtle,
                            unselectedTextColor = RunaColors.Subtle,
                            indicatorColor = androidx.compose.ui.graphics.Color.Transparent,
                        ),
                    )
                }
            }
        },
    ) { innerPadding ->
        NavHost(
            navController = navController,
            startDestination = Routes.HOME,
            modifier = androidx.compose.ui.Modifier.padding(innerPadding),
        ) {
            composable(Routes.HOME) {
                HomeScreen(
                    displayName = displayName,
                    onSettingsClick = { navController.navigate(Routes.SETTINGS) },
                    onOpenTodaysMoon = { navController.navigate(Routes.TODAYS_MOON) },
                )
            }
            composable(Routes.TODAYS_SONG) {
                TodaysSongScreen(
                    onOpenArchive = { navController.navigate(Routes.SONG_ARCHIVE) },
                )
            }
            composable(Routes.SONG_ARCHIVE) {
                SongArchiveScreen(
                    onPlayAndReturn = { navController.popBackStack() },
                    onBack = { navController.popBackStack() },
                )
            }
            composable(Routes.DIARY) {
                DiaryListScreen(
                    onOpenEntry = { clientId -> navController.navigate(diaryDetailRoute(clientId)) },
                    onNewEntry = { navController.navigate(Routes.DIARY_EDITOR_NEW) },
                    onOpenCalendar = { navController.navigate(Routes.CALENDAR) },
                )
            }
            composable(Routes.DIARY_EDITOR_NEW) {
                DiaryEditorScreen(clientId = null, onDone = { navController.popBackStack() })
            }
            composable(
                Routes.DIARY_EDITOR_EDIT,
                arguments = listOf(navArgument("clientId") { type = NavType.StringType }),
            ) { entry ->
                DiaryEditorScreen(
                    clientId = entry.arguments?.getString("clientId"),
                    onDone = { navController.popBackStack() },
                )
            }
            composable(
                Routes.DIARY_DETAIL,
                arguments = listOf(navArgument("clientId") { type = NavType.StringType }),
            ) { entry ->
                DiaryDetailScreen(
                    clientId = entry.arguments?.getString("clientId").orEmpty(),
                    onEdit = { clientId -> navController.navigate(diaryEditorRoute(clientId)) },
                    onDeleted = { navController.popBackStack() },
                    onBack = { navController.popBackStack() },
                )
            }
            composable(Routes.CALENDAR) {
                CalendarScreen(
                    onOpenDayRecords = { iso -> navController.navigate(dayRecordsRoute(iso)) },
                    onWriteOnDay = { iso -> navController.navigate(diaryWriteOnRoute(iso)) },
                    onBack = { navController.popBackStack() },
                )
            }
            composable(
                Routes.DAY_RECORDS,
                arguments = listOf(navArgument("date") { type = NavType.StringType }),
            ) { entry ->
                val date = entry.arguments?.getString("date").orEmpty()
                DayRecordsScreen(
                    isoDate = date,
                    onOpenEntry = { clientId -> navController.navigate(diaryDetailRoute(clientId)) },
                    onWrite = { navController.navigate(diaryWriteOnRoute(date)) },
                    onBack = { navController.popBackStack() },
                )
            }
            composable(
                Routes.DIARY_WRITE_ON,
                arguments = listOf(navArgument("date") { type = NavType.StringType }),
            ) { entry ->
                DiaryEditorScreen(
                    clientId = null,
                    onDone = { navController.popBackStack() },
                    backdateIsoDate = entry.arguments?.getString("date"),
                )
            }
            composable(Routes.TODAYS_MOON) {
                TodaysMoonScreen(onBack = { navController.popBackStack() })
            }
            composable(Routes.GALLERY) { GalleryScreen() }
            composable(Routes.SETTINGS) {
                SettingsScreen(
                    onBack = { navController.popBackStack() },
                    onSignOut = onSignOut,
                )
            }
        }
    }
}

/**
 * The four bottom-navigation tabs. A minimal text glyph stands in for an icon so
 * the skeleton avoids a hard dependency on the material-icons artifact.
 */
private enum class RunaTab(
    val route: String,
    @StringRes val labelRes: Int,
    val glyph: String,
) {
    HOME(Routes.HOME, R.string.tab_home, "◐"),
    TODAYS_SONG(Routes.TODAYS_SONG, R.string.tab_todays_song, "♪"),
    DIARY(Routes.DIARY, R.string.tab_diary, "❏"),
    GALLERY(Routes.GALLERY, R.string.tab_gallery, "❖"),
}
