package com.runa.android.navigation

import androidx.annotation.StringRes
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.res.stringResource
import androidx.navigation.NavDestination.Companion.hierarchy
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import com.runa.android.R
import com.runa.android.ui.screens.DiaryScreen
import com.runa.android.ui.screens.GalleryScreen
import com.runa.android.ui.screens.HomeScreen
import com.runa.android.ui.screens.SettingsScreen
import com.runa.android.ui.screens.TodaysSongScreen

/** App routes. Tab routes appear in the bottom bar; settings is a pushed screen. */
object Routes {
    const val HOME = "home"
    const val TODAYS_SONG = "todays_song"
    const val DIARY = "diary"
    const val GALLERY = "gallery"
    const val SETTINGS = "settings"
}

/**
 * The four bottom-navigation tabs. A minimal text glyph stands in for an icon so
 * the skeleton avoids a hard dependency on the material-icons artifact; swap in
 * real vector icons later.
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

@Composable
fun RunaApp() {
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
                                // Standard bottom-nav behaviour: single instance per tab,
                                // preserve each tab's own back stack.
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
                HomeScreen(onSettingsClick = { navController.navigate(Routes.SETTINGS) })
            }
            composable(Routes.TODAYS_SONG) { TodaysSongScreen() }
            composable(Routes.DIARY) { DiaryScreen() }
            composable(Routes.GALLERY) { GalleryScreen() }
            composable(Routes.SETTINGS) {
                SettingsScreen(onBack = { navController.popBackStack() })
            }
        }
    }
}
