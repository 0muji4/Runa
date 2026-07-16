package com.runa.android.ui.screens

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
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.runa.android.R
import com.runa.android.ui.components.MoonPhaseDisc
import com.runa.android.ui.theme.RunaColors
import com.runa.android.ui.theme.ShipporiMincho
import com.runa.shared.feature.today.moon.moonIsWaxing
import com.runa.shared.feature.today.moon.moonPhaseNameJa
import com.runa.shared.feature.todaymoon.TodayMoon
import com.runa.shared.feature.todaymoon.TodayMoonUiState
import com.runa.shared.feature.todaymoon.TodayMoonViewModel
import org.koin.compose.koinInject

/**
 * 15 今日の月. A large, quiet moon under the "今日の月" label, its phase name, 月齢 and
 * date, a hushed 明朝 line for the phase, and a whisper of the next principal phase.
 * Reached from the home screen's moon. Fully offline — every value is computed on
 * device by the shared moon calculator.
 */
@Composable
fun TodaysMoonScreen(
    onBack: () -> Unit,
    viewModel: TodayMoonViewModel = koinInject(),
) {
    val state by viewModel.state.collectAsState()

    Box(
        Modifier
            .fillMaxSize()
            .background(RunaColors.Background),
    ) {
        Text(
            text = "‹ ${stringResource(R.string.action_back)}",
            style = MaterialTheme.typography.labelLarge,
            color = RunaColors.Subtle,
            modifier = Modifier
                .padding(top = 14.dp, start = 20.dp)
                .clickable(onClick = onBack)
                .padding(vertical = 6.dp, horizontal = 4.dp),
        )

        when (val current = state) {
            is TodayMoonUiState.Content -> MoonContent(current.moon)
            TodayMoonUiState.Loading -> Unit // resolves synchronously
        }
    }
}

@Composable
private fun MoonContent(moon: TodayMoon) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = 32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Text(
            text = stringResource(R.string.todays_moon_label),
            style = TextStyle(fontFamily = ShipporiMincho, fontSize = 15.sp),
            color = RunaColors.Subtle,
        )
        Spacer(Modifier.height(36.dp))

        MoonPhaseDisc(
            illumination = moon.illumination.toFloat(),
            waxing = moonIsWaxing(moon.phaseKey),
            diameter = 236.dp,
        )

        Spacer(Modifier.height(40.dp))
        Text(
            text = moonPhaseNameJa(moon.phaseKey),
            style = TextStyle(fontFamily = ShipporiMincho, fontSize = 34.sp, lineHeight = 44.sp),
            color = RunaColors.Heading,
        )
        Spacer(Modifier.height(10.dp))
        Text(
            text = stringResource(R.string.todays_moon_age, formatAge(moon.ageDays), moon.dateLabel),
            style = MaterialTheme.typography.bodyMedium,
            color = RunaColors.Subtle,
        )

        Spacer(Modifier.height(36.dp))
        Text(
            text = moon.phrase,
            style = TextStyle(fontFamily = ShipporiMincho, fontSize = 20.sp, lineHeight = 34.sp),
            color = RunaColors.Body,
            textAlign = TextAlign.Center,
        )
    }

    // The next principal phase, quietly pinned to the bottom.
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(bottom = 40.dp),
        verticalArrangement = Arrangement.Bottom,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(
            text = stringResource(
                R.string.todays_moon_next,
                moon.nextPhaseDateLabel,
                moonPhaseNameJa(moon.nextPhaseKey),
            ),
            style = MaterialTheme.typography.labelLarge,
            color = RunaColors.Subtle,
            modifier = Modifier.fillMaxWidth(),
            textAlign = TextAlign.Center,
        )
    }
}

private fun formatAge(ageDays: Double): String = "%.1f".format(ageDays)
