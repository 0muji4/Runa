package com.runa.android.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
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
import com.runa.android.ui.components.LocalReauthenticate
import com.runa.android.ui.components.RunaEmptyView
import com.runa.android.ui.components.RunaErrorView
import com.runa.android.ui.components.RunaLoadingView
import com.runa.android.ui.components.RunaOfflineView
import com.runa.android.ui.components.RunaScreenHeader
import com.runa.android.ui.theme.RunaColors
import com.runa.shared.core.state.AppError
import com.runa.shared.feature.today.SongArchiveViewModel
import com.runa.shared.feature.today.player.SongPlayerViewModel
import com.runa.shared.network.dto.SongDto
import org.koin.compose.koinInject

/**
 * 08 これまでの一曲. The song archive (newest first) plus the local play history.
 * Tapping a song plays it through the shared [SongPlayerViewModel] and returns to
 * the player (07), recording the play.
 */
@Composable
fun SongArchiveScreen(
    onPlayAndReturn: () -> Unit,
    onBack: () -> Unit,
    viewModel: SongArchiveViewModel = koinInject(),
    playerViewModel: SongPlayerViewModel = koinInject(),
) {
    val state by viewModel.state.collectAsState()

    Column(
        Modifier
            .fillMaxSize()
            .background(RunaColors.Background)
            .padding(horizontal = 24.dp),
    ) {
        RunaScreenHeader(title = stringResource(R.string.song_archive_title), onBack = onBack)

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            // Empty / initial-loading / load-failure all route through the shared
            // state surfaces (as a tall item so the local play history can still show
            // below). The list itself is offline-tolerant once a page has landed.
            if (state.songs.isEmpty()) {
                item {
                    val stateModifier = Modifier.fillMaxWidth().height(360.dp)
                    val reauthenticate = LocalReauthenticate.current
                    val error = state.error
                    when {
                        state.isLoading -> RunaLoadingView(modifier = stateModifier)
                        error is AppError.Offline -> RunaOfflineView(
                            onRetry = { viewModel.loadNextPage(reset = true) },
                            modifier = stateModifier,
                        )
                        // Session expired → re-authenticate (retrying would just re-401);
                        // matches RunaStateView's own Failure→Auth branch and iOS.
                        error is AppError.Auth -> RunaErrorView(
                            title = stringResource(R.string.state_auth_title),
                            body = stringResource(R.string.state_auth_body),
                            ctaLabel = stringResource(R.string.state_auth_cta),
                            onCta = reauthenticate,
                            modifier = stateModifier,
                        )
                        error != null -> RunaErrorView(
                            onCta = { viewModel.loadNextPage(reset = true) },
                            modifier = stateModifier,
                        )
                        else -> RunaEmptyView(
                            title = stringResource(R.string.song_archive_empty),
                            body = stringResource(R.string.song_archive_empty_body),
                            modifier = stateModifier,
                        )
                    }
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
