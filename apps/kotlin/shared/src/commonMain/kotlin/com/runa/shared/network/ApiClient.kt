package com.runa.shared.network

import com.runa.shared.network.dto.HealthzResponse
import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.request.get
import io.ktor.http.appendPathSegments

/**
 * The single network surface shared between Android and iOS.
 *
 * TODO: feature endpoints (today's song, diary, gallery, ...) will be added here
 * as the vertical slices land. Keep this interface as the platform-agnostic seam.
 */
interface ApiClient {
    suspend fun healthz(): HealthzResponse
}

/**
 * Ktor-backed [ApiClient].
 *
 * @param baseUrl host+port ONLY (e.g. http://10.0.2.2:8080), WITHOUT the /api/v1
 * suffix. This class owns the versioned API prefix so callers never hardcode it.
 */
class KtorApiClient(
    private val httpClient: HttpClient,
    private val baseUrl: String,
) : ApiClient {

    override suspend fun healthz(): HealthzResponse =
        httpClient.get(baseUrl) {
            url { appendPathSegments("api", "v1", "healthz") }
        }.body()
}
