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
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant
import kotlinx.datetime.LocalDate
import kotlinx.datetime.TimeZone
import kotlinx.datetime.toLocalDateTime
import kotlin.random.Random

data class SongHistoryEntry(
    val id: String,
    val songId: String,
    val title: String,
    val artist: String,
    val playedAtMs: Long,
)

/** Reads the song archive and records/observes local play history. */
interface SongRepository {
    /** The local play log, newest first. */
    fun observeSongHistory(limit: Long = 100): Flow<List<SongHistoryEntry>>

    /** One page of the backend song archive (newest first). */
    suspend fun getArchive(limit: Int?, cursor: String?): SongsArchiveResponse

    /** Record a play locally, then best-effort POST to the server. */
    suspend fun markPlayed(song: SongDto, playedAtMs: Long)
}

/** Default [SongRepository]. History is authoritative locally; the server call never fails the local write. */
class DefaultSongRepository(
    private val apiClient: ApiClient,
    private val database: RunaDatabase,
    // The archive ends at the user's local day, so tomorrow's song does not show tonight.
    private val today: () -> LocalDate = { Clock.System.now().toLocalDateTime(TimeZone.currentSystemDefault()).date },
) : SongRepository {

    override fun observeSongHistory(limit: Long): Flow<List<SongHistoryEntry>> =
        database.todayQueries.selectHistory(limit)
            .asFlow()
            .mapToList(Dispatchers.Default)
            .map { rows ->
                rows.map {
                    SongHistoryEntry(
                        id = it.id, songId = it.song_id, title = it.title,
                        artist = it.artist, playedAtMs = it.played_at,
                    )
                }
            }

    override suspend fun getArchive(limit: Int?, cursor: String?): SongsArchiveResponse =
        apiClient.getSongs(until = today().toString(), limit = limit, cursor = cursor)

    override suspend fun markPlayed(song: SongDto, playedAtMs: Long) = withContext(Dispatchers.Default) {
        database.todayQueries.insertPlay(
            id = randomId(),
            song_id = song.id,
            title = song.title,
            artist = song.artist,
            played_at = playedAtMs,
        )
        try {
            apiClient.markSongPlayed(song.id, Instant.fromEpochMilliseconds(playedAtMs).toString())
        } catch (_: Exception) {
            // Offline or server error: the play stays local.
        }
    }
}

/** A random v4-style UUID string for local-only ids. */
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
