package com.runa.shared.network.auth

import com.runa.shared.network.dto.AuthTokens
import com.runa.shared.network.dto.RefreshRequest
import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.request.post
import io.ktor.client.request.setBody
import io.ktor.client.statement.HttpResponse
import io.ktor.http.ContentType
import io.ktor.http.appendPathSegments
import io.ktor.http.contentType
import io.ktor.http.isSuccess
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock

/**
 * Refreshes the access token after a 401. Must use the bare client so a failing refresh cannot recurse;
 * the [Mutex] collapses concurrent refreshes so only the first hits the network.
 */
class TokenRefresher(
    private val bareClient: HttpClient,
    private val baseUrl: String,
    private val tokenStore: TokenStore,
) {
    private val mutex = Mutex()

    /**
     * Returns the new access token, or null when the session has ended (store cleared, [TokenStore.sessionExpired] fired).
     * If the stored token already differs from [previousAccess], another coroutine refreshed first and its token is returned.
     */
    suspend fun refresh(previousAccess: String?): String? = mutex.withLock {
        val current = tokenStore.load()
        if (current != null && previousAccess != null && current.accessToken != previousAccess) {
            return@withLock current.accessToken
        }

        val refreshToken = current?.refreshToken
        if (refreshToken == null) {
            tokenStore.clearAndNotifyExpired()
            return@withLock null
        }

        val tokens = try {
            val response: HttpResponse = bareClient.post(baseUrl) {
                url { appendPathSegments("api", "v1", "auth", "refresh") }
                contentType(ContentType.Application.Json)
                setBody(RefreshRequest(refreshToken))
            }
            if (response.status.isSuccess()) response.body<AuthTokens>() else null
        } catch (_: Exception) {
            null
        }

        if (tokens == null) {
            tokenStore.clearAndNotifyExpired()
            return@withLock null
        }

        tokenStore.save(StoredTokens(tokens.accessToken, tokens.refreshToken))
        tokens.accessToken
    }
}
