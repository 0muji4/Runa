package com.runa.shared.feature.today

import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.http.HttpHeaders
import io.ktor.http.HttpStatusCode
import io.ktor.http.headersOf
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.runTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

class SongRepositoryTest {

    @Test
    fun markPlayedWritesLocalHistoryAndPostsToServer() = runTest {
        val posted = mutableListOf<String>()
        val engine = MockEngine { request ->
            posted += request.url.encodedPath
            respond("", HttpStatusCode.NoContent)
        }
        val repo = DefaultSongRepository(mockApiClient(engine), inMemoryDatabase())

        repo.markPlayed(songSample, playedAtMs = 1_734_264_000_000L)

        val history = repo.observeSongHistory().first()
        assertEquals(1, history.size)
        assertEquals("s1", history[0].songId)
        assertEquals("夜想曲", history[0].title)
        assertEquals(1_734_264_000_000L, history[0].playedAtMs)
        assertTrue(posted.any { it.endsWith("/songs/s1/played") }, "server play not posted: $posted")
    }

    @Test
    fun markPlayedKeepsLocalHistoryEvenWhenServerFails() = runTest {
        val engine = MockEngine { respond("", HttpStatusCode.InternalServerError) }
        val repo = DefaultSongRepository(mockApiClient(engine), inMemoryDatabase())

        repo.markPlayed(songSample, playedAtMs = 42L)

        assertEquals(1, repo.observeSongHistory().first().size)
    }

    @Test
    fun getArchivePassesThroughBackendPage() = runTest {
        val archiveJson = """
            {"songs":[{"id":"s1","date":"2024-12-15","title":"夜想曲","artist":"月詠",
                       "artwork_url":"https://x/a.jpg","audio_url":"https://x/a.mp3"}],
             "next_cursor":"CURSOR2"}
        """.trimIndent()
        val engine = MockEngine {
            respond(archiveJson, HttpStatusCode.OK, headersOf(HttpHeaders.ContentType, "application/json"))
        }
        val repo = DefaultSongRepository(mockApiClient(engine), inMemoryDatabase())

        val page = repo.getArchive(limit = 20, cursor = null)

        assertEquals(1, page.songs.size)
        assertEquals("夜想曲", page.songs[0].title)
        assertEquals("CURSOR2", page.nextCursor)
    }
}
