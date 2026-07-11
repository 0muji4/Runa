package com.runa.shared.feature.today

import app.cash.sqldelight.coroutines.asFlow
import app.cash.sqldelight.coroutines.mapToList
import com.runa.shared.db.RunaDatabase
import com.runa.shared.network.ApiClient
import com.runa.shared.network.dto.SongDto
import com.runa.shared.network.dto.SongsArchiveResponse
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.withContext
import kotlinx.datetime.Instant
import kotlin.random.Random

/** One local play-history record (also used as the archive list's "played" feed). */
data class SongHistoryEntry(
    val id: String,
    val songId: String,
    val title: String,
    val artist: String,
    val artworkUrl: String,
    val playedAtMs: Long,
)

/** Reads the song archive and records/observes local play history. */
interface SongRepository {
    /** The local play log, newest first, as a reactive stream. */
    fun observeSongHistory(limit: Long = 100): Flow<List<SongHistoryEntry>>

    /** One page of the backend song archive (newest first). */
    suspend fun getArchive(limit: Int?, cursor: String?): SongsArchiveResponse

    /** Record a play: write it to the local log AND best-effort POST to the server. */
    suspend fun markPlayed(song: SongDto, playedAtMs: Long)
}

/**
 * Default [SongRepository]. History is authoritative locally (SQLDelight) so it
 * survives offline; [markPlayed] also notifies the server but never fails the
 * local write if that call errors.
 */
class DefaultSongRepository(
    private val apiClient: ApiClient,
    private val database: RunaDatabase,
) : SongRepository {

    override fun observeSongHistory(limit: Long): Flow<List<SongHistoryEntry>> =
        database.todayQueries.selectHistory(limit)
            .asFlow()
            .mapToList(Dispatchers.Default)
            .map { rows ->
                rows.map {
                    SongHistoryEntry(
                        id = it.id, songId = it.song_id, title = it.title,
                        artist = it.artist, artworkUrl = it.artwork_url, playedAtMs = it.played_at,
                    )
                }
            }

    override suspend fun getArchive(limit: Int?, cursor: String?): SongsArchiveResponse =
        apiClient.getSongs(limit, cursor)

    override suspend fun markPlayed(song: SongDto, playedAtMs: Long) = withContext(Dispatchers.Default) {
        database.todayQueries.insertPlay(
            id = randomId(),
            song_id = song.id,
            title = song.title,
            artist = song.artist,
            artwork_url = song.artworkUrl,
            played_at = playedAtMs,
        )
        // Best-effort server notification; the local record already succeeded.
        try {
            apiClient.markSongPlayed(song.id, Instant.fromEpochMilliseconds(playedAtMs).toString())
        } catch (_: Exception) {
            // Offline or server error: the play stays local; a later slice may sync it.
        }
    }
}

/** A random v4-style UUID string for local-only ids (no dependency needed). */
private fun randomId(): String {
    val hex = "0123456789abcdef"
    val sb = StringBuilder(36)
    for (i in 0 until 36) {
        sb.append(
            when (i) {
                8, 13, 18, 23 -> '-'
                14 -> '4'
                19 -> hex[(Random.nextInt(4) + 8)] // 8,9,a,b
                else -> hex[Random.nextInt(16)]
            }
        )
    }
    return sb.toString()
}
