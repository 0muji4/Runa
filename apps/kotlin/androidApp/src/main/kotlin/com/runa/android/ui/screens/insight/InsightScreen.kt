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
import androidx.compose.material3.MaterialTheme
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
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.runa.android.R
import com.runa.android.ui.components.NewMoonEmblem
import com.runa.android.ui.theme.CormorantGaramond
import com.runa.android.ui.theme.RunaColors
import com.runa.android.ui.theme.ShipporiMincho
import com.runa.android.ui.theme.ZenKakuGothicNew
import com.runa.shared.feature.diary.DiaryMood
import com.runa.shared.feature.insight.Insight
import com.runa.shared.feature.insight.InsightBanner
import com.runa.shared.feature.insight.InsightPeriodType
import com.runa.shared.feature.insight.InsightUiState
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

    Column(
        Modifier
            .fillMaxSize()
            .background(RunaColors.Background)
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 28.dp),
    ) {
        Text(
            text = "‹ ${stringResource(R.string.action_back)}",
            style = MaterialLabel,
            color = RunaColors.Subtle,
            modifier = Modifier
                .padding(top = 14.dp)
                .clickable(onClick = onBack)
                .padding(vertical = 6.dp, horizontal = 4.dp),
        )

        when (val current = state) {
            is InsightUiState.Content -> {
                PeriodBar(current.periodLabel, current.periodType, viewModel)
                LetterContent(current.insight)
                Banner(current.banner)
                Spacer(Modifier.height(40.dp))
            }
            is InsightUiState.Empty -> {
                PeriodBar(current.periodLabel, current.periodType, viewModel)
                EmptyLetter()
                Banner(current.banner)
            }
            InsightUiState.Loading -> Box(Modifier.fillMaxWidth().height(240.dp))
            is InsightUiState.Error -> ErrorLetter()
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
        text = stringResource(R.string.insight_letter_title),
        style = TextStyle(fontFamily = ShipporiMincho, fontSize = 32.sp, lineHeight = 44.sp),
        color = RunaColors.Heading,
        modifier = Modifier.padding(top = 16.dp),
    )
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
private fun EmptyLetter() {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(top = 64.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        NewMoonEmblem(diameter = 116.dp)
        Spacer(Modifier.height(28.dp))
        Text(
            text = stringResource(R.string.insight_empty_title),
            style = TextStyle(fontFamily = ShipporiMincho, fontSize = 22.sp, lineHeight = 34.sp),
            color = RunaColors.Heading,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(14.dp))
        Text(
            text = stringResource(R.string.insight_empty_body),
            style = MaterialTheme.typography.bodyMedium,
            color = RunaColors.Subtle,
            textAlign = TextAlign.Center,
        )
    }
}

@Composable
private fun ErrorLetter() {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(top = 80.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(
            text = stringResource(R.string.diary_banner_error),
            style = MaterialTheme.typography.bodyMedium,
            color = RunaColors.Subtle,
            textAlign = TextAlign.Center,
        )
    }
}

@Composable
private fun Banner(banner: InsightBanner) {
    val text = when (banner) {
        InsightBanner.Offline -> stringResource(R.string.diary_banner_offline)
        InsightBanner.Error -> stringResource(R.string.diary_banner_error)
        else -> return // None / Syncing stay silent
    }
    Text(
        text = text,
        style = MaterialLabel,
        color = RunaColors.Subtle,
        textAlign = TextAlign.Center,
        modifier = Modifier
            .fillMaxWidth()
            .padding(top = 24.dp),
    )
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

private val MaterialLabel = TextStyle(fontFamily = ZenKakuGothicNew, fontSize = 13.sp)
