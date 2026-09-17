package com.runa.shared.network.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** DTOs for /api/v1/gallery. Image bytes never appear here; they go client↔store via the presigned URLs. */

@Serializable
data class GalleryUploadURLRequest(
    @SerialName("content_type") val contentType: String,
    val size: Long,
)

/** The upload target: PUT the bytes to [uploadUrl], then register with [objectKey]. */
@Serializable
data class GalleryUploadURLResponse(
    @SerialName("object_key") val objectKey: String,
    @SerialName("upload_url") val uploadUrl: String,
    val method: String,
    val headers: Map<String, String> = emptyMap(),
    @SerialName("expires_at") val expiresAt: String,
    @SerialName("max_size") val maxSize: Long,
)

@Serializable
data class CreateGalleryRequest(
    @SerialName("object_key") val objectKey: String,
    val width: Int,
    val height: Int,
)

/** One gallery image with a short-lived presigned GET URL ([url]). */
@Serializable
data class GalleryImageDto(
    val id: String,
    val url: String,
    @SerialName("url_expires_at") val urlExpiresAt: String,
    val width: Int,
    val height: Int,
    @SerialName("created_at") val createdAt: String,
)

/** GET /gallery page. [nextCursor] is null on the last page. */
@Serializable
data class GalleryListResponse(
    val items: List<GalleryImageDto>,
    @SerialName("next_cursor") val nextCursor: String? = null,
)
