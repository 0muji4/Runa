package com.runa.android.ui.screens

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
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
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
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
import com.runa.android.ui.theme.RunaColors
import com.runa.android.ui.theme.runaColorsFor
import com.runa.shared.feature.settings.AppTheme
import com.runa.shared.feature.settings.ThemeViewModel
import org.koin.compose.koinInject

/**
 * テーマ (20). The three app themes as selectable cards; the active one is bordered
 * with a filled radio. Selecting one calls the shared [ThemeViewModel], which
 * persists it and re-emits — so the whole app (this screen included) recolors
 * immediately, giving the live preview the design calls for.
 */
@Composable
fun ThemeScreen(
    onBack: () -> Unit,
    viewModel: ThemeViewModel = koinInject(),
) {
    val selected by viewModel.theme.collectAsState()

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
            text = stringResource(R.string.theme_eyebrow),
            color = RunaColors.Subtle,
            fontSize = 13.sp,
            letterSpacing = 3.sp,
        )
        Spacer(Modifier.height(8.dp))
        Text(
            text = stringResource(R.string.theme_title),
            color = RunaColors.Heading,
            fontWeight = FontWeight.Medium,
            fontSize = 40.sp,
        )
        Spacer(Modifier.height(32.dp))

        ThemeOptions.forEach { option ->
            ThemeCard(
                option = option,
                selected = option.theme == selected,
                onSelect = { viewModel.select(option.theme) },
            )
            Spacer(Modifier.height(16.dp))
        }

        Spacer(Modifier.weight(1f))
        Text(
            text = stringResource(R.string.theme_footer),
            color = RunaColors.Subtle,
            fontSize = 14.sp,
            modifier = Modifier.fillMaxWidth(),
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(32.dp))
    }
}

private data class ThemeOption(val theme: AppTheme, val nameRes: Int, val descRes: Int)

private val ThemeOptions = listOf(
    ThemeOption(AppTheme.DARK, R.string.theme_dark_name, R.string.theme_dark_desc),
    ThemeOption(AppTheme.LIGHT, R.string.theme_light_name, R.string.theme_light_desc),
    ThemeOption(AppTheme.PINK, R.string.theme_pink_name, R.string.theme_pink_desc),
)

@Composable
private fun ThemeCard(
    option: ThemeOption,
    selected: Boolean,
    onSelect: () -> Unit,
) {
    val border = if (selected) BorderStroke(1.dp, RunaColors.Accent) else null
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .then(if (border != null) Modifier.border(border, RoundedCornerShape(20.dp)) else Modifier)
            .background(RunaColors.Surface, RoundedCornerShape(20.dp))
            .clickable(onClick = onSelect)
            .padding(18.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        ThemeSwatch(option.theme)
        Column(Modifier.weight(1f)) {
            Text(stringResource(option.nameRes), color = RunaColors.Heading, fontSize = 20.sp)
            Spacer(Modifier.height(4.dp))
            Text(stringResource(option.descRes), color = RunaColors.Subtle, fontSize = 13.sp)
        }
        RadioDot(selected)
    }
}

/** A miniature preview of a theme: its background with an accent + sub-accent dot. */
@Composable
private fun ThemeSwatch(theme: AppTheme) {
    val colors = runaColorsFor(theme)
    Box(
        modifier = Modifier
            .size(56.dp)
            .background(colors.Background, RoundedCornerShape(14.dp)),
    ) {
        Box(
            Modifier
                .padding(start = 12.dp, top = 12.dp)
                .size(14.dp)
                .background(colors.SubAccent, CircleShape),
        )
        Box(
            Modifier
                .align(Alignment.BottomEnd)
                .padding(end = 12.dp, bottom = 12.dp)
                .size(10.dp)
                .background(colors.Accent, CircleShape),
        )
    }
}

@Composable
private fun RadioDot(selected: Boolean) {
    Box(
        modifier = Modifier
            .size(24.dp)
            .border(1.dp, if (selected) RunaColors.Accent else RunaColors.Subtle, CircleShape),
        contentAlignment = Alignment.Center,
    ) {
        if (selected) {
            Box(
                Modifier
                    .size(12.dp)
                    .background(RunaColors.Accent, CircleShape),
            )
        }
    }
}
