package com.runa.android

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import com.runa.android.navigation.RunaApp
import com.runa.android.ui.theme.RunaTheme
import com.runa.shared.feature.settings.ThemeViewModel
import org.koin.compose.koinInject

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        enableEdgeToEdge()
        super.onCreate(savedInstanceState)
        setContent {
            // The selected theme (persisted in shared) drives the whole app; changing
            // it recomposes every screen against the new palette.
            val themeViewModel: ThemeViewModel = koinInject()
            val theme by themeViewModel.theme.collectAsState()
            RunaTheme(theme = theme) {
                RunaApp()
            }
        }
    }
}
