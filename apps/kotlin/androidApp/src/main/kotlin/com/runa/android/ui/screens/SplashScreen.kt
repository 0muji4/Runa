package com.runa.android.ui.screens

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.runa.android.R
import com.runa.android.ui.components.GlowingMoon
import com.runa.android.ui.theme.RunaColors

/**
 * Quiet startup splash shown while [com.runa.shared.feature.auth.AuthState.Restoring]
 * — the app is checking the stored session. The glowing moon over the LUNA
 * wordmark, minimal decoration, in keeping with the design system.
 */
@Composable
fun SplashScreen() {
    Surface(color = RunaColors.Background) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(32.dp),
            verticalArrangement = Arrangement.Center,
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            GlowingMoon(diameter = 152.dp)
            Spacer(Modifier.height(22.dp))
            Text(
                text = stringResource(R.string.logo_wordmark),
                style = MaterialTheme.typography.displayLarge.copy(letterSpacing = 14.sp),
                color = RunaColors.Heading,
            )
            Spacer(Modifier.height(10.dp))
            Text(
                text = stringResource(R.string.splash_tagline),
                style = MaterialTheme.typography.bodyMedium,
                color = RunaColors.Subtle,
            )
            Spacer(Modifier.height(40.dp))
            CircularProgressIndicator(color = RunaColors.Accent, strokeWidth = 2.dp)
        }
    }
}
