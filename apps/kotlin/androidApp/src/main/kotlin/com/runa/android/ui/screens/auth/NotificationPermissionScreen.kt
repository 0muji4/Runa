package com.runa.android.ui.screens.auth

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
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
import com.runa.android.ui.components.NotificationMoon
import com.runa.android.ui.theme.RunaColors

/**
 * Notification permission shell (④). A quiet night-time request: the moon with a
 * small moonlight-pink bell badge, a poetic 明朝 line, and a gentle ask. The real
 * runtime permission request belongs to the notification slice; this advances.
 */
@Composable
fun NotificationPermissionScreen(
    onContinue: () -> Unit,
    onSkip: () -> Unit,
) {
    Surface(color = RunaColors.Background) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(horizontal = 40.dp, vertical = 48.dp),
            verticalArrangement = Arrangement.Center,
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            NotificationMoon(diameter = 156.dp)
            Spacer(Modifier.height(36.dp))
            Text(
                text = stringResource(R.string.notif_title),
                style = MaterialTheme.typography.headlineMedium,
                color = RunaColors.Heading,
                textAlign = TextAlign.Center,
            )
            Spacer(Modifier.height(18.dp))
            Text(
                text = stringResource(R.string.notif_body),
                style = MaterialTheme.typography.bodyMedium,
                color = RunaColors.Subtle,
                textAlign = TextAlign.Center,
            )
            Spacer(Modifier.height(44.dp))
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(56.dp)
                    .background(RunaColors.Accent, RoundedCornerShape(16.dp))
                    .clickable(onClick = onContinue),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    text = stringResource(R.string.notif_allow),
                    style = MaterialTheme.typography.bodyLarge,
                    color = RunaColors.Background,
                )
            }
            Spacer(Modifier.height(18.dp))
            Text(
                text = stringResource(R.string.action_skip),
                style = MaterialTheme.typography.labelLarge.copy(letterSpacing = 4.sp),
                color = RunaColors.Subtle,
                modifier = Modifier
                    .clickable(onClick = onSkip)
                    .padding(12.dp),
            )
        }
    }
}
