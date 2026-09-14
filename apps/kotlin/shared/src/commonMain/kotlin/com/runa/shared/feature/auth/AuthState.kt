package com.runa.shared.feature.auth

import com.runa.shared.network.dto.UserDto

/** The app-wide authentication state, observed via [AuthRepository.authState]. */
sealed interface AuthState {
    /** Startup: checking the secure store for a session (drives the splash). */
    data object Restoring : AuthState

    data object Unauthenticated : AuthState

    data object Authenticating : AuthState

    data class Authenticated(val user: UserDto) : AuthState

    /** The last sign-in attempt failed; [message] is user-presentable. */
    data class Error(val message: String) : AuthState
}
