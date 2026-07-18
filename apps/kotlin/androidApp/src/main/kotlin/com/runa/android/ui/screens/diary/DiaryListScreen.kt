package com.runa.android.ui.screens.diary

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.runa.android.R
import com.runa.android.ui.components.MoonPhaseDisc
import com.runa.android.ui.components.NewMoonEmblem
import com.runa.android.ui.theme.RunaColors
import com.runa.android.ui.theme.ShipporiMincho
import com.runa.shared.feature.diary.DiaryEntry
import com.runa.shared.feature.diary.DiaryListState
import com.runa.shared.feature.diary.DiaryListViewModel
import com.runa.shared.feature.diary.SyncBanner
import org.koin.compose.koinInject

/**
 * Diary list (09) — "日々の記録". A large 明朝 heading over a quiet column of record
 * cards, each led by its day's moon phase. Pull-to-refresh, a whisper-quiet sync
 * line, a new-moon empty state with a "綴りはじめる" invitation, and a round
 * moonlight-pink FAB into the editor. No Material app bar — the header is the page.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DiaryListScreen(
    onOpenEntry: (String) -> Unit,
    onNewEntry: () -> Unit,
    onOpenCalendar: () -> Unit,
    onOpenInsight: () -> Unit,
    viewModel: DiaryListViewModel = koinInject(),
) {
    val state by viewModel.state.collectAsState()
    val banner = state.banner()

    Box(Modifier.fillMaxSize().background(RunaColors.Background)) {
        Column(Modifier.fillMaxSize()) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(start = 24.dp, end = 24.dp, top = 40.dp, bottom = 16.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = stringResource(R.string.diary_list_title),
                    style = TextStyle(fontFamily = ShipporiMincho, fontSize = 34.sp, lineHeight = 42.sp),
                    color = RunaColors.Heading,
                    modifier = Modifier.weight(1f),
                )
                // Quiet links into the retrospective calendar (12) and insight (16).
                Text(
                    text = stringResource(R.string.diary_open_insight),
                    style = MaterialTheme.typography.labelLarge,
                    color = RunaColors.Accent,
                    modifier = Modifier
                        .clickable(onClick = onOpenInsight)
                        .padding(8.dp),
                )
                Text(
                    text = stringResource(R.string.diary_open_calendar),
                    style = MaterialTheme.typography.labelLarge,
                    color = RunaColors.Accent,
                    modifier = Modifier
                        .clickable(onClick = onOpenCalendar)
                        .padding(8.dp),
                )
            }
            SyncBannerLine(banner)
            PullToRefreshBox(
                isRefreshing = banner == SyncBanner.Syncing,
                onRefresh = viewModel::refresh,
                modifier = Modifier.fillMaxSize(),
            ) {
                when (val current = state) {
                    is DiaryListState.Content -> EntryList(current.entries, onOpenEntry)
                    is DiaryListState.Empty -> EmptyState(onNewEntry)
                    DiaryListState.Loading -> Box(Modifier.fillMaxSize()) // brief; DB emits quickly
                }
            }
        }

        if (state is DiaryListState.Content) {
            PlusFab(
                onClick = onNewEntry,
                modifier = Modifier
                    .align(Alignment.BottomEnd)
                    .padding(28.dp),
            )
        }
    }
}

@Composable
private fun EntryList(entries: List<DiaryEntry>, onOpenEntry: (String) -> Unit) {
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(start = 20.dp, end = 20.dp, top = 8.dp, bottom = 120.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        items(entries, key = { it.clientId }) { entry ->
            DiaryCard(entry, onClick = { onOpenEntry(entry.clientId) })
        }
    }
}

@Composable
private fun DiaryCard(entry: DiaryEntry, onClick: () -> Unit) {
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
                    text = formatDiaryDate(entry.createdAtEpochMs),
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

@Composable
private fun EmptyState(onNewEntry: () -> Unit) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = 40.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        NewMoonEmblem(diameter = 116.dp)
        Spacer(Modifier.height(28.dp))
        Text(
            text = stringResource(R.string.diary_empty_title),
            style = MaterialTheme.typography.headlineMedium,
            color = RunaColors.Heading,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(14.dp))
        Text(
            text = stringResource(R.string.diary_empty_body),
            style = MaterialTheme.typography.bodyMedium,
            color = RunaColors.Subtle,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(36.dp))
        Box(
            modifier = Modifier
                .clickable(onClick = onNewEntry)
                .border(1.dp, RunaColors.Accent.copy(alpha = 0.7f), RoundedCornerShape(28.dp))
                .padding(horizontal = 32.dp, vertical = 14.dp),
        ) {
            Text(
                text = stringResource(R.string.diary_empty_cta),
                style = MaterialTheme.typography.bodyLarge,
                color = RunaColors.Accent,
            )
        }
    }
}

/** Round moonlight-pink FAB carrying a drawn "+" (no glyph). */
@Composable
private fun PlusFab(onClick: () -> Unit, modifier: Modifier = Modifier) {
    Box(
        modifier = modifier
            .size(64.dp)
            .background(RunaColors.Accent, CircleShape)
            .clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Canvas(Modifier.size(22.dp)) {
            val c = center
            val arm = size.minDimension * 0.34f
            drawLine(RunaColors.Background, androidx.compose.ui.geometry.Offset(c.x - arm, c.y), androidx.compose.ui.geometry.Offset(c.x + arm, c.y), strokeWidth = 4f, cap = StrokeCap.Round)
            drawLine(RunaColors.Background, androidx.compose.ui.geometry.Offset(c.x, c.y - arm), androidx.compose.ui.geometry.Offset(c.x, c.y + arm), strokeWidth = 4f, cap = StrokeCap.Round)
        }
    }
}

/** The quiet status line above the list. Renders nothing when there is no news. */
@Composable
private fun SyncBannerLine(banner: SyncBanner) {
    val text = when (banner) {
        SyncBanner.None, SyncBanner.Syncing -> null // syncing is shown by the pull indicator
        SyncBanner.Offline -> stringResource(R.string.diary_banner_offline)
        SyncBanner.Error -> stringResource(R.string.diary_banner_error)
    } ?: return

    Text(
        text = text,
        style = MaterialTheme.typography.labelLarge,
        color = RunaColors.Subtle,
        textAlign = TextAlign.Center,
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 24.dp, vertical = 8.dp),
    )
}

private fun DiaryListState.banner(): SyncBanner = when (this) {
    is DiaryListState.Content -> banner
    is DiaryListState.Empty -> banner
    DiaryListState.Loading -> SyncBanner.None
}
