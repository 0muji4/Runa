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
 * Builds the shared [HttpClient]s: [createBase] (JSON, no auth; used by [TokenRefresher] so a refresh
 * never triggers another), [createAuthenticated] (Bearer + one 401→refresh→replay), and
 * [createStorage] (bare; must NOT be the authenticated one or the Runa Bearer would leak to the storage host).
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

    fun createStorage(engine: HttpClientEngine): HttpClient = HttpClient(engine)

    fun createAuthenticated(
        engine: HttpClientEngine,
        tokenStore: TokenStore,
        refresher: TokenRefresher,
    ): HttpClient {
        val client = HttpClient(engine) {
            install(ContentNegotiation) { json(json) }
        }

        client.plugin(HttpSend).intercept { request ->
            // Public auth endpoints (/api/v1/auth/...): no Bearer, no refresh handling.
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
