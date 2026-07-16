package com.runa.android.ui.screens.calendar

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
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
import com.runa.android.ui.theme.CormorantGaramond
import com.runa.android.ui.theme.RunaColors
import com.runa.android.ui.theme.ShipporiMincho
import com.runa.android.ui.theme.ZenKakuGothicNew
import com.runa.shared.feature.calendar.CalendarBanner
import com.runa.shared.feature.calendar.CalendarDay
import com.runa.shared.feature.calendar.CalendarUiState
import com.runa.shared.feature.calendar.CalendarViewModel
import com.runa.shared.feature.today.moon.moonIsWaxing
import org.koin.compose.koinInject

private val WEEKDAYS = listOf("日", "月", "火", "水", "木", "金", "土")

/**
 * 12 ふりかえりカレンダー. A quiet month grid: a serif "YYYY M月" header flanked by
 * prev/next chevrons (the title taps back to today), the 日〜土 week row, then the
 * days. Per the confirmed design the moon is drawn ONLY on days that have a record
 * ("記録のある日に、月あかり"); today wears a soft moonlight-pink rounded outline.
 * Tapping a day opens its records, or the backdated writer when it has none.
 */
@Composable
fun CalendarScreen(
    onOpenDayRecords: (isoDate: String) -> Unit,
    onWriteOnDay: (isoDate: String) -> Unit,
    onBack: () -> Unit,
    viewModel: CalendarViewModel = koinInject(),
) {
    val state by viewModel.state.collectAsState()

    Column(
        Modifier
            .fillMaxSize()
            .background(RunaColors.Background)
            .padding(horizontal = 20.dp),
    ) {
        // Slim back affordance (this screen is pushed from the diary tab).
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
            is CalendarUiState.Content -> MonthContent(
                content = current,
                onPrev = viewModel::showPreviousMonth,
                onNext = viewModel::showNextMonth,
                onToday = viewModel::showToday,
                onDayClick = { day ->
                    val iso = isoDateOf(day)
                    if (day.entryCount > 0) onOpenDayRecords(iso) else onWriteOnDay(iso)
                },
            )
            CalendarUiState.Loading -> Box(Modifier.fillMaxSize()) // brief; DB emits quickly
        }
    }
}

@Composable
private fun ColumnScope.MonthContent(
    content: CalendarUiState.Content,
    onPrev: () -> Unit,
    onNext: () -> Unit,
    onToday: () -> Unit,
    onDayClick: (CalendarDay) -> Unit,
) {
    Spacer(Modifier.height(8.dp))
    MonthHeader(content.year, content.month, onPrev, onNext, onToday)
    Spacer(Modifier.height(24.dp))
    WeekdayRow()
    Spacer(Modifier.height(8.dp))
    MonthGrid(content.firstDayOfWeek, content.days, onDayClick)

    Spacer(Modifier.weight(1f))
    CalendarLegend(content.banner)
    Spacer(Modifier.height(28.dp))
}

@Composable
private fun MonthHeader(year: Int, month: Int, onPrev: () -> Unit, onNext: () -> Unit, onToday: () -> Unit) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.SpaceBetween,
    ) {
        Chevron("‹", onPrev)
        Row(
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(14.dp),
            modifier = Modifier.clickable(onClick = onToday),
        ) {
            Text(
                text = year.toString(),
                style = TextStyle(fontFamily = CormorantGaramond, fontSize = 32.sp),
                color = RunaColors.Heading,
            )
            Text(
                text = "${month}月",
                style = TextStyle(fontFamily = ShipporiMincho, fontSize = 30.sp),
                color = RunaColors.Heading,
            )
        }
        Chevron("›", onNext)
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

@Composable
private fun WeekdayRow() {
    Row(Modifier.fillMaxWidth()) {
        WEEKDAYS.forEach { label ->
            Text(
                text = label,
                style = TextStyle(fontFamily = ZenKakuGothicNew, fontSize = 12.sp),
                color = RunaColors.Subtle,
                textAlign = TextAlign.Center,
                modifier = Modifier.weight(1f),
            )
        }
    }
}

@Composable
private fun MonthGrid(firstDayOfWeek: Int, days: List<CalendarDay>, onDayClick: (CalendarDay) -> Unit) {
    val cells: List<CalendarDay?> = List(firstDayOfWeek) { null } + days
    val padded = cells + List((7 - cells.size % 7) % 7) { null }
    Column(Modifier.fillMaxWidth()) {
        padded.chunked(7).forEach { week ->
            Row(Modifier.fillMaxWidth()) {
                week.forEach { day ->
                    Box(Modifier.weight(1f), contentAlignment = Alignment.TopCenter) {
                        if (day != null) DayCell(day, onDayClick)
                    }
                }
            }
        }
    }
}

@Composable
private fun DayCell(day: CalendarDay, onDayClick: (CalendarDay) -> Unit) {
    val bright = day.isToday || day.entryCount > 0
    Column(
        modifier = Modifier
            .height(58.dp)
            .then(
                if (day.isToday) {
                    Modifier.border(1.dp, RunaColors.Accent.copy(alpha = 0.8f), RoundedCornerShape(14.dp))
                } else {
                    Modifier
                },
            )
            .clickable(onClick = { onDayClick(day) })
            .padding(top = 8.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        Text(
            text = day.day.toString(),
            style = TextStyle(fontFamily = ZenKakuGothicNew, fontSize = 15.sp),
            color = if (bright) RunaColors.Heading else RunaColors.Subtle,
        )
        // The moon lights only on days that hold a record.
        if (day.entryCount > 0) {
            MoonPhaseDisc(
                illumination = day.illumination.toFloat(),
                waxing = moonIsWaxing(day.phaseKey),
                diameter = 18.dp,
            )
        }
    }
}

@Composable
private fun CalendarLegend(banner: CalendarBanner) {
    Column(Modifier.fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally) {
        bannerText(banner)?.let {
            Text(
                text = it,
                style = MaterialLabel,
                color = RunaColors.Subtle,
                modifier = Modifier.padding(bottom = 12.dp),
            )
        }
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            Box(
                Modifier
                    .size(8.dp)
                    .background(RunaColors.SubAccent, RoundedCornerShape(50)),
            )
            Text(
                text = stringResource(R.string.calendar_legend),
                style = TextStyle(fontFamily = ShipporiMincho, fontSize = 13.sp),
                color = RunaColors.Subtle,
            )
        }
    }
}

@Composable
private fun bannerText(banner: CalendarBanner): String? = when (banner) {
    CalendarBanner.Offline -> stringResource(R.string.diary_banner_offline)
    CalendarBanner.Error -> stringResource(R.string.diary_banner_error)
    else -> null // None / Syncing stay silent
}

private val MaterialLabel = TextStyle(fontFamily = ZenKakuGothicNew, fontSize = 13.sp)

private fun isoDateOf(day: CalendarDay): String =
    "%04d-%02d-%02d".format(day.year, day.month, day.day)
