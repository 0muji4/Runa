package com.runa.shared.feature.auth

import com.runa.shared.network.dto.UserDto
import kotlinx.coroutines.flow.StateFlow

/**
 * The authentication boundary the rest of the app depends on. It owns the single
 * source of truth [authState] and persists tokens through the secure store.
 *
 * The sign-in methods return [Result] for callers that want the outcome
 * inline; the observable [authState] is the primary way the UI reacts.
 */
interface AuthRepository {

    /** App-wide auth state. Later feature slices subscribe here to run only when
     *  [AuthState.Authenticated]. */
    val authState: StateFlow<AuthState>

    suspend fun signupEmail(email: String, password: String, displayName: String?): Result<Unit>
    suspend fun loginEmail(email: String, password: String): Result<Unit>
    suspend fun loginApple(idToken: String, displayName: String?): Result<Unit>
    suspend fun loginGoogle(idToken: String): Result<Unit>

    /** Explicitly refresh the token pair. The 401 path refreshes automatically in
     *  the HTTP layer; this is for tests and manual use. */
    suspend fun refresh(): Result<Unit>

    suspend fun logout(): Result<Unit>

    suspend fun getMe(): Result<UserDto>

    /** Startup: load any stored session and confirm it with /me. */
    suspend fun restoreSession()

    /** Dismiss an [AuthState.Error] back to [AuthState.Unauthenticated]. */
    fun clearError()
}
