package com.runa.shared.feature.today.player

import com.runa.shared.feature.today.SongRepository
import com.runa.shared.network.dto.SongDto
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import kotlinx.datetime.Clock

/**
 * Shared song-player view model. It owns the playback INTENT (which song, and
 * play/pause/seek) while the actual audio engine is the platform [AudioPlayer]
 * ([ExoPlayer]/[AVPlayer], injected). [state] merges the current song with the
 * engine's live [PlaybackState]. Starting a NEW song also records a play via
 * [SongRepository], so history accumulates whether the song is today's or an
 * archived one.
 */
class SongPlayerViewModel(
    private val audioPlayer: AudioPlayer,
    private val songRepository: SongRepository,
    private val scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Default),
) {
    private val currentSong = MutableStateFlow<SongDto?>(null)

    val state: StateFlow<PlayerUiState> =
        combine(currentSong, audioPlayer.playbackState) { song, playback ->
            PlayerUiState(
                song = song,
                isPlaying = playback.isPlaying,
                positionMs = playback.positionMs,
                durationMs = playback.durationMs,
                isBuffering = playback.isBuffering,
            )
        }.stateIn(scope, SharingStarted.Eagerly, PlayerUiState())

    /** Load and play [song]. A different song reloads the engine and records a play. */
    fun play(song: SongDto) {
        val isNewSong = currentSong.value?.id != song.id
        currentSong.value = song
        if (isNewSong) {
            audioPlayer.load(song.audioUrl)
            scope.launch { songRepository.markPlayed(song, Clock.System.now().toEpochMilliseconds()) }
        }
        audioPlayer.play()
    }

    /** Toggle between play and pause for the current song. */
    fun togglePlayPause() {
        if (state.value.isPlaying) audioPlayer.pause() else audioPlayer.play()
    }

    fun pause() = audioPlayer.pause()

    fun seekTo(positionMs: Long) = audioPlayer.seekTo(positionMs)
}
