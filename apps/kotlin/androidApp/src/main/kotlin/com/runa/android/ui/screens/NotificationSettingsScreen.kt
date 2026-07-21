package com.runa.android.ui.screens

import android.Manifest
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TimePicker
import androidx.compose.material3.rememberTimePickerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.runa.android.R
import com.runa.android.ui.theme.RunaColors
import com.runa.shared.feature.notification.NotificationSettingsViewModel
import com.runa.shared.feature.notification.ReminderTime
import org.koin.compose.koinInject

/**
 * 通知設定 (21) — 夜のリマインド. A quiet toggle, a large time display, three preset
 * chips (21:00 / 22:00 / 23:00) and a free time picker, over a poetic footer.
 * Turning the reminder on requests POST_NOTIFICATIONS (API 33+); a denial doesn't
 * break the screen — the preference is still saved.
 */
@Composable
fun NotificationSettingsScreen(
    onBack: () -> Unit,
    viewModel: NotificationSettingsViewModel = koinInject(),
) {
    val state by viewModel.state.collectAsState()
    var showPicker by remember { mutableStateOf(false) }

    val permissionLauncher = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestPermission(),
    ) { /* granted or not, the preference is already saved; a denial just means the
           OS won't display it — the screen never breaks (DoD#3). */ }

    fun enableReminder() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            permissionLauncher.launch(Manifest.permission.POST_NOTIFICATIONS)
        }
        viewModel.onToggle(true)
    }

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
            text = stringResource(R.string.notif_settings_eyebrow),
            color = RunaColors.Subtle,
            fontSize = 13.sp,
            letterSpacing = 3.sp,
        )
        Spacer(Modifier.height(8.dp))
        Text(
            text = stringResource(R.string.notif_settings_title),
            color = RunaColors.Heading,
            fontWeight = FontWeight.Medium,
            fontSize = 40.sp,
        )
        Spacer(Modifier.height(36.dp))

        Row(
            modifier = Modifier.fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = stringResource(R.string.notif_settings_toggle),
                color = RunaColors.Heading,
                fontSize = 17.sp,
                modifier = Modifier.weight(1f),
            )
            Switch(
                checked = state.enabled,
                onCheckedChange = { on -> if (on) enableReminder() else viewModel.onToggle(false) },
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

        Spacer(Modifier.height(40.dp))
        Text(
            text = stringResource(R.string.notif_settings_time_caption),
            color = RunaColors.Subtle,
            fontSize = 14.sp,
            modifier = Modifier.fillMaxWidth(),
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(16.dp))
        // The big time display opens the free picker.
        Text(
            text = state.time.label,
            color = RunaColors.Heading,
            fontWeight = FontWeight.Light,
            fontSize = 72.sp,
            modifier = Modifier
                .fillMaxWidth()
                .clickable { showPicker = true },
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(24.dp))
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(12.dp, Alignment.CenterHorizontally),
        ) {
            state.presets.forEach { preset ->
                PresetChip(
                    label = preset.label,
                    selected = preset == state.time,
                    onClick = { viewModel.onSelectTime(preset) },
                )
            }
        }

        Spacer(Modifier.weight(1f))
        Text(
            text = stringResource(R.string.notif_settings_footer),
            color = RunaColors.Subtle,
            fontSize = 14.sp,
            modifier = Modifier.fillMaxWidth(),
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(40.dp))
    }

    if (showPicker) {
        TimePickerDialog(
            initial = state.time,
            onConfirm = { picked ->
                viewModel.onSelectTime(picked)
                showPicker = false
            },
            onDismiss = { showPicker = false },
        )
    }
}

@Composable
private fun PresetChip(label: String, selected: Boolean, onClick: () -> Unit) {
    val contentColor = if (selected) RunaColors.Accent else RunaColors.Subtle
    val borderColor = if (selected) RunaColors.Accent else RunaColors.Subtle.copy(alpha = 0.4f)
    Text(
        text = label,
        color = contentColor,
        fontSize = 16.sp,
        modifier = Modifier
            .border(1.dp, borderColor, RoundedCornerShape(50))
            .clickable(onClick = onClick)
            .padding(horizontal = 22.dp, vertical = 12.dp),
    )
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun TimePickerDialog(
    initial: ReminderTime,
    onConfirm: (ReminderTime) -> Unit,
    onDismiss: () -> Unit,
) {
    val pickerState = rememberTimePickerState(
        initialHour = initial.hour,
        initialMinute = initial.minute,
        is24Hour = true,
    )
    AlertDialog(
        onDismissRequest = onDismiss,
        containerColor = RunaColors.Surface,
        confirmButton = {
            TextButton(onClick = { onConfirm(ReminderTime.of(pickerState.hour, pickerState.minute)) }) {
                Text(stringResource(R.string.time_picker_confirm), color = RunaColors.Accent)
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text(stringResource(R.string.time_picker_cancel), color = RunaColors.Subtle)
            }
        },
        text = { TimePicker(state = pickerState) },
    )
}
