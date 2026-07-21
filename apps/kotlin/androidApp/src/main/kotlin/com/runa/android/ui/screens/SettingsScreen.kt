package com.runa.android.ui.screens

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.background
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.runa.android.BuildConfig
import com.runa.android.R
import com.runa.android.ui.theme.RunaColors
import com.runa.shared.feature.lock.AppLockViewModel
import com.runa.shared.feature.notification.NotificationSettingsViewModel
import com.runa.shared.feature.settings.AppTheme
import com.runa.shared.feature.settings.SettingsViewModel
import org.koin.compose.koinInject

/**
 * 設定 トップ (19). A quiet list of entry points: theme, notification and
 * privacy-lock (the latter two are 導線 only for a later feature), then
 * account・データ, the LUNA+ card, and the app version. Sign-out lives on the
 * account screen (per the confirmed design), not here.
 */
@Composable
fun SettingsScreen(
    onBack: () -> Unit,
    onOpenTheme: () -> Unit,
    onOpenNotifications: () -> Unit,
    onOpenPrivacyLock: () -> Unit,
    onOpenAccount: () -> Unit,
    viewModel: SettingsViewModel = koinInject(),
    notificationViewModel: NotificationSettingsViewModel = koinInject(),
    appLockViewModel: AppLockViewModel = koinInject(),
) {
    val theme by viewModel.theme.collectAsState()
    val notification by notificationViewModel.state.collectAsState()
    val lockEnabled by appLockViewModel.lockEnabled.collectAsState()

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(RunaColors.Background)
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 28.dp),
    ) {
        Spacer(Modifier.height(24.dp))
        Text(
            text = stringResource(R.string.action_back),
            color = RunaColors.Subtle,
            fontSize = 15.sp,
            modifier = Modifier.clickable(onClick = onBack),
        )
        Spacer(Modifier.height(28.dp))
        Text(
            text = stringResource(R.string.settings_title),
            color = RunaColors.Heading,
            fontWeight = FontWeight.Medium,
            fontSize = 40.sp,
        )
        Spacer(Modifier.height(40.dp))

        SettingRow(
            glyph = "☾",
            label = stringResource(R.string.settings_row_theme),
            value = themeShortName(theme),
            onClick = onOpenTheme,
        )
        SettingDivider()
        SettingRow(
            glyph = "◷",
            label = stringResource(R.string.settings_row_notifications),
            value = if (notification.enabled) notification.time.label
            else stringResource(R.string.notif_value_off),
            onClick = onOpenNotifications,
        )
        SettingDivider()
        SettingRow(
            glyph = "⚿",
            label = stringResource(R.string.settings_row_privacy_lock),
            value = stringResource(
                if (lockEnabled) R.string.lock_value_on else R.string.lock_value_off,
            ),
            onClick = onOpenPrivacyLock,
        )
        SettingDivider()
        SettingRow(
            glyph = "◍",
            label = stringResource(R.string.settings_row_account),
            onClick = onOpenAccount,
        )

        Spacer(Modifier.height(48.dp))
        PremiumCard()

        Spacer(Modifier.height(40.dp))
        Text(
            text = stringResource(R.string.settings_version, BuildConfig.VERSION_NAME),
            color = RunaColors.Subtle,
            fontSize = 12.sp,
            modifier = Modifier.fillMaxWidth(),
            textAlign = androidx.compose.ui.text.style.TextAlign.Center,
        )
        Spacer(Modifier.height(32.dp))
    }
}

@Composable
private fun SettingRow(
    glyph: String,
    label: String,
    value: String? = null,
    enabled: Boolean = true,
    onClick: (() -> Unit)? = null,
) {
    val rowModifier = if (enabled && onClick != null) {
        Modifier.fillMaxWidth().clickable(onClick = onClick)
    } else {
        Modifier.fillMaxWidth()
    }
    val labelColor = if (enabled) RunaColors.Heading else RunaColors.Subtle
    Row(
        modifier = rowModifier.padding(vertical = 20.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(glyph, color = labelColor, fontSize = 18.sp, modifier = Modifier.width(36.dp))
        Text(label, color = labelColor, fontSize = 17.sp, modifier = Modifier.weight(1f))
        if (value != null) {
            Text(value, color = RunaColors.Subtle, fontSize = 14.sp)
            Spacer(Modifier.width(12.dp))
        }
        Text("›", color = RunaColors.Subtle, fontSize = 20.sp)
    }
}

@Composable
private fun SettingDivider() {
    Box(
        Modifier
            .fillMaxWidth()
            .height(1.dp)
            .background(RunaColors.Subtle.copy(alpha = 0.15f)),
    )
}

@Composable
private fun PremiumCard() {
    // LUNA+ 導線. The paywall is a separate, not-yet-built feature, so this is a
    // quiet static card for now.
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(RunaColors.Surface, RoundedCornerShape(20.dp))
            .padding(20.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Box(
            Modifier
                .width(56.dp)
                .height(56.dp)
                .background(RunaColors.SubAccent, RoundedCornerShape(28.dp)),
        )
        Column(Modifier.weight(1f)) {
            Text(
                stringResource(R.string.settings_premium_title),
                color = RunaColors.Heading,
                fontSize = 22.sp,
            )
            Spacer(Modifier.height(4.dp))
            Text(
                stringResource(R.string.settings_premium_subtitle),
                color = RunaColors.Subtle,
                fontSize = 13.sp,
            )
        }
        Text("›", color = RunaColors.Accent, fontSize = 20.sp)
    }
}

/** The short theme name shown as the テーマ row's trailing value. */
@Composable
private fun themeShortName(theme: AppTheme): String = when (theme) {
    AppTheme.DARK -> stringResource(R.string.theme_dark_name)
    AppTheme.LIGHT -> stringResource(R.string.theme_light_name)
    AppTheme.PINK -> stringResource(R.string.theme_pink_name)
}
