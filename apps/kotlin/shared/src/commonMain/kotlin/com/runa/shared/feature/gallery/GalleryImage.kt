package com.runa.shared.feature.gallery

/** A gallery image as the UI sees it; [serverId] is null until the upload completes. */
data class GalleryImage(
    val clientId: String,
    val serverId: String?,
    val width: Int,
    val height: Int,
    val viewUrl: String?,
    val localBytes: ByteArray?,
    val createdAtEpochMs: Long,
    val uploadState: UploadState,
    val progress: Float,
) {
    // ByteArray needs content equality so list diffing (Compose/SwiftUI) is correct.
    override fun equals(other: Any?): Boolean {
        if (this === other) return true
        if (other !is GalleryImage) return false
        return clientId == other.clientId &&
            serverId == other.serverId &&
            width == other.width &&
            height == other.height &&
            viewUrl == other.viewUrl &&
            createdAtEpochMs == other.createdAtEpochMs &&
            uploadState == other.uploadState &&
            progress == other.progress &&
            localBytes.contentEqualsOrNull(other.localBytes)
    }

    override fun hashCode(): Int {
        var result = clientId.hashCode()
        result = 31 * result + (serverId?.hashCode() ?: 0)
        result = 31 * result + width
        result = 31 * result + height
        result = 31 * result + (viewUrl?.hashCode() ?: 0)
        result = 31 * result + createdAtEpochMs.hashCode()
        result = 31 * result + uploadState.hashCode()
        result = 31 * result + progress.hashCode()
        result = 31 * result + (localBytes?.contentHashCode() ?: 0)
        return result
    }
}

private fun ByteArray?.contentEqualsOrNull(other: ByteArray?): Boolean =
    if (this == null || other == null) this === other else this.contentEquals(other)

/** Where an image is in its upload lifecycle (derived from sync_state + progress). */
enum class UploadState {
    Queued,
    Uploading,
    Uploaded,
    Failed,
}
