package com.runa.shared.feature.auth

import com.runa.shared.network.HttpClientFactory
import com.runa.shared.network.KtorApiClient
import com.runa.shared.network.auth.SecureKeyValueStore
import com.runa.shared.network.auth.TokenRefresher
import com.runa.shared.network.auth.TokenStore
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.MockRequestHandleScope
import io.ktor.client.engine.mock.respond
import io.ktor.client.request.HttpRequestData
import io.ktor.client.request.HttpResponseData
import io.ktor.http.HttpHeaders
import io.ktor.http.HttpStatusCode
import io.ktor.http.headersOf

const val BASE_URL = "http://localhost:8080"

/** In-memory [SecureKeyValueStore] for tests — no device Keychain/prefs needed. */
class FakeSecureStore(initial: Map<String, String> = emptyMap()) : SecureKeyValueStore {
    private val map = initial.toMutableMap()
    override fun get(key: String): String? = map[key]
    override fun set(key: String, value: String) { map[key] = value }
    override fun remove(key: String) { map.remove(key) }
}

typealias MockHandler = suspend MockRequestHandleScope.(HttpRequestData) -> HttpResponseData

/**
 * Wires the real auth graph (TokenStore → refresher → auth client → ApiClient →
 * repository) over a [MockEngine], so tests exercise the actual Bearer-injection
 * and 401→refresh→retry logic. Two engines share one [handler]: the bare client
 * (refresh) and the authenticated client (protected calls).
 */
class AuthHarness(
    initialTokens: Map<String, String> = emptyMap(),
    handler: MockHandler,
) {
    val secureStore = FakeSecureStore(initialTokens)
    val tokenStore = TokenStore(secureStore)

    private val bareClient = HttpClientFactory.createBase(MockEngine(handler))
    private val refresher = TokenRefresher(bareClient, BASE_URL, tokenStore)
    private val authClient = HttpClientFactory.createAuthenticated(MockEngine(handler), tokenStore, refresher)

    val apiClient = KtorApiClient(authClient, BASE_URL)
    val repository = DefaultAuthRepository(apiClient, tokenStore)
}

/** Keys mirror [TokenStore]'s private constants. */
const val KEY_ACCESS = "auth.access_token"
const val KEY_REFRESH = "auth.refresh_token"

fun MockRequestHandleScope.jsonOk(body: String, status: HttpStatusCode = HttpStatusCode.OK): HttpResponseData =
    respond(body, status, headersOf(HttpHeaders.ContentType, "application/json"))

fun authTokensJson(access: String, refresh: String, withUser: Boolean = true): String {
    val user = if (withUser) ""","user":${userJson()}""" else ""
    return """{"access_token":"$access","refresh_token":"$refresh","token_type":"Bearer","expires_in":900$user}"""
}

fun userJson(): String =
    """{"id":"u1","email":"a@b.com","display_name":"Runa","auth_provider":"email","is_premium":false,"created_at":"2026-01-01T00:00:00Z"}"""

fun errorJson(code: String): String = """{"error":{"code":"$code","message":"$code"}}"""
