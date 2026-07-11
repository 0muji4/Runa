package com.runa.android.navigation

import androidx.annotation.StringRes
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
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
    const val GALLERY = "gallery"
    const val SETTINGS = "settings"
}

/** Route builders for the diary sub-screens (clientId is a UUID, path-safe). */
fun diaryEditorRoute(clientId: String): String = "diary/editor/$clientId"
fun diaryDetailRoute(clientId: String): String = "diary/detail/$clientId"

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
            NavigationBar {
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
