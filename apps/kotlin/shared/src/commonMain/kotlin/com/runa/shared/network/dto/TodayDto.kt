package com.runa.shared.network.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * Request/response DTOs for the /api/v1/today, /songs and /songs/{id}/played
 * endpoints. As with the auth DTOs the backend speaks snake_case, so each
 * camelCase field carries an explicit [SerialName].
 *
 * Note the moon phase is NOT part of this contract: it is computed on the client
 * by [com.runa.shared.feature.today.moon.MoonPhaseCalculator].
 */

@Serializable
data class TodayResponse(
    val date: String,
    val quote: QuoteDto? = null,
    val song: SongDto? = null,
)

@Serializable
data class QuoteDto(
    val id: String,
    val date: String,
    @SerialName("body_text") val bodyText: String,
)

@Serializable
data class SongDto(
    val id: String,
    val date: String,
    val title: String,
    val artist: String,
    @SerialName("artwork_url") val artworkUrl: String,
    @SerialName("audio_url") val audioUrl: String,
)

/** One page of the song archive; [nextCursor] is null on the last page. */
@Serializable
data class SongsArchiveResponse(
    val songs: List<SongDto> = emptyList(),
    @SerialName("next_cursor") val nextCursor: String? = null,
)

/** Optional body for POST /songs/{id}/played; the server clock is used when null. */
@Serializable
data class PlayedRequest(
    @SerialName("played_at") val playedAt: String? = null,
)
