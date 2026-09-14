package com.runa.android.ui.screens

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.runa.android.R
import com.runa.android.ui.components.MoonPhaseDisc
import com.runa.android.ui.components.RunaIcons
import com.runa.android.ui.components.RunaStateView
import com.runa.android.ui.theme.RunaColors
import com.runa.android.ui.theme.RunaHeader
import com.runa.shared.core.state.SyncPhase
import com.runa.shared.feature.today.HomeViewModel
import com.runa.shared.feature.today.Today
import com.runa.shared.feature.today.moon.moonIsWaxing
import com.runa.shared.feature.today.moon.moonPhaseNameJa
import org.koin.compose.koinInject

/**
 * Home: the daily quote with the day's moon + date at the top. No app bar — like every
 * tab, content starts at [RunaHeader.TopTab]. Quote and moon still render offline.
 */
@Composable
fun HomeScreen(
    displayName: String,
    onSettingsClick: () -> Unit,
    onOpenTodaysMoon: () -> Unit,
    viewModel: HomeViewModel = koinInject(),
) {
    val state by viewModel.state.collectAsStateWithLifecycle()

    Box(
        modifier = Modifier
            .fillMaxSize()
            .drawBehind {
                val glowCenter = Offset(size.width / 2f, size.height * 0.16f)
                val glowRadius = size.width * 0.62f
                drawCircle(
                    brush = Brush.radialGradient(
                        colors = listOf(Color(0x1AF7F2E4), Color.Transparent),
                        center = glowCenter,
                        radius = glowRadius,
                    ),
                    radius = glowRadius,
                    center = glowCenter,
                )
            },
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(horizontal = 32.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            RunaStateView(
                state = state,
                onRetry = viewModel::load,
                empty = {}, // the home always has content (the moon is computed locally)
                modifier = Modifier.fillMaxSize(),
            ) { today, sync ->
                HomeContent(today, offline = sync == SyncPhase.Offline, onOpenTodaysMoon)
            }
        }

        IconButton(
            onClick = onSettingsClick,
            modifier = Modifier
                .align(Alignment.TopEnd)
                .padding(top = RunaHeader.TopTab - 8.dp, end = 8.dp),
        ) {
            Icon(
                RunaIcons.Settings,
                contentDescription = stringResource(R.string.tab_settings),
                tint = RunaColors.Subtle,
            )
        }
    }
}

@Composable
private fun HomeContent(today: Today, offline: Boolean, onOpenTodaysMoon: () -> Unit) {
    val moon = today.moon

    Column(
        modifier = Modifier.fillMaxSize(),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Spacer(Modifier.height(RunaHeader.TopTab))

        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.clickable(onClick = onOpenTodaysMoon),
        ) {
            MoonPhaseDisc(
                illumination = moon.illumination.toFloat(),
                waxing = moonIsWaxing(moon.phaseKey),
                diameter = 30.dp,
            )
            Spacer(Modifier.width(12.dp))
            Text(
                text = today.dateLabel,
                style = MaterialTheme.typography.titleLarge,
                color = RunaColors.Heading,
            )
            Spacer(Modifier.width(10.dp))
            Text(
                text = moonPhaseNameJa(moon.phaseKey),
                style = MaterialTheme.typography.bodyMedium,
                color = RunaColors.Subtle,
            )
        }

        Spacer(Modifier.weight(1f))

        Text(
            text = today.quote?.bodyText ?: stringResource(R.string.home_no_quote),
            style = MaterialTheme.typography.headlineMedium,
            color = RunaColors.Heading,
            textAlign = TextAlign.Center,
        )
        if (today.quote != null) {
            Spacer(Modifier.height(24.dp))
            Text(
                text = "—  ${stringResource(R.string.home_quote_caption)}  —",
                style = MaterialTheme.typography.bodyMedium,
                color = RunaColors.Subtle,
                textAlign = TextAlign.Center,
            )
        }

        if (offline) {
            Spacer(Modifier.height(24.dp))
            Text(
                text = stringResource(R.string.home_offline_hint),
                style = MaterialTheme.typography.bodyMedium,
                color = RunaColors.Subtle,
                textAlign = TextAlign.Center,
            )
        }

        Spacer(Modifier.weight(1.15f))
    }
}
