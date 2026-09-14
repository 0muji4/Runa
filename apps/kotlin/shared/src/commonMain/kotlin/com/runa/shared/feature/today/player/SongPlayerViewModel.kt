package com.runa.shared.feature.today.player

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.runa.shared.feature.today.SongRepository
import com.runa.shared.network.dto.SongDto
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import kotlinx.datetime.Clock

/**
 * Owns the playback intent (which song, play/pause) over the platform [AudioPlayer].
 * Starting a new song also records a play via [SongRepository].
 */
class SongPlayerViewModel(
    private val audioPlayer: AudioPlayer,
    private val songRepository: SongRepository,
) : ViewModel() {
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
        }.stateIn(viewModelScope, SharingStarted.Eagerly, PlayerUiState())

    /** Load and play [song]; a different song reloads the engine and records a play. */
    fun play(song: SongDto) {
        val isNewSong = currentSong.value?.id != song.id
        currentSong.value = song
        if (isNewSong) {
            audioPlayer.load(song.previewUrl)
            viewModelScope.launch { songRepository.markPlayed(song, Clock.System.now().toEpochMilliseconds()) }
        }
        audioPlayer.play()
    }

    fun togglePlayPause() {
        if (state.value.isPlaying) audioPlayer.pause() else audioPlayer.play()
    }

    fun pause() = audioPlayer.pause()
}
