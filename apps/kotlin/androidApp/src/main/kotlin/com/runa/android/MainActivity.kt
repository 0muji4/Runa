package com.runa.android

import android.content.Intent
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
import com.runa.shared.feature.push.DeviceRegistrar
import com.runa.shared.feature.push.PendingRoute
import com.runa.shared.feature.push.PendingRouteKind
import com.runa.shared.feature.settings.ThemeViewModel
import org.koin.android.ext.android.inject
import org.koin.compose.koinInject

/** Must be a [FragmentActivity] (not ComponentActivity): androidx.biometric requires one. */
class MainActivity : FragmentActivity() {

    private val appLockViewModel: AppLockViewModel by inject()
    private val pendingRoute: PendingRoute by inject()
    private val deviceRegistrar: DeviceRegistrar by inject()

    override fun onCreate(savedInstanceState: Bundle?) {
        enableEdgeToEdge()
        super.onCreate(savedInstanceState)
        // Not on recreation: the launching intent is retained and the tap was already consumed.
        if (savedInstanceState == null) captureRoute(intent)
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

    // singleTop: a notification tap while running lands here instead of onCreate.
    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        captureRoute(intent)
    }

    override fun onResume() {
        super.onResume()
        // Register before prompting so the biometric authenticator has an Activity.
        CurrentActivityHolder.set(this)
        appLockViewModel.onAppForegrounded()
        deviceRegistrar.refresh()
    }

    override fun onPause() {
        appLockViewModel.onAppBackgrounded()
        CurrentActivityHolder.clear(this)
        super.onPause()
    }

    private fun captureRoute(intent: Intent?) {
        val data = intent?.data ?: return
        if (data.scheme == "runa" && data.host == "diary") {
            pendingRoute.set(PendingRouteKind.DIARY_EDITOR_NEW)
        }
    }
}
