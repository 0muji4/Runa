package com.runa.shared.network

import com.runa.shared.network.auth.TokenRefresher
import com.runa.shared.network.auth.TokenStore
import io.ktor.client.HttpClient
import io.ktor.client.engine.HttpClientEngine
import io.ktor.client.plugins.HttpSend
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.plugins.plugin
import io.ktor.http.HttpHeaders
import io.ktor.http.HttpStatusCode
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.Json

/**
 * Builds the shared [HttpClient]s.
 *
 * Two flavours exist:
 *  - [createBase]: JSON only, no auth. Used by the [TokenRefresher] to call
 *    /auth/refresh without risking a refresh-of-a-refresh loop.
 *  - [createAuthenticated]: JSON plus an [HttpSend] interceptor that attaches the
 *    stored Bearer access token and, on a 401 from a protected route, refreshes
 *    once and replays the original request. Public /api/v1/auth/... routes are
 *    skipped so sign-in calls carry no token and never trigger a refresh.
 */
object HttpClientFactory {

    val json: Json = Json {
        ignoreUnknownKeys = true
        isLenient = true
    }

    fun createBase(engine: HttpClientEngine): HttpClient =
        HttpClient(engine) {
            install(ContentNegotiation) { json(json) }
        }

    fun createAuthenticated(
        engine: HttpClientEngine,
        tokenStore: TokenStore,
        refresher: TokenRefresher,
    ): HttpClient {
        val client = HttpClient(engine) {
            install(ContentNegotiation) { json(json) }
        }

        client.plugin(HttpSend).intercept { request ->
            // Public auth endpoints (/api/v1/auth/...): send as-is, no Bearer, no
            // refresh handling. Checked via path segments (Ktor 3 URL API).
            if (request.url.encodedPathSegments.any { it == "auth" }) {
                return@intercept execute(request)
            }

            val access = tokenStore.load()?.accessToken
            if (access != null) {
                request.headers.remove(HttpHeaders.Authorization)
                request.headers.append(HttpHeaders.Authorization, "Bearer $access")
            }

            var call = execute(request)
            if (call.response.status == HttpStatusCode.Unauthorized) {
                val refreshed = refresher.refresh(access)
                if (refreshed != null) {
                    request.headers.remove(HttpHeaders.Authorization)
                    request.headers.append(HttpHeaders.Authorization, "Bearer $refreshed")
                    call = execute(request)
                }
            }
            call
        }

        return client
    }
}
