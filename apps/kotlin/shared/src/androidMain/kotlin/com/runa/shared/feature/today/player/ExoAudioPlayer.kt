package com.runa.shared.feature.today.player

import android.content.Context
import android.os.Handler
import android.os.Looper
import androidx.media3.common.C
import androidx.media3.common.MediaItem
import androidx.media3.common.Player
import androidx.media3.exoplayer.ExoPlayer
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch

/**
 * ExoPlayer-backed [AudioPlayer] (Media3). ExoPlayer must be created and driven
 * on a thread with a Looper, so every engine call is posted to the main thread;
 * the shared view model may invoke play/pause/seek from any dispatcher. Position
 * is not pushed by ExoPlayer, so a coroutine ticker samples it while playing.
 */
class ExoAudioPlayer(context: Context) : AudioPlayer {

    private val mainHandler = Handler(Looper.getMainLooper())
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main)

    private val _state = MutableStateFlow(PlaybackState())
    override val playbackState: StateFlow<PlaybackState> = _state.asStateFlow()

    private var ticker: Job? = null

    private val player: ExoPlayer = ExoPlayer.Builder(context.applicationContext).build().apply {
        addListener(object : Player.Listener {
            override fun onIsPlayingChanged(isPlaying: Boolean) {
                syncState()
                manageTicker(isPlaying)
            }

            override fun onPlaybackStateChanged(playbackState: Int) {
                syncState()
            }
        })
    }

    override fun load(url: String) = onMain {
        player.setMediaItem(MediaItem.fromUri(url))
        player.prepare()
        syncState()
    }

    override fun play() = onMain { player.play() }

    override fun pause() = onMain { player.pause() }

    override fun seekTo(positionMs: Long) = onMain {
        player.seekTo(positionMs)
        syncState()
    }

    override fun release() = onMain {
        ticker?.cancel()
        player.release()
    }

    private fun manageTicker(isPlaying: Boolean) {
        ticker?.cancel()
        if (isPlaying) {
            ticker = scope.launch {
                while (isActive) {
                    syncState()
                    delay(POSITION_POLL_MS)
                }
            }
        }
    }

    private fun syncState() {
        val duration = player.duration
        _state.value = PlaybackState(
            isPlaying = player.isPlaying,
            positionMs = player.currentPosition.coerceAtLeast(0),
            durationMs = if (duration == C.TIME_UNSET) 0 else duration.coerceAtLeast(0),
            isBuffering = player.playbackState == Player.STATE_BUFFERING,
        )
    }

    /** Run [block] on the main thread (ExoPlayer's single-thread requirement). */
    private inline fun onMain(crossinline block: () -> Unit) {
        if (Looper.myLooper() == Looper.getMainLooper()) block() else mainHandler.post { block() }
    }

    private companion object {
        const val POSITION_POLL_MS = 500L
    }
}
