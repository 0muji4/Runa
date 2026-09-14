package com.runa.android.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import coil.compose.AsyncImage
import com.runa.android.R
import com.runa.android.ui.components.AppleMusicBadge
import com.runa.android.ui.components.ITunesCourtesyLine
import com.runa.android.ui.components.RunaEmptyView
import com.runa.android.ui.components.RunaIcons
import com.runa.android.ui.components.RunaScreenHeader
import com.runa.android.ui.theme.RunaColors
import com.runa.shared.core.state.UiState
import com.runa.shared.feature.today.HomeViewModel
import com.runa.shared.feature.today.Today
import com.runa.shared.feature.today.player.SongPlayerViewModel
import com.runa.shared.network.dto.SongDto
import org.koin.compose.koinInject

/**
 * きょうの一曲: today's song from [HomeViewModel], or whatever [SongPlayerViewModel] is
 * playing. Apple's Promo Content terms: badge + attribution on the same screen, no seek.
 */
@Composable
fun TodaysSongScreen(
    onOpenArchive: () -> Unit,
    homeViewModel: HomeViewModel = koinInject(),
    playerViewModel: SongPlayerViewModel = koinInject(),
) {
    val homeState by homeViewModel.state.collectAsStateWithLifecycle()
    val playerState by playerViewModel.state.collectAsStateWithLifecycle()

    // Offline now rides along on Content (sync = Offline), so both cases are Content.
    val todaySong = (homeState as? UiState.Content<Today>)?.data?.song
    val song = playerState.song ?: todaySong

    Box(
        Modifier
            .fillMaxSize()
            .background(RunaColors.Background),
    ) {
        Column(Modifier.fillMaxSize()) {
            RunaScreenHeader(
                title = stringResource(R.string.tab_todays_song),
                modifier = Modifier.padding(horizontal = 24.dp),
            ) {
                Text(
                    text = stringResource(R.string.today_song_open_archive),
                    style = MaterialTheme.typography.labelMedium,
                    color = RunaColors.Accent,
                    modifier = Modifier
                        .clickable(onClick = onOpenArchive)
                        .padding(8.dp),
                )
            }

            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(horizontal = 32.dp, vertical = 8.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.Center,
            ) {
                if (song == null) {
                    RunaEmptyView(
                        title = stringResource(R.string.today_song_none),
                        body = stringResource(R.string.today_song_none_body),
                        modifier = Modifier.fillMaxSize(),
                    )
                    return@Column
                }

                SongIntroduction(
                    song = song,
                    isPlaying = playerState.isPlaying,
                    positionMs = playerState.positionMs,
                    durationMs = playerState.durationMs,
                    onToggle = {
                        if (playerState.song == null) playerViewModel.play(song) else playerViewModel.togglePlayPause()
                    },
                )
            }
        }
    }
}

@Composable
private fun SongIntroduction(
    song: SongDto,
    isPlaying: Boolean,
    positionMs: Long,
    durationMs: Long,
    onToggle: () -> Unit,
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

    Spacer(Modifier.height(28.dp))
    AppleMusicBadge(storeUrl = song.storeUrl, height = 48.dp)

    Spacer(Modifier.height(28.dp))
    Row(verticalAlignment = Alignment.CenterVertically) {
        IconButton(onClick = onToggle, modifier = Modifier.size(56.dp)) {
            Icon(
                imageVector = if (isPlaying) RunaIcons.Pause else RunaIcons.Play,
                contentDescription = stringResource(if (isPlaying) R.string.player_pause else R.string.player_play),
                tint = RunaColors.Accent,
                modifier = Modifier.size(40.dp),
            )
        }
        Spacer(Modifier.width(12.dp))
        Column(Modifier.weight(1f)) {
            Text(
                text = stringResource(R.string.song_preview_label),
                style = MaterialTheme.typography.labelMedium,
                color = RunaColors.Subtle,
            )
            Spacer(Modifier.height(8.dp))
            LinearProgressIndicator(
                progress = { if (durationMs > 0) positionMs.coerceIn(0, durationMs).toFloat() / durationMs else 0f },
                color = RunaColors.Accent,
                trackColor = RunaColors.Surface,
                gapSize = 0.dp,
                drawStopIndicator = {},
                modifier = Modifier.fillMaxWidth(),
            )
        }
    }

    Spacer(Modifier.height(20.dp))
    ITunesCourtesyLine()
}
