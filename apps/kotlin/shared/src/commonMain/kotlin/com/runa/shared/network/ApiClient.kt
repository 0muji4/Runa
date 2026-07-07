package com.runa.shared.network

import com.runa.shared.network.dto.ApiError
import com.runa.shared.network.dto.AppleLoginRequest
import com.runa.shared.network.dto.AuthTokens
import com.runa.shared.network.dto.GoogleLoginRequest
import com.runa.shared.network.dto.HealthzResponse
import com.runa.shared.network.dto.LoginRequest
import com.runa.shared.network.dto.LogoutRequest
import com.runa.shared.network.dto.RefreshRequest
import com.runa.shared.network.dto.SignupRequest
import com.runa.shared.network.dto.UserDto
import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.request.get
import io.ktor.client.request.post
import io.ktor.client.request.setBody
import io.ktor.client.statement.HttpResponse
import io.ktor.http.ContentType
import io.ktor.http.appendPathSegments
import io.ktor.http.contentType
import io.ktor.http.isSuccess

/**
 * The single network surface shared between Android and iOS.
 *
 * The auth methods post to /api/v1/auth/... and read /api/v1/me. Automatic Bearer
 * injection and 401→refresh→retry live in the underlying [HttpClient]
 * ([HttpClientFactory.createAuthenticated]), so this interface stays a plain,
 * request/response seam.
 */
interface ApiClient {
    suspend fun healthz(): HealthzResponse

    suspend fun signup(req: SignupRequest): AuthTokens
    suspend fun login(req: LoginRequest): AuthTokens
    suspend fun loginApple(req: AppleLoginRequest): AuthTokens
    suspend fun loginGoogle(req: GoogleLoginRequest): AuthTokens
    suspend fun refresh(req: RefreshRequest): AuthTokens
    suspend fun logout(req: LogoutRequest)
    suspend fun getMe(): UserDto
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

    override suspend fun signup(req: SignupRequest): AuthTokens =
        postJson(listOf("api", "v1", "auth", "signup"), req).decodeOrThrow()

    override suspend fun login(req: LoginRequest): AuthTokens =
        postJson(listOf("api", "v1", "auth", "login"), req).decodeOrThrow()

    override suspend fun loginApple(req: AppleLoginRequest): AuthTokens =
        postJson(listOf("api", "v1", "auth", "apple"), req).decodeOrThrow()

    override suspend fun loginGoogle(req: GoogleLoginRequest): AuthTokens =
        postJson(listOf("api", "v1", "auth", "google"), req).decodeOrThrow()

    override suspend fun refresh(req: RefreshRequest): AuthTokens =
        postJson(listOf("api", "v1", "auth", "refresh"), req).decodeOrThrow()

    override suspend fun logout(req: LogoutRequest) {
        val response = postJson(listOf("api", "v1", "auth", "logout"), req)
        if (!response.status.isSuccess()) response.throwApiError()
    }

    override suspend fun getMe(): UserDto =
        httpClient.get(baseUrl) {
            url { appendPathSegments("api", "v1", "me") }
        }.decodeOrThrow()

    private suspend inline fun <reified B> postJson(segments: List<String>, body: B): HttpResponse =
        httpClient.post(baseUrl) {
            url { appendPathSegments(segments) }
            contentType(ContentType.Application.Json)
            setBody(body)
        }
}

/** Decodes a successful body as [T], otherwise throws [ApiException] built from
 *  the shared error envelope. */
private suspend inline fun <reified T> HttpResponse.decodeOrThrow(): T {
    if (status.isSuccess()) return body()
    throwApiError()
}

private suspend fun HttpResponse.throwApiError(): Nothing {
    val parsed = try {
        body<ApiError>()
    } catch (_: Exception) {
        null
    }
    throw ApiException(
        statusCode = status.value,
        code = parsed?.error?.code,
        message = parsed?.error?.message ?: "request failed with status ${status.value}",
    )
}
