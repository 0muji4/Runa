package com.runa.shared.feature.today.player

import com.runa.shared.feature.today.SongHistoryEntry
import com.runa.shared.feature.today.SongRepository
import com.runa.shared.network.dto.SongDto
import com.runa.shared.network.dto.SongsArchiveResponse
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.emptyFlow
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

/** In-memory AudioPlayer that records the calls the view model makes to it. */
private class FakeAudioPlayer : AudioPlayer {
    override val playbackState = MutableStateFlow(PlaybackState())
    val calls = mutableListOf<String>()
    var loadedUrl: String? = null

    override fun load(url: String) { loadedUrl = url; calls += "load" }
    override fun play() { calls += "play"; playbackState.value = playbackState.value.copy(isPlaying = true) }
    override fun pause() { calls += "pause"; playbackState.value = playbackState.value.copy(isPlaying = false) }
    override fun seekTo(positionMs: Long) { calls += "seek:$positionMs" }
    override fun release() { calls += "release" }
}

private class RecordingSongRepository : SongRepository {
    val played = mutableListOf<Pair<String, Long>>()
    override fun observeSongHistory(limit: Long): Flow<List<SongHistoryEntry>> = emptyFlow()
    override suspend fun getArchive(limit: Int?, cursor: String?) = SongsArchiveResponse()
    override suspend fun markPlayed(song: SongDto, playedAtMs: Long) { played += song.id to playedAtMs }
}

class SongPlayerViewModelTest {

    private val song = SongDto("s1", "2024-12-15", "夜想曲", "月詠", "https://x/a.jpg", "https://x/a.mp3")

    @Test
    fun playLoadsTheSongAndRecordsAPlay() = runTest(UnconfinedTestDispatcher()) {
        val audio = FakeAudioPlayer()
        val repo = RecordingSongRepository()
        val vm = SongPlayerViewModel(audio, repo, scope = backgroundScope)

        vm.play(song)
        advanceUntilIdle()

        assertEquals("https://x/a.mp3", audio.loadedUrl)
        assertTrue(audio.calls.contains("play"))
        assertEquals(listOf("s1"), repo.played.map { it.first })
        assertEquals(song, vm.state.value.song)
        assertTrue(vm.state.value.isPlaying)
    }

    @Test
    fun replayingSameSongDoesNotReloadOrDoubleRecord() = runTest(UnconfinedTestDispatcher()) {
        val audio = FakeAudioPlayer()
        val repo = RecordingSongRepository()
        val vm = SongPlayerViewModel(audio, repo, scope = backgroundScope)

        vm.play(song)
        advanceUntilIdle()
        vm.togglePlayPause() // pause
        vm.play(song)        // resume, same song
        advanceUntilIdle()

        assertEquals(1, audio.calls.count { it == "load" }, "reloaded on same song")
        assertEquals(1, repo.played.size, "recorded a duplicate play")
    }
}
