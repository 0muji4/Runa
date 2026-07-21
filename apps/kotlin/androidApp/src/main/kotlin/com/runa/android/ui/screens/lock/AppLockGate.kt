package com.runa.android.ui.screens.lock

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.runa.android.R
import com.runa.android.ui.components.GlowingMoon
import com.runa.android.ui.theme.RunaColors
import com.runa.shared.feature.lock.AppLockUiState
import com.runa.shared.feature.lock.AppLockViewModel

/**
 * Privacy-lock gate — a layer SEPARATE from the auth gate. While the lock is
 * engaged the real [content] is NOT composed (so nothing private can flash behind
 * the lock); a quiet moon-motif lock screen with an unlock affordance is shown
 * instead. When the lock is off (or authentication succeeds, or no device security
 * exists) the content renders normally.
 *
 * The [AppLockViewModel] is the app-lifetime single also driven by MainActivity's
 * lifecycle (foreground/background); this gate only reflects its state and offers
 * the retry/unlock action.
 */
@Composable
fun AppLockGate(
    viewModel: AppLockViewModel,
    content: @Composable () -> Unit,
) {
    val state by viewModel.state.collectAsState()

    when (state) {
        AppLockUiState.Unlocked, AppLockUiState.Unavailable -> content()
        AppLockUiState.Locked -> LockScreen(authenticating = false, onUnlock = { viewModel.authenticate() })
        AppLockUiState.Authenticating -> LockScreen(authenticating = true, onUnlock = {})
    }
}

@Composable
private fun LockScreen(authenticating: Boolean, onUnlock: () -> Unit) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(RunaColors.Background)
            .padding(horizontal = 40.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        GlowingMoon(diameter = 132.dp)
        Spacer(Modifier.height(36.dp))
        Text(
            text = stringResource(R.string.lock_gate_message),
            color = RunaColors.Heading,
            fontSize = 22.sp,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(40.dp))
        if (authenticating) {
            Text(
                text = stringResource(R.string.lock_gate_authenticating),
                color = RunaColors.Subtle,
                fontSize = 14.sp,
            )
        } else {
            Text(
                text = stringResource(R.string.lock_gate_unlock),
                color = RunaColors.Accent,
                fontSize = 16.sp,
                letterSpacing = 4.sp,
                modifier = Modifier
                    .background(RunaColors.Surface, RoundedCornerShape(16.dp))
                    .clickable(onClick = onUnlock)
                    .padding(horizontal = 40.dp, vertical = 16.dp),
            )
        }
    }
}
