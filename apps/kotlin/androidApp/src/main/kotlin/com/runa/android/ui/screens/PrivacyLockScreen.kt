package com.runa.android.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.runa.android.R
import com.runa.android.ui.components.LockEmblem
import com.runa.android.ui.theme.RunaColors
import com.runa.shared.feature.lock.AppLockViewModel
import org.koin.compose.koinInject

/**
 * プライバシー・ロック (22). The confirmed design's パスコード / Face ID / すぐにロック
 * controls are simplified to a single ON/OFF toggle (per the agreed spec-minimal
 * model): when on, the app requires biometric — with the device passcode as
 * fallback — on launch/resume. Shows a quiet notice when the device has no security
 * set up, so enabling the lock would be ineffective.
 */
@Composable
fun PrivacyLockScreen(
    onBack: () -> Unit,
    viewModel: AppLockViewModel = koinInject(),
) {
    val enabled by viewModel.lockEnabled.collectAsState()
    val available = viewModel.biometricAvailable()

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(RunaColors.Background)
            .padding(horizontal = 28.dp),
    ) {
        Spacer(Modifier.height(24.dp))
        Text(
            text = stringResource(R.string.action_back),
            color = RunaColors.Subtle,
            fontSize = 15.sp,
            modifier = Modifier.clickable(onClick = onBack),
        )
        Spacer(Modifier.height(24.dp))
        Text(
            text = stringResource(R.string.lock_settings_eyebrow),
            color = RunaColors.Subtle,
            fontSize = 13.sp,
            letterSpacing = 3.sp,
        )
        Spacer(Modifier.height(8.dp))
        Text(
            text = stringResource(R.string.lock_settings_title),
            color = RunaColors.Heading,
            fontWeight = FontWeight.Medium,
            fontSize = 40.sp,
        )

        Spacer(Modifier.height(48.dp))
        Box(Modifier.fillMaxWidth(), contentAlignment = Alignment.Center) {
            LockEmblem()
        }
        Spacer(Modifier.height(28.dp))
        Text(
            text = stringResource(R.string.lock_settings_caption),
            color = RunaColors.Body,
            fontSize = 18.sp,
            modifier = Modifier.fillMaxWidth(),
            textAlign = TextAlign.Center,
        )

        Spacer(Modifier.height(48.dp))
        Row(
            modifier = Modifier.fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = stringResource(R.string.lock_settings_toggle),
                color = RunaColors.Heading,
                fontSize = 17.sp,
                modifier = Modifier.weight(1f),
            )
            Switch(
                checked = enabled,
                onCheckedChange = { viewModel.setLockEnabled(it) },
                colors = SwitchDefaults.colors(
                    checkedThumbColor = RunaColors.Background,
                    checkedTrackColor = RunaColors.Accent,
                    checkedBorderColor = RunaColors.Accent,
                    uncheckedThumbColor = RunaColors.Subtle,
                    uncheckedTrackColor = RunaColors.Surface,
                    uncheckedBorderColor = RunaColors.Subtle,
                ),
            )
        }
        if (!available) {
            Spacer(Modifier.height(16.dp))
            Text(
                text = stringResource(R.string.lock_settings_unavailable),
                color = RunaColors.Subtle,
                fontSize = 13.sp,
            )
        }
    }
}
