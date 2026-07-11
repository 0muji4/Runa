package com.runa.shared.feature.today.player

import kotlinx.coroutines.flow.StateFlow

/**
 * Snapshot of the platform audio engine's state, mirrored into the shared player
 * view model. Positions/durations are in milliseconds; durationMs is 0 until the
 * media is prepared.
 */
data class PlaybackState(
    val isPlaying: Boolean = false,
    val positionMs: Long = 0,
    val durationMs: Long = 0,
    val isBuffering: Boolean = false,
)

/**
 * The platform audio engine seam. The shared
 * [com.runa.shared.feature.today.player.SongPlayerViewModel] owns the playback
 * INTENT (which song, play/pause/seek) and observes [playbackState]; the actual
 * decoding/output is a platform implementation bound in
 * [com.runa.shared.platform.platformModule] — ExoPlayer (Media3) on Android,
 * AVPlayer (AVFoundation) on iOS.
 */
interface AudioPlayer {
    /** Current engine state, updated as playback progresses. */
    val playbackState: StateFlow<PlaybackState>

    /** Prepare the given stream URL for playback (does not auto-start). */
    fun load(url: String)

    /** Start (or resume) playback of the loaded media. */
    fun play()

    /** Pause playback, keeping the current position. */
    fun pause()

    /** Seek to an absolute position in milliseconds. */
    fun seekTo(positionMs: Long)

    /** Release engine resources; the player is unusable afterwards. */
    fun release()
}
