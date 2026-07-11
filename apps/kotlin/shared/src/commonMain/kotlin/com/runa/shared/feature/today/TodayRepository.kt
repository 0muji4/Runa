package com.runa.shared.feature.today

import com.runa.shared.db.RunaDatabase
import com.runa.shared.feature.today.moon.MoonPhase
import com.runa.shared.feature.today.moon.MoonPhaseCalculator
import com.runa.shared.network.ApiClient
import com.runa.shared.network.dto.QuoteDto
import com.runa.shared.network.dto.SongDto
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.datetime.LocalDate
import kotlinx.datetime.TimeZone

/**
 * The composed home payload: the day's quote and song fetched from the backend
 * (or the last-cached copy when offline) combined with the locally computed moon
 * phase. [isOffline] is true when the network fetch failed and the quote/song (if
 * any) came from the SQLDelight cache; the moon is always freshly computed.
 *
 * [dateLabel] is pre-formatted (e.g. "7月11日") here rather than exposing a
 * kotlinx-datetime type, so the UIs need not depend on that library (matching the
 * diary slice's epoch-millis convention).
 */
data class Today(
    val dateLabel: String,
    val quote: QuoteDto?,
    val song: SongDto?,
    val moon: MoonPhase,
    val isOffline: Boolean,
)

/** Loads and composes the home's "today" payload. */
interface TodayRepository {
    /** Fetch (and cache) the day's quote+song and compute its moon phase. Falls
     *  back to the cached quote/song — plus the computed moon — when offline. */
    suspend fun getToday(localDate: LocalDate, zone: TimeZone): Today
}

/**
 * Default [TodayRepository]. The moon phase is computed first and unconditionally
 * (it never needs the network), then the quote/song are fetched and cached; a
 * network failure falls back to the SQLDelight cache so the home still renders
 * the day's copy and moon fully offline.
 */
class DefaultTodayRepository(
    private val apiClient: ApiClient,
    private val database: RunaDatabase,
) : TodayRepository {

    override suspend fun getToday(localDate: LocalDate, zone: TimeZone): Today = withContext(Dispatchers.Default) {
        val moon = MoonPhaseCalculator.phaseFor(localDate, zone)
        val dateKey = localDate.toString() // ISO yyyy-MM-dd, matches the backend
        val dateLabel = "${localDate.monthNumber}月${localDate.dayOfMonth}日"

        try {
            val response = apiClient.getToday(dateKey)
            response.quote?.let { cacheQuote(it) }
            response.song?.let { cacheSong(it) }
            Today(dateLabel, response.quote, response.song, moon, isOffline = false)
        } catch (_: Exception) {
            Today(dateLabel, cachedQuote(dateKey), cachedSong(dateKey), moon, isOffline = true)
        }
    }

    private fun cacheQuote(q: QuoteDto) =
        database.todayQueries.upsertQuote(date = q.date, id = q.id, body_text = q.bodyText)

    private fun cacheSong(s: SongDto) =
        database.todayQueries.upsertSong(
            date = s.date, id = s.id, title = s.title, artist = s.artist,
            artwork_url = s.artworkUrl, audio_url = s.audioUrl,
        )

    private fun cachedQuote(dateKey: String): QuoteDto? =
        database.todayQueries.selectQuote(dateKey).executeAsOneOrNull()
            ?.let { QuoteDto(id = it.id, date = it.date, bodyText = it.body_text) }

    private fun cachedSong(dateKey: String): SongDto? =
        database.todayQueries.selectSong(dateKey).executeAsOneOrNull()
            ?.let {
                SongDto(
                    id = it.id, date = it.date, title = it.title, artist = it.artist,
                    artworkUrl = it.artwork_url, audioUrl = it.audio_url,
                )
            }
}
