package com.runa.android.ui.screens

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.runa.android.R
import com.runa.android.ui.moon.MoonPresentation
import com.runa.android.ui.theme.RunaColors
import com.runa.shared.feature.today.HomeUiState
import com.runa.shared.feature.today.HomeViewModel
import com.runa.shared.feature.today.Today
import kotlin.math.roundToInt
import org.koin.compose.koinInject

/**
 * 06 Home. The quiet face of the app: a large 明朝 daily quote centered in
 * generous whitespace, with the day's moon phase + date above it. The quote and
 * moon still render when offline (the moon is always computed on-device).
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun HomeScreen(
    displayName: String,
    onSettingsClick: () -> Unit,
    onOpenTodaysMoon: () -> Unit,
    viewModel: HomeViewModel = koinInject(),
) {
    val state by viewModel.state.collectAsState()

    Scaffold(
        containerColor = RunaColors.Background,
        topBar = {
            TopAppBar(
                title = { Text("") },
                actions = {
                    TextButton(onClick = onSettingsClick) {
                        Text(stringResource(R.string.tab_settings), color = RunaColors.Subtle)
                    }
                },
            )
        },
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .padding(horizontal = 32.dp, vertical = 24.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) {
            when (val current = state) {
                is HomeUiState.Loading -> CircularProgressIndicator(color = RunaColors.Accent)
                is HomeUiState.Content -> HomeContent(current.today, offline = false, onOpenTodaysMoon)
                is HomeUiState.Offline -> HomeContent(current.today, offline = true, onOpenTodaysMoon)
                is HomeUiState.Error -> Text(
                    text = stringResource(R.string.home_error),
                    style = MaterialTheme.typography.bodyLarge,
                    color = RunaColors.Subtle,
                    textAlign = TextAlign.Center,
                )
            }
        }
    }
}

@Composable
private fun HomeContent(today: Today, offline: Boolean, onOpenTodaysMoon: () -> Unit) {
    val moon = today.moon

    // Moon glyph + date + phase name above the quote. Tapping the moon opens 今日の月.
    Text(
        text = MoonPresentation.glyph(moon.phaseKey),
        style = MaterialTheme.typography.displayLarge,
        modifier = Modifier.clickable(onClick = onOpenTodaysMoon),
    )
    Spacer(Modifier.height(8.dp))
    Text(
        text = "${today.dateLabel} · ${MoonPresentation.name(moon.phaseKey)}",
        style = MaterialTheme.typography.titleLarge,
        color = RunaColors.Heading,
    )
    Text(
        text = stringResource(R.string.moon_illumination, (moon.illumination * 100).roundToInt()),
        style = MaterialTheme.typography.bodyMedium,
        color = RunaColors.Subtle,
    )

    Spacer(Modifier.height(48.dp))

    // The daily quote — the emotional center of the screen.
    Text(
        text = today.quote?.bodyText ?: stringResource(R.string.home_no_quote),
        style = MaterialTheme.typography.headlineMedium,
        color = RunaColors.Heading,
        textAlign = TextAlign.Center,
    )

    if (offline) {
        Spacer(Modifier.height(32.dp))
        Text(
            text = stringResource(R.string.home_offline_hint),
            style = MaterialTheme.typography.bodyMedium,
            color = RunaColors.Subtle,
            textAlign = TextAlign.Center,
        )
    }
}
