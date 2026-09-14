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
 * The composed home payload. [isOffline] is true when the quote/song came from the
 * local cache after a failed fetch; the moon is always computed locally.
 */
data class Today(
    val dateLabel: String,
    val quote: QuoteDto?,
    val song: SongDto?,
    val moon: MoonPhase,
    val isOffline: Boolean,
)

interface TodayRepository {
    /** Fetch (and cache) the day's quote+song; falls back to the cache when offline. */
    suspend fun getToday(localDate: LocalDate, zone: TimeZone): Today
}

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
            artwork_url = s.artworkUrl, preview_url = s.previewUrl, store_url = s.storeUrl,
        )

    private fun cachedQuote(dateKey: String): QuoteDto? =
        database.todayQueries.selectQuote(dateKey).executeAsOneOrNull()
            ?.let { QuoteDto(id = it.id, date = it.date, bodyText = it.body_text) }

    private fun cachedSong(dateKey: String): SongDto? =
        database.todayQueries.selectSong(dateKey).executeAsOneOrNull()
            ?.let {
                SongDto(
                    id = it.id, date = it.date, title = it.title, artist = it.artist,
                    artworkUrl = it.artwork_url, previewUrl = it.preview_url, storeUrl = it.store_url,
                )
            }
}
