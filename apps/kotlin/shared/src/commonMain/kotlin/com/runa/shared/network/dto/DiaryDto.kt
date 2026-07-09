package com.runa.shared.network.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * Request/response DTOs for the /api/v1/diary endpoints. As with [AuthDto] the
 * backend speaks snake_case JSON, so camelCase fields carry an explicit
 * [SerialName]. Timestamps are RFC3339 strings on the wire.
 */

@Serializable
data class CreateDiaryRequest(
    @SerialName("body_text") val bodyText: String,
    val mood: String? = null,
    @SerialName("client_id") val clientId: String,
    @SerialName("created_at") val createdAt: String,
)

@Serializable
data class UpdateDiaryRequest(
    @SerialName("body_text") val bodyText: String,
    val mood: String? = null,
)

/** One diary entry as returned by every diary endpoint. [deletedAt] is only
 *  non-null on tombstones seen through /diary/sync. */
@Serializable
data class DiaryEntryDto(
    val id: String,
    @SerialName("client_id") val clientId: String,
    @SerialName("body_text") val bodyText: String,
    val mood: String? = null,
    @SerialName("created_at") val createdAt: String,
    @SerialName("updated_at") val updatedAt: String,
    @SerialName("deleted_at") val deletedAt: String? = null,
)

/** GET /diary page. [nextCursor] is null on the last page. */
@Serializable
data class DiaryListResponse(
    val entries: List<DiaryEntryDto>,
    @SerialName("next_cursor") val nextCursor: String? = null,
)

/** GET /diary/sync delta. [serverTime] becomes the client's next `since`. */
@Serializable
data class DiarySyncResponse(
    val entries: List<DiaryEntryDto>,
    @SerialName("server_time") val serverTime: String,
)
