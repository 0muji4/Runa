package com.runa.shared.network.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** DTOs for /api/v1/auth and /api/v1/me. The wire format is snake_case, hence the explicit [SerialName]s. */

@Serializable
data class SignupRequest(
    val email: String,
    val password: String,
    @SerialName("display_name") val displayName: String? = null,
)

@Serializable
data class LoginRequest(
    val email: String,
    val password: String,
)

@Serializable
data class AppleLoginRequest(
    @SerialName("id_token") val idToken: String,
    @SerialName("display_name") val displayName: String? = null,
)

@Serializable
data class GoogleLoginRequest(
    @SerialName("id_token") val idToken: String,
)

@Serializable
data class RefreshRequest(
    @SerialName("refresh_token") val refreshToken: String,
)

@Serializable
data class LogoutRequest(
    @SerialName("refresh_token") val refreshToken: String,
)

/** Token bundle returned by every successful auth call; [user] is omitted on refresh. */
@Serializable
data class AuthTokens(
    @SerialName("access_token") val accessToken: String,
    @SerialName("refresh_token") val refreshToken: String,
    @SerialName("token_type") val tokenType: String = "Bearer",
    @SerialName("expires_in") val expiresIn: Int = 0,
    val user: UserDto? = null,
)

/** The authenticated user, as returned by /me and embedded in [AuthTokens]. */
@Serializable
data class UserDto(
    val id: String,
    val email: String? = null,
    @SerialName("display_name") val displayName: String,
    @SerialName("auth_provider") val authProvider: String,
    @SerialName("is_premium") val isPremium: Boolean = false,
    @SerialName("premium_expires_at") val premiumExpiresAt: String? = null,
    @SerialName("created_at") val createdAt: String? = null,
)

/** Shared error envelope: {"error": {"code": "...", "message": "..."}}. */
@Serializable
data class ApiError(val error: ApiErrorBody)

@Serializable
data class ApiErrorBody(
    val code: String,
    val message: String,
)
