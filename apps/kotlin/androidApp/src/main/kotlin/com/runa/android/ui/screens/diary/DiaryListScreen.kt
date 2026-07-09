package com.runa.android.ui.screens.diary

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExtendedFloatingActionButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.runa.android.R
import com.runa.shared.feature.diary.DiaryEntry
import com.runa.shared.feature.diary.DiaryListState
import com.runa.shared.feature.diary.DiaryListViewModel
import com.runa.shared.feature.diary.SyncBanner
import org.koin.compose.koinInject

/**
 * Diary list (screen 09). A quiet vertical list of record cards over the Runa
 * background, with pull-to-refresh, a subtle sync banner, a moon-motif empty
 * state, and a still "綴る" entry point (FAB) into the editor.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DiaryListScreen(
    onOpenEntry: (String) -> Unit,
    onNewEntry: () -> Unit,
    viewModel: DiaryListViewModel = koinInject(),
) {
    val state by viewModel.state.collectAsState()
    val banner = state.banner()

    Scaffold(
        containerColor = MaterialTheme.colorScheme.background,
        topBar = { TopAppBar(title = { Text(stringResource(R.string.tab_diary)) }) },
        floatingActionButton = {
            ExtendedFloatingActionButton(
                onClick = onNewEntry,
                containerColor = MaterialTheme.colorScheme.surface,
                contentColor = MaterialTheme.colorScheme.primary,
                text = { Text(stringResource(R.string.diary_new)) },
                icon = { Text("☾") },
            )
        },
    ) { innerPadding ->
        PullToRefreshBox(
            isRefreshing = banner == SyncBanner.Syncing,
            onRefresh = viewModel::refresh,
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding),
        ) {
            Column(Modifier.fillMaxSize()) {
                SyncBannerLine(banner)
                when (val current = state) {
                    is DiaryListState.Content -> EntryList(current.entries, onOpenEntry)
                    is DiaryListState.Empty -> EmptyState()
                    DiaryListState.Loading -> Box(Modifier.fillMaxSize()) // brief; DB emits quickly
                }
            }
        }
    }
}

@Composable
private fun EntryList(entries: List<DiaryEntry>, onOpenEntry: (String) -> Unit) {
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(horizontal = 20.dp, vertical = 16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        items(entries, key = { it.clientId }) { entry ->
            DiaryCard(entry, onClick = { onOpenEntry(entry.clientId) })
        }
    }
}

@Composable
private fun DiaryCard(entry: DiaryEntry, onClick: () -> Unit) {
    Surface(
        color = MaterialTheme.colorScheme.surface,
        shape = MaterialTheme.shapes.medium,
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick),
    ) {
        Column(
            modifier = Modifier.padding(horizontal = 20.dp, vertical = 18.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Text(
                text = entry.bodyText,
                style = MaterialTheme.typography.bodyLarge,
                color = MaterialTheme.colorScheme.onSurface,
                maxLines = 3,
                overflow = TextOverflow.Ellipsis,
            )
            Text(
                text = formatDiaryDate(entry.createdAtEpochMs),
                style = MaterialTheme.typography.labelLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

@Composable
private fun EmptyState() {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = 40.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp, Alignment.CenterVertically),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text("☾", style = MaterialTheme.typography.displayLarge, color = MaterialTheme.colorScheme.onSurfaceVariant)
        Text(
            text = stringResource(R.string.diary_empty_title),
            style = MaterialTheme.typography.titleLarge,
            color = MaterialTheme.colorScheme.onBackground,
            textAlign = TextAlign.Center,
        )
        Text(
            text = stringResource(R.string.diary_empty_body),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            textAlign = TextAlign.Center,
        )
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

    Surface(color = MaterialTheme.colorScheme.surface, modifier = Modifier.fillMaxWidth()) {
        Text(
            text = text,
            style = MaterialTheme.typography.labelLarge,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            textAlign = TextAlign.Center,
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 20.dp, vertical = 10.dp),
        )
    }
}

private fun DiaryListState.banner(): SyncBanner = when (this) {
    is DiaryListState.Content -> banner
    is DiaryListState.Empty -> banner
    DiaryListState.Loading -> SyncBanner.None
}
