package com.runa.shared.feature.today.player

import kotlinx.coroutines.flow.StateFlow

/** Snapshot of the platform audio engine's state; [durationMs] is 0 until the media is prepared. */
data class PlaybackState(
    val isPlaying: Boolean = false,
    val positionMs: Long = 0,
    val durationMs: Long = 0,
    val isBuffering: Boolean = false,
)

/**
 * Platform audio engine seam (ExoPlayer on Android, AVPlayer on iOS). The media is Apple's
 * 30-second preview, allowed only as a stream: implementations must not add a disk cache, and no seek.
 */
interface AudioPlayer {
    val playbackState: StateFlow<PlaybackState>

    /** Prepare the given stream URL for playback (does not auto-start). */
    fun load(url: String)

    fun play()

    fun pause()

    /** Release engine resources; the player is unusable afterwards. */
    fun release()
}
