package com.runa.shared.feature.today.player

import com.runa.shared.network.dto.SongDto

/**
 * UI state for the song player. [song] is the current target; the rest mirror the
 * platform engine's [PlaybackState]. A null [song] means nothing is loaded yet.
 */
data class PlayerUiState(
    val song: SongDto? = null,
    val isPlaying: Boolean = false,
    val positionMs: Long = 0,
    val durationMs: Long = 0,
    val isBuffering: Boolean = false,
)
