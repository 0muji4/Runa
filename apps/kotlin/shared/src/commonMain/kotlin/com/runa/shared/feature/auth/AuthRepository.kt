package com.runa.shared.feature.auth

import com.runa.shared.network.dto.UserDto
import kotlinx.coroutines.flow.StateFlow

/** The authentication boundary: owns the single source of truth [authState] and persists tokens. */
interface AuthRepository {

    val authState: StateFlow<AuthState>

    suspend fun signupEmail(email: String, password: String, displayName: String?): Result<Unit>
    suspend fun loginEmail(email: String, password: String): Result<Unit>
    suspend fun loginApple(idToken: String, displayName: String?): Result<Unit>
    suspend fun loginGoogle(idToken: String): Result<Unit>

    /** Explicitly refresh the token pair; the 401 path already refreshes in the HTTP layer. */
    suspend fun refresh(): Result<Unit>

    suspend fun logout(): Result<Unit>

    suspend fun getMe(): Result<UserDto>

    /** Startup: load any stored session and confirm it with /me. */
    suspend fun restoreSession()

    /** Dismiss an [AuthState.Error] back to [AuthState.Unauthenticated]. */
    fun clearError()

    /** Local-only session teardown after the account was deleted server-side.
     *  Unlike [logout] this must make NO network call (the account no longer exists). */
    fun endSession()

    /** Replace the cached user in [authState] when [AuthState.Authenticated]; no-op otherwise. */
    fun updateCachedUser(user: UserDto)
}
