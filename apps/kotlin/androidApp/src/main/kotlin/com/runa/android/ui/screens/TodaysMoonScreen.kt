package com.runa.android.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.runa.android.R
import com.runa.android.ui.components.MoonPhaseDisc
import com.runa.android.ui.components.RunaScreenHeader
import com.runa.android.ui.theme.RunaColors
import com.runa.android.ui.theme.ShipporiMincho
import com.runa.shared.core.state.UiState
import com.runa.shared.feature.today.moon.moonIsWaxing
import com.runa.shared.feature.today.moon.moonPhaseNameJa
import com.runa.shared.feature.todaymoon.TodayMoon
import com.runa.shared.feature.todaymoon.TodayMoonViewModel
import org.koin.compose.koinInject

/** 今日の月: phase name, 月齢, date and the next principal phase, all computed on device. */
@Composable
fun TodaysMoonScreen(
    onBack: () -> Unit,
    viewModel: TodayMoonViewModel = koinInject(),
) {
    val state by viewModel.state.collectAsStateWithLifecycle()

    Column(
        Modifier
            .fillMaxSize()
            .background(RunaColors.Background)
            .padding(horizontal = 32.dp),
    ) {
        RunaScreenHeader(
            title = stringResource(R.string.todays_moon_label),
            onBack = onBack,
        )

        // Local computation settles to Content synchronously; other states never surface.
        (state as? UiState.Content<TodayMoon>)?.let { MoonContent(it.data) }
    }
}

@Composable
private fun MoonContent(moon: TodayMoon) {
    Column(
        modifier = Modifier.fillMaxSize(),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
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
