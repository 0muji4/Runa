package com.runa.shared.network.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** DTOs for the account-data endpoints (PATCH /me, GET /me/export). */

/** Body for PATCH /api/v1/me. */
@Serializable
data class UpdateMeRequest(
    @SerialName("display_name") val displayName: String,
)

/** GET /api/v1/me/export payload; image [ExportImageDto.url] is absent when object storage is unavailable. */
@Serializable
data class ExportDto(
    @SerialName("exported_at") val exportedAt: String,
    @SerialName("schema_version") val schemaVersion: Int,
    val user: UserDto,
    val diaries: List<DiaryEntryDto> = emptyList(),
    val images: List<ExportImageDto> = emptyList(),
)

@Serializable
data class ExportImageDto(
    val id: String,
    @SerialName("object_key") val objectKey: String,
    val width: Int,
    val height: Int,
    val theme: String,
    @SerialName("created_at") val createdAt: String,
    val url: String? = null,
    @SerialName("url_expires_at") val urlExpiresAt: String? = null,
)
