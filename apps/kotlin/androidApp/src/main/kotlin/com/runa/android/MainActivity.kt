package com.runa.android

import android.os.Bundle
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.fragment.app.FragmentActivity
import com.runa.android.navigation.RunaApp
import com.runa.android.ui.screens.lock.AppLockGate
import com.runa.android.ui.theme.RunaTheme
import com.runa.shared.feature.lock.AppLockViewModel
import com.runa.shared.feature.lock.CurrentActivityHolder
import com.runa.shared.feature.settings.ThemeViewModel
import org.koin.android.ext.android.inject
import org.koin.compose.koinInject

/**
 * A [FragmentActivity] (not a bare ComponentActivity) because androidx.biometric
 * BiometricPrompt requires one. The privacy-lock gate wraps the whole app: the
 * shared [AppLockViewModel] is driven by this Activity's lifecycle
 * (foreground/background), and the current Activity is registered so the biometric
 * authenticator can present its prompt.
 */
class MainActivity : FragmentActivity() {

    private val appLockViewModel: AppLockViewModel by inject()

    override fun onCreate(savedInstanceState: Bundle?) {
        enableEdgeToEdge()
        super.onCreate(savedInstanceState)
        setContent {
            // The selected theme (persisted in shared) drives the whole app; changing
            // it recomposes every screen against the new palette.
            val themeViewModel: ThemeViewModel = koinInject()
            val theme by themeViewModel.theme.collectAsState()
            RunaTheme(theme = theme) {
                AppLockGate(viewModel = appLockViewModel) {
                    RunaApp()
                }
            }
        }
    }

    override fun onResume() {
        super.onResume()
        // Register before prompting so the biometric authenticator has an Activity.
        CurrentActivityHolder.set(this)
        appLockViewModel.onAppForegrounded()
    }

    override fun onPause() {
        appLockViewModel.onAppBackgrounded()
        CurrentActivityHolder.clear(this)
        super.onPause()
    }
}
