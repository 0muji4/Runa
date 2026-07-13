package com.runa.android.ui.screens.auth

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.runa.android.R
import com.runa.android.ui.components.GlowingMoon
import com.runa.android.ui.theme.RunaColors

/**
 * Onboarding (①②). Whitespace-first and spare, exactly as the design intends: a
 * softly glowing moon, one large left-aligned 明朝 line, and a quiet "すすむ" to
 * advance — no filled buttons, no body paragraph.
 */
@Composable
fun OnboardingScreen(
    titleRes: Int,
    onNext: () -> Unit,
) {
    Surface(color = RunaColors.Background) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(horizontal = 40.dp, vertical = 56.dp),
            verticalArrangement = Arrangement.SpaceBetween,
            horizontalAlignment = Alignment.Start,
        ) {
            GlowingMoon(diameter = 116.dp, modifier = Modifier.padding(top = 24.dp))

            Text(
                text = stringResource(titleRes),
                style = MaterialTheme.typography.headlineMedium.copy(fontSize = 32.sp, lineHeight = 48.sp),
                color = RunaColors.Heading,
            )

            Text(
                text = stringResource(R.string.onboarding_hint),
                style = MaterialTheme.typography.labelLarge.copy(letterSpacing = 6.sp),
                color = RunaColors.Subtle,
                textAlign = TextAlign.Center,
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable(onClick = onNext)
                    .padding(16.dp),
            )
        }
    }
}
