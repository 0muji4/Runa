package com.runa.shared.feature.today.player

import kotlinx.cinterop.ExperimentalForeignApi
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import platform.AVFoundation.AVPlayer
import platform.AVFoundation.AVPlayerItem
import platform.AVFoundation.addPeriodicTimeObserverForInterval
import platform.AVFoundation.currentItem
import platform.AVFoundation.currentTime
import platform.AVFoundation.duration
import platform.AVFoundation.pause
import platform.AVFoundation.play
import platform.AVFoundation.rate
import platform.AVFoundation.removeTimeObserver
import platform.AVFoundation.replaceCurrentItemWithPlayerItem
import platform.AVFoundation.seekToTime
import platform.CoreMedia.CMTimeGetSeconds
import platform.CoreMedia.CMTimeMakeWithSeconds
import platform.Foundation.NSURL

/**
 * AVPlayer-backed [AudioPlayer] (AVFoundation). AVPlayer is safe to drive from any
 * thread for these basic operations, so calls run directly. Position is sampled
 * via a periodic time observer, which republishes [PlaybackState] as it fires.
 */
@OptIn(ExperimentalForeignApi::class)
class AvAudioPlayer : AudioPlayer {

    private val player = AVPlayer()
    private var timeObserver: Any? = null

    private val _state = MutableStateFlow(PlaybackState())
    override val playbackState: StateFlow<PlaybackState> = _state.asStateFlow()

    override fun load(url: String) {
        val nsUrl = NSURL.URLWithString(url) ?: return
        player.replaceCurrentItemWithPlayerItem(AVPlayerItem(uRL = nsUrl))
        addObserverIfNeeded()
        sync()
    }

    override fun play() {
        player.play()
        sync()
    }

    override fun pause() {
        player.pause()
        sync()
    }

    override fun seekTo(positionMs: Long) {
        player.seekToTime(CMTimeMakeWithSeconds(positionMs / 1000.0, PREFERRED_TIMESCALE))
        sync()
    }

    override fun release() {
        timeObserver?.let { player.removeTimeObserver(it) }
        timeObserver = null
        player.pause()
    }

    /** Registers a ~0.5s time observer that mirrors playback progress into state. */
    private fun addObserverIfNeeded() {
        if (timeObserver != null) return
        val interval = CMTimeMakeWithSeconds(POSITION_POLL_SECONDS, PREFERRED_TIMESCALE)
        timeObserver = player.addPeriodicTimeObserverForInterval(interval, null) { _ ->
            sync()
        }
    }

    private fun sync() {
        val positionSeconds = CMTimeGetSeconds(player.currentTime())
        val durationSeconds = player.currentItem?.duration?.let { CMTimeGetSeconds(it) } ?: Double.NaN
        _state.value = PlaybackState(
            isPlaying = player.rate != 0f,
            positionMs = positionSeconds.toMillisOrZero(),
            durationMs = durationSeconds.toMillisOrZero(),
            isBuffering = false,
        )
    }

    private fun Double.toMillisOrZero(): Long =
        if (isNaN() || this < 0.0) 0L else (this * 1000).toLong()

    private companion object {
        const val POSITION_POLL_SECONDS = 0.5
        const val PREFERRED_TIMESCALE = 600
    }
}
