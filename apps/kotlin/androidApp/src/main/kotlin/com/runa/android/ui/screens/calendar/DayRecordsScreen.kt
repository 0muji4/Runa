package com.runa.android.ui.screens.calendar

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.runa.android.R
import com.runa.android.ui.components.MoonPhaseDisc
import com.runa.android.ui.screens.diary.diaryMoonFor
import com.runa.android.ui.screens.diary.formatDiaryWeekday
import com.runa.android.ui.theme.RunaColors
import com.runa.android.ui.theme.ShipporiMincho
import com.runa.shared.feature.calendar.DayRecordsViewModel
import com.runa.shared.feature.diary.DiaryEntry
import org.koin.compose.getKoin
import org.koin.core.parameter.parametersOf

/**
 * The records of one calendar day, reached by tapping a day that has entries. A
 * quiet "M月d日" header over the day's record cards (each leads with the day's moon
 * and taps into the existing diary detail), plus a subtle "この日を綴る" invitation
 * that opens the backdated writer.
 */
@Composable
fun DayRecordsScreen(
    isoDate: String,
    onOpenEntry: (String) -> Unit,
    onWrite: () -> Unit,
    onBack: () -> Unit,
) {
    val koin = getKoin()
    val viewModel = remember(isoDate) { koin.get<DayRecordsViewModel> { parametersOf(isoDate) } }
    val entries by viewModel.state.collectAsState()

    Column(
        Modifier
            .fillMaxSize()
            .background(RunaColors.Background)
            .padding(horizontal = 20.dp),
    ) {
        Text(
            text = "‹ ${stringResource(R.string.action_back)}",
            style = MaterialTheme.typography.labelLarge,
            color = RunaColors.Subtle,
            modifier = Modifier
                .padding(top = 14.dp)
                .clickable(onClick = onBack)
                .padding(vertical = 6.dp, horizontal = 4.dp),
        )
        Text(
            text = viewModel.dateLabel,
            style = TextStyle(fontFamily = ShipporiMincho, fontSize = 30.sp, lineHeight = 38.sp),
            color = RunaColors.Heading,
            modifier = Modifier.padding(top = 12.dp, bottom = 20.dp),
        )

        LazyColumn(
            modifier = Modifier.weight(1f),
            contentPadding = PaddingValues(bottom = 24.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            items(entries, key = { it.clientId }) { entry ->
                DayRecordCard(entry, onClick = { onOpenEntry(entry.clientId) })
            }
        }

        Box(
            modifier = Modifier
                .padding(vertical = 20.dp)
                .clickable(onClick = onWrite)
                .border(1.dp, RunaColors.Accent.copy(alpha = 0.7f), RoundedCornerShape(28.dp))
                .padding(horizontal = 32.dp, vertical = 14.dp)
                .align(Alignment.CenterHorizontally),
        ) {
            Text(
                text = stringResource(R.string.calendar_write_on_day),
                style = MaterialTheme.typography.bodyLarge,
                color = RunaColors.Accent,
            )
        }
    }
}

@Composable
private fun DayRecordCard(entry: DiaryEntry, onClick: () -> Unit) {
    val moon = remember(entry.createdAtEpochMs) { diaryMoonFor(entry.createdAtEpochMs) }
    Surface(
        color = RunaColors.Surface,
        shape = RoundedCornerShape(22.dp),
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick),
    ) {
        Column(
            modifier = Modifier.padding(horizontal = 22.dp, vertical = 20.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                MoonPhaseDisc(illumination = moon.illumination, waxing = moon.waxing, diameter = 20.dp)
                Text(
                    text = formatDiaryWeekday(entry.createdAtEpochMs),
                    style = MaterialTheme.typography.labelLarge,
                    color = RunaColors.Body,
                )
                Text(
                    text = moon.name,
                    style = MaterialTheme.typography.labelLarge,
                    color = RunaColors.Subtle,
                )
            }
            Text(
                text = entry.bodyText,
                style = TextStyle(fontFamily = ShipporiMincho, fontSize = 16.sp, lineHeight = 27.sp),
                color = RunaColors.Body,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
        }
    }
}
