package com.runa.shared.network.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** DTOs for /api/v1/today, /songs and /songs/{id}/played. The moon phase is computed on the client, not served. */

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

/** A day's song from Apple's catalog. [previewUrl] is the 30-second preview: stream it, never cache it. */
@Serializable
data class SongDto(
    val id: String,
    val date: String,
    val title: String,
    val artist: String,
    @SerialName("artwork_url") val artworkUrl: String,
    @SerialName("preview_url") val previewUrl: String,
    @SerialName("store_url") val storeUrl: String,
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
