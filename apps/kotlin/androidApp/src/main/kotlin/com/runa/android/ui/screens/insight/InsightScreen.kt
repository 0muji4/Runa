package com.runa.android.ui.screens.insight

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.runa.android.R
import com.runa.android.ui.components.RunaEmptyView
import com.runa.android.ui.components.RunaErrorView
import com.runa.android.ui.components.RunaLoadingView
import com.runa.android.ui.components.RunaScreenHeader
import com.runa.android.ui.components.RunaSyncBanner
import com.runa.android.ui.theme.CormorantGaramond
import com.runa.android.ui.theme.RunaColors
import com.runa.android.ui.theme.ShipporiMincho
import com.runa.android.ui.theme.ZenKakuGothicNew
import com.runa.shared.core.state.UiState
import com.runa.shared.feature.diary.DiaryMood
import com.runa.shared.feature.insight.Insight
import com.runa.shared.feature.insight.InsightPeriodType
import com.runa.shared.feature.insight.InsightViewModel
import com.runa.shared.feature.insight.MoodCount
import com.runa.shared.feature.insight.MoonPhaseBucket
import org.koin.compose.koinInject

/**
 * 16 インサイト — "あなたへの、手紙". A quiet retrospective letter: the period label,
 * a 明朝 heading, the rule-based summary, then the moon-phase overlap histogram (the
 * hero, a lone pink peak) and a soft mood-dot line, closed by a still footnote card.
 * A minimal 週/月 toggle and ‹ › period nav sit above. Everything renders from the
 * local diary — no network. The empty period keeps the moon motif.
 */
@Composable
fun InsightScreen(
    onBack: () -> Unit,
    viewModel: InsightViewModel = koinInject(),
) {
    val state by viewModel.state.collectAsState()
    val header by viewModel.header.collectAsState()

    Column(
        Modifier
            .fillMaxSize()
            .background(RunaColors.Background)
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 28.dp),
    ) {
        // The letter title is the screen title, so it shows over every state — not
        // just when a letter has been composed.
        RunaScreenHeader(
            title = stringResource(R.string.insight_letter_title),
            onBack = onBack,
        )

        // The period chrome always shows (over content and empty alike); the shared
        // state surfaces drive the letter body below it.
        PeriodBar(header.periodLabel, header.periodType, viewModel)
        when (val current = state) {
            is UiState.Content -> {
                LetterContent(current.data)
                RunaSyncBanner(current.sync)
                Spacer(Modifier.height(40.dp))
            }
            UiState.Empty -> RunaEmptyView(
                title = stringResource(R.string.insight_empty_title),
                body = stringResource(R.string.insight_empty_body),
                modifier = Modifier.fillMaxWidth().padding(top = 48.dp),
            )
            UiState.Loading -> RunaLoadingView(modifier = Modifier.fillMaxWidth().height(320.dp))
            is UiState.Failure -> RunaErrorView(
                onCta = viewModel::refresh,
                modifier = Modifier.fillMaxWidth().height(320.dp),
            )
        }
    }
}

@Composable
private fun PeriodBar(periodLabel: String, periodType: InsightPeriodType, viewModel: InsightViewModel) {
    // Quiet 週 | 月 toggle.
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(top = 12.dp),
        horizontalArrangement = Arrangement.Center,
    ) {
        ToggleChip(stringResource(R.string.insight_toggle_week), periodType == InsightPeriodType.Weekly) {
            viewModel.setPeriodType(InsightPeriodType.Weekly)
        }
        Spacer(Modifier.width(10.dp))
        ToggleChip(stringResource(R.string.insight_toggle_month), periodType == InsightPeriodType.Monthly) {
            viewModel.setPeriodType(InsightPeriodType.Monthly)
        }
    }
    // ‹ period label › — the label taps back to the current period.
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(top = 20.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.SpaceBetween,
    ) {
        Chevron("‹", viewModel::showPrevious)
        Text(
            text = periodLabel,
            style = TextStyle(fontFamily = ShipporiMincho, fontSize = 15.sp),
            color = RunaColors.Subtle,
            modifier = Modifier.clickable(onClick = viewModel::showCurrent),
        )
        Chevron("›", viewModel::showNext)
    }
}

@Composable
private fun LetterContent(insight: Insight) {
    Text(
        text = insight.narrative.body,
        style = TextStyle(fontFamily = ShipporiMincho, fontSize = 18.sp, lineHeight = 32.sp),
        color = RunaColors.Body,
        modifier = Modifier.padding(top = 28.dp),
    )

    Spacer(Modifier.height(40.dp))
    MoonOverlapChart(insight.summary.moonOverlap)

    Spacer(Modifier.height(36.dp))
    MoodDots(insight.summary.moodDistribution, insight.summary.unmoodedCount)

    insight.narrative.footnote?.let { footnote ->
        Spacer(Modifier.height(40.dp))
        Surface(color = RunaColors.Surface, shape = RoundedCornerShape(20.dp), modifier = Modifier.fillMaxWidth()) {
            Text(
                text = footnote,
                style = TextStyle(fontFamily = ShipporiMincho, fontSize = 17.sp, lineHeight = 30.sp),
                color = RunaColors.Body,
                modifier = Modifier.padding(horizontal = 26.dp, vertical = 26.dp),
            )
        }
    }
}

/**
 * The hero histogram: entries bucketed across the lunar cycle (新月 → 満月 → 新月).
 * The busiest phase glows moonlight-pink; the rest stay muted — a thing to gaze at,
 * not read.
 */
@Composable
private fun MoonOverlapChart(buckets: List<MoonPhaseBucket>) {
    val maxCount = buckets.maxOfOrNull { it.count } ?: 0
    val peakIndex = if (maxCount > 0) buckets.indexOfFirst { it.count == maxCount } else -1

    Column(Modifier.fillMaxWidth()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .height(120.dp),
            verticalAlignment = Alignment.Bottom,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            buckets.forEachIndexed { index, bucket ->
                val fraction = if (maxCount > 0) bucket.count.toFloat() / maxCount else 0f
                val isPeak = index == peakIndex
                Box(
                    modifier = Modifier
                        .weight(1f)
                        .fillMaxHeight(fraction.coerceIn(0.06f, 1f))
                        .clip(RoundedCornerShape(6.dp))
                        .background(
                            if (isPeak) RunaColors.Accent else RunaColors.Subtle.copy(alpha = 0.28f),
                        ),
                )
            }
        }
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 10.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            MoonEndLabel(stringResource(R.string.insight_moon_new))
            MoonEndLabel(stringResource(R.string.insight_moon_full))
            MoonEndLabel(stringResource(R.string.insight_moon_new))
        }
    }
}

@Composable
private fun MoonEndLabel(text: String) {
    Text(
        text = text,
        style = TextStyle(fontFamily = ZenKakuGothicNew, fontSize = 12.sp),
        color = RunaColors.Subtle,
    )
}

/** The soft mood line: a few dots per recorded mood, and a quiet note for the unmarked nights. */
@Composable
private fun MoodDots(distribution: List<MoodCount>, unmoodedCount: Int) {
    val present = distribution.filter { it.count > 0 }
    Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        present.forEach { moodCount ->
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    text = moodCount.mood.labelJa,
                    style = TextStyle(fontFamily = ShipporiMincho, fontSize = 14.sp),
                    color = RunaColors.Body,
                    modifier = Modifier.width(64.dp),
                )
                Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    repeat(moodCount.count.coerceAtMost(12)) {
                        Box(
                            Modifier
                                .size(7.dp)
                                .background(RunaColors.SubAccent, CircleShape),
                        )
                    }
                }
            }
        }
        if (unmoodedCount > 0) {
            Text(
                text = stringResource(R.string.insight_unmooded, unmoodedCount),
                style = TextStyle(fontFamily = ShipporiMincho, fontSize = 13.sp),
                color = RunaColors.Subtle,
            )
        }
    }
}

@Composable
private fun ToggleChip(label: String, selected: Boolean, onClick: () -> Unit) {
    Box(
        modifier = Modifier
            .clip(RoundedCornerShape(16.dp))
            .then(
                if (selected) {
                    Modifier.border(1.dp, RunaColors.Accent.copy(alpha = 0.7f), RoundedCornerShape(16.dp))
                } else {
                    Modifier
                },
            )
            .clickable(onClick = onClick)
            .padding(horizontal = 20.dp, vertical = 7.dp),
    ) {
        Text(
            text = label,
            style = TextStyle(fontFamily = ShipporiMincho, fontSize = 15.sp),
            color = if (selected) RunaColors.Accent else RunaColors.Subtle,
        )
    }
}

@Composable
private fun Chevron(glyph: String, onClick: () -> Unit) {
    Text(
        text = glyph,
        style = TextStyle(fontFamily = CormorantGaramond, fontSize = 34.sp),
        color = RunaColors.Subtle,
        modifier = Modifier
            .clickable(onClick = onClick)
            .padding(horizontal = 12.dp, vertical = 6.dp),
    )
}
