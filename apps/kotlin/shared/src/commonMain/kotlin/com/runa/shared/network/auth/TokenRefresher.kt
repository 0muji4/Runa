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
 * Performs the access-token refresh when a protected request returns 401.
 *
 * It uses a **bare** HTTP client (no auth interceptor) to hit /auth/refresh so a
 * failing refresh can never recurse into itself. A [Mutex] collapses concurrent
 * refreshes: if several requests 401 at once, only the first refreshes and the
 * rest reuse the freshly stored token.
 */
class TokenRefresher(
    private val bareClient: HttpClient,
    private val baseUrl: String,
    private val tokenStore: TokenStore,
) {
    private val mutex = Mutex()

    /**
     * Refreshes the token pair and returns the new access token, or null when the
     * session has ended (no refresh token, or the server rejected it). On failure
     * it clears the store and fires [TokenStore.sessionExpired].
     *
     * @param previousAccess the access token the caller tried to use; if the
     * stored token already differs, another coroutine refreshed first and its
     * token is returned without a second network call.
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
