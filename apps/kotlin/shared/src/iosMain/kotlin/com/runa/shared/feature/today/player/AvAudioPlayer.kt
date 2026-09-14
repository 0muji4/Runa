package com.runa.shared.feature.today.player

import kotlinx.cinterop.ExperimentalForeignApi
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import platform.AVFoundation.AVPlayer
import platform.AVFoundation.AVPlayerItem
import platform.AVFoundation.AVPlayerItemDidPlayToEndTimeNotification
import platform.AVFoundation.AVPlayerTimeControlStatusPlaying
import platform.AVFoundation.timeControlStatus
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
import platform.Foundation.NSNotificationCenter
import platform.Foundation.NSURL

/**
 * AVPlayer-backed [AudioPlayer]. Nothing here may download or cache the preview (see [AudioPlayer]).
 * The periodic observer stops at the end of the item, so the end is caught via AVPlayerItemDidPlayToEndTimeNotification.
 */
@OptIn(ExperimentalForeignApi::class)
class AvAudioPlayer : AudioPlayer {

    private val player = AVPlayer()
    private var timeObserver: Any? = null
    private var endObserver: Any? = null
    private var ended = false

    private val _state = MutableStateFlow(PlaybackState())
    override val playbackState: StateFlow<PlaybackState> = _state.asStateFlow()

    override fun load(url: String) {
        val nsUrl = NSURL.URLWithString(url) ?: return
        val item = AVPlayerItem(uRL = nsUrl)
        player.replaceCurrentItemWithPlayerItem(item)
        ended = false
        addObserversIfNeeded()
        sync()
    }

    override fun play() {
        if (ended) {
            player.seekToTime(CMTimeMakeWithSeconds(0.0, PREFERRED_TIMESCALE))
            ended = false
        }
        player.play()
        sync()
    }

    override fun pause() {
        player.pause()
        sync()
    }

    override fun release() {
        timeObserver?.let { player.removeTimeObserver(it) }
        timeObserver = null
        endObserver?.let { NSNotificationCenter.defaultCenter.removeObserver(it) }
        endObserver = null
        player.pause()
    }

    private fun addObserversIfNeeded() {
        if (timeObserver == null) {
            val interval = CMTimeMakeWithSeconds(POSITION_POLL_SECONDS, PREFERRED_TIMESCALE)
            timeObserver = player.addPeriodicTimeObserverForInterval(interval, null) { _ ->
                sync()
            }
        }
        if (endObserver == null) {
            endObserver = NSNotificationCenter.defaultCenter.addObserverForName(
                name = AVPlayerItemDidPlayToEndTimeNotification,
                `object` = null,
                queue = null,
            ) { _ ->
                ended = true
                sync()
            }
        }
    }

    private fun sync() {
        val positionSeconds = CMTimeGetSeconds(player.currentTime())
        val durationSeconds = player.currentItem?.duration?.let { CMTimeGetSeconds(it) } ?: Double.NaN
        // The last periodic tick can land on the final frame before the end notification arrives.
        if (!durationSeconds.isNaN() && durationSeconds > 0 && positionSeconds >= durationSeconds - END_TOLERANCE_SECONDS) {
            ended = true
        }
        _state.value = PlaybackState(
            isPlaying = !ended && player.timeControlStatus == AVPlayerTimeControlStatusPlaying,
            positionMs = positionSeconds.toMillisOrZero(),
            durationMs = durationSeconds.toMillisOrZero(),
            isBuffering = false,
        )
    }

    private fun Double.toMillisOrZero(): Long =
        if (isNaN() || this < 0.0) 0L else (this * 1000).toLong()

    private companion object {
        const val POSITION_POLL_SECONDS = 0.5
        const val END_TOLERANCE_SECONDS = 0.25
        const val PREFERRED_TIMESCALE = 600
    }
}
