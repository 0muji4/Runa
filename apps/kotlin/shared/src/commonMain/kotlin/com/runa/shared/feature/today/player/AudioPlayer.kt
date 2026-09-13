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
 * INTENT (which song, play/pause) and observes [playbackState]; the actual
 * decoding/output is a platform implementation bound in
 * [com.runa.shared.platform.platformModule] — ExoPlayer (Media3) on Android,
 * AVPlayer (AVFoundation) on iOS.
 *
 * The media is Apple's 30-second preview, which Apple's terms allow only as a
 * stream: implementations must not add a disk cache (no Media3
 * CacheDataSource, no AVAssetDownloadTask) — see
 * docs/dd/todays-song-itunes-preview.md, Q3. There is deliberately no seek: the
 * preview is a promotional excerpt, not a player timeline (Q4).
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

    /** Release engine resources; the player is unusable afterwards. */
    fun release()
}
