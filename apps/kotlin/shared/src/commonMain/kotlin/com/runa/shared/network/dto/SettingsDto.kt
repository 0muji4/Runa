package com.runa.shared.network.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/**
 * Request/response DTOs for the account-data endpoints (PATCH /me, GET /me/export).
 * As with the other DTOs the backend speaks snake_case JSON, so camelCase fields
 * carry an explicit [SerialName].
 */

/** Body for PATCH /api/v1/me — the only editable profile field for now. */
@Serializable
data class UpdateMeRequest(
    @SerialName("display_name") val displayName: String,
)

/** GET /api/v1/me/export payload. Diaries reuse [DiaryEntryDto]; images carry a
 *  short-lived presigned URL that is absent when object storage is unavailable. */
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
