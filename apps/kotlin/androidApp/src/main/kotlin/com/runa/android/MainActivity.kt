package com.runa.android

import android.os.Bundle
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.runtime.getValue
import androidx.fragment.app.FragmentActivity
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.runa.android.navigation.RunaApp
import com.runa.android.ui.screens.lock.AppLockGate
import com.runa.android.ui.theme.RunaTheme
import com.runa.shared.feature.lock.AppLockViewModel
import com.runa.shared.feature.lock.CurrentActivityHolder
import com.runa.shared.feature.settings.ThemeViewModel
import org.koin.android.ext.android.inject
import org.koin.compose.koinInject

/** Must be a [FragmentActivity] (not ComponentActivity): androidx.biometric requires one. */
class MainActivity : FragmentActivity() {

    private val appLockViewModel: AppLockViewModel by inject()

    override fun onCreate(savedInstanceState: Bundle?) {
        enableEdgeToEdge()
        super.onCreate(savedInstanceState)
        setContent {
            val themeViewModel: ThemeViewModel = koinInject()
            val theme by themeViewModel.theme.collectAsStateWithLifecycle()
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
