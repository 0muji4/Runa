package com.runa.android.ui.screens

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Slider
import androidx.compose.material3.SliderDefaults
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
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import com.runa.android.R
import com.runa.android.ui.theme.RunaColors
import com.runa.shared.feature.today.HomeUiState
import com.runa.shared.feature.today.HomeViewModel
import com.runa.shared.feature.today.player.SongPlayerViewModel
import com.runa.shared.network.dto.SongDto
import org.koin.compose.koinInject

/**
 * 07 きょうの一曲. A spacious, refined player. Defaults to today's song (from the
 * shared [HomeViewModel]); once a song is playing (today's or one chosen from the
 * archive) it reflects the shared [SongPlayerViewModel]'s live state.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TodaysSongScreen(
    onOpenArchive: () -> Unit,
    homeViewModel: HomeViewModel = koinInject(),
    playerViewModel: SongPlayerViewModel = koinInject(),
) {
    val homeState by homeViewModel.state.collectAsState()
    val playerState by playerViewModel.state.collectAsState()

    val todaySong = (homeState as? HomeUiState.Content)?.today?.song
        ?: (homeState as? HomeUiState.Offline)?.today?.song
    val song = playerState.song ?: todaySong

    Scaffold(
        containerColor = RunaColors.Background,
        topBar = {
            TopAppBar(
                title = { Text(stringResource(R.string.tab_todays_song), color = RunaColors.Heading) },
                actions = {
                    TextButton(onClick = onOpenArchive) {
                        Text(stringResource(R.string.today_song_open_archive), color = RunaColors.Accent)
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = RunaColors.Background),
            )
        },
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .padding(horizontal = 32.dp, vertical = 24.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) {
            if (song == null) {
                Text(
                    text = stringResource(R.string.today_song_none),
                    style = MaterialTheme.typography.bodyLarge,
                    color = RunaColors.Subtle,
                    textAlign = TextAlign.Center,
                )
                return@Column
            }

            Player(
                song = song,
                isPlaying = playerState.isPlaying,
                positionMs = playerState.positionMs,
                durationMs = playerState.durationMs,
                onToggle = {
                    if (playerState.song == null) playerViewModel.play(song) else playerViewModel.togglePlayPause()
                },
                onSeek = { playerViewModel.seekTo(it) },
            )
        }
    }
}

@Composable
private fun Player(
    song: SongDto,
    isPlaying: Boolean,
    positionMs: Long,
    durationMs: Long,
    onToggle: () -> Unit,
    onSeek: (Long) -> Unit,
) {
    AsyncImage(
        model = song.artworkUrl,
        contentDescription = null,
        modifier = Modifier
            .fillMaxWidth()
            .aspectRatio(1f)
            .clip(RoundedCornerShape(16.dp)),
    )

    Spacer(Modifier.height(32.dp))
    Text(song.title, style = MaterialTheme.typography.headlineMedium, color = RunaColors.Heading, textAlign = TextAlign.Center)
    Text(song.artist, style = MaterialTheme.typography.bodyLarge, color = RunaColors.Subtle)

    Spacer(Modifier.height(24.dp))

    // Seek bar (shown once the media reports a duration).
    if (durationMs > 0) {
        Slider(
            value = positionMs.coerceIn(0, durationMs).toFloat(),
            onValueChange = { onSeek(it.toLong()) },
            valueRange = 0f..durationMs.toFloat(),
            colors = SliderDefaults.colors(
                thumbColor = RunaColors.Accent,
                activeTrackColor = RunaColors.Accent,
                inactiveTrackColor = RunaColors.Surface,
            ),
        )
        Box(Modifier.fillMaxWidth()) {
            Text(formatTime(positionMs), style = MaterialTheme.typography.bodyMedium, color = RunaColors.Subtle, modifier = Modifier.align(Alignment.CenterStart))
            Text(formatTime(durationMs), style = MaterialTheme.typography.bodyMedium, color = RunaColors.Subtle, modifier = Modifier.align(Alignment.CenterEnd))
        }
    }

    Spacer(Modifier.height(16.dp))
    Button(onClick = onToggle) {
        Text(stringResource(if (isPlaying) R.string.player_pause else R.string.player_play))
    }
}

private fun formatTime(ms: Long): String {
    val totalSeconds = (ms / 1000).coerceAtLeast(0)
    val minutes = totalSeconds / 60
    val seconds = totalSeconds % 60
    return "$minutes:${seconds.toString().padStart(2, '0')}"
}
