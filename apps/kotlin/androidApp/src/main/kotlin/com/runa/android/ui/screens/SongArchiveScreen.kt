package com.runa.android.ui.screens

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import com.runa.android.R
import com.runa.android.ui.theme.RunaColors
import com.runa.shared.feature.today.SongArchiveViewModel
import com.runa.shared.feature.today.player.SongPlayerViewModel
import com.runa.shared.network.dto.SongDto
import org.koin.compose.koinInject

/**
 * 08 これまでの一曲. The song archive (newest first) plus the local play history.
 * Tapping a song plays it through the shared [SongPlayerViewModel] and returns to
 * the player (07), recording the play.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SongArchiveScreen(
    onPlayAndReturn: () -> Unit,
    onBack: () -> Unit,
    viewModel: SongArchiveViewModel = koinInject(),
    playerViewModel: SongPlayerViewModel = koinInject(),
) {
    val state by viewModel.state.collectAsState()

    Scaffold(
        containerColor = RunaColors.Background,
        topBar = {
            TopAppBar(
                title = { Text(stringResource(R.string.song_archive_title), color = RunaColors.Heading) },
                navigationIcon = {
                    TextButton(onClick = onBack) {
                        Text("‹", color = RunaColors.Body, style = MaterialTheme.typography.titleLarge)
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = RunaColors.Background),
            )
        },
    ) { innerPadding ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .padding(horizontal = 24.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            if (state.songs.isEmpty() && !state.isLoading) {
                item {
                    Text(
                        text = stringResource(R.string.song_archive_empty),
                        style = MaterialTheme.typography.bodyLarge,
                        color = RunaColors.Subtle,
                        modifier = Modifier.padding(vertical = 32.dp),
                    )
                }
            }

            items(state.songs, key = { it.id }) { song ->
                SongRow(song) {
                    playerViewModel.play(song)
                    onPlayAndReturn()
                }
            }

            if (state.canLoadMore) {
                item {
                    TextButton(onClick = { viewModel.loadNextPage() }, modifier = Modifier.fillMaxWidth()) {
                        Text(stringResource(R.string.song_archive_load_more), color = RunaColors.Accent)
                    }
                }
            }

            // Recent plays (local history).
            if (state.history.isNotEmpty()) {
                item {
                    Text(
                        text = stringResource(R.string.song_archive_history),
                        style = MaterialTheme.typography.titleLarge,
                        color = RunaColors.SubAccent,
                        modifier = Modifier.padding(top = 24.dp),
                    )
                }
                items(state.history, key = { it.id }) { entry ->
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        AsyncImage(
                            model = entry.artworkUrl,
                            contentDescription = null,
                            modifier = Modifier.size(40.dp).clip(RoundedCornerShape(6.dp)),
                        )
                        Spacer(Modifier.width(16.dp))
                        Text(
                            "${entry.title} · ${entry.artist}",
                            style = MaterialTheme.typography.bodyMedium,
                            color = RunaColors.Subtle,
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun SongRow(song: SongDto, onClick: () -> Unit) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        AsyncImage(
            model = song.artworkUrl,
            contentDescription = null,
            modifier = Modifier.size(56.dp).clip(RoundedCornerShape(8.dp)),
        )
        Spacer(Modifier.width(16.dp))
        Column {
            Text(song.title, style = MaterialTheme.typography.titleLarge, color = RunaColors.Heading)
            Text("${song.artist} · ${song.date}", style = MaterialTheme.typography.bodyMedium, color = RunaColors.Subtle)
        }
    }
}
