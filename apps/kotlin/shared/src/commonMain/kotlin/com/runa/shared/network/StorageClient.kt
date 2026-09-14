package com.runa.shared.network

import io.ktor.client.HttpClient
import io.ktor.client.plugins.onUpload
import io.ktor.client.request.put
import io.ktor.client.request.setBody
import io.ktor.client.statement.HttpResponse
import io.ktor.http.ContentType
import io.ktor.http.contentType
import io.ktor.http.isSuccess

/**
 * Uploads raw image bytes directly to object storage via a presigned URL; Bearer-free (see
 * [HttpClientFactory.createStorage]).
 */
interface StorageClient {
    /**
     * PUT [bytes] to a presigned [url]; [onProgress] receives a 0..1 fraction. Throws
     * [StorageException] on non-2xx.
     */
    suspend fun putBytes(
        url: String,
        bytes: ByteArray,
        contentType: String,
        onProgress: (Float) -> Unit = {},
    )
}

/** Thrown when a direct storage PUT fails (non-2xx). */
class StorageException(val statusCode: Int, message: String) : Exception(message)

/** Ktor-backed [StorageClient] over the bare storage [HttpClient]. */
class KtorStorageClient(private val client: HttpClient) : StorageClient {
    override suspend fun putBytes(
        url: String,
        bytes: ByteArray,
        contentType: String,
        onProgress: (Float) -> Unit,
    ) {
        val response: HttpResponse = client.put(url) {
            contentType(ContentType.parse(contentType))
            setBody(bytes)
            onUpload { sent, total ->
                if (total != null && total > 0L) {
                    onProgress((sent.toFloat() / total.toFloat()).coerceIn(0f, 1f))
                }
            }
        }
        if (!response.status.isSuccess()) {
            throw StorageException(response.status.value, "storage PUT failed: ${response.status}")
        }
        onProgress(1f)
    }
}
