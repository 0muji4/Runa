package com.runa.shared.feature.auth

import com.runa.shared.network.dto.UserDto

/**
 * The app-wide authentication state. The whole UI (both platforms) subscribes to
 * this via [AuthRepository.authState] / [AuthViewModel.state], and later feature
 * slices can gate on [Authenticated].
 *
 * [Restoring] is the initial state while the app checks the stored session at
 * startup; the other four are the states named in the feature spec.
 */
sealed interface AuthState {
    /** Startup: checking the secure store for a session (drives the splash). */
    data object Restoring : AuthState

    /** No session; show the sign-in flow. */
    data object Unauthenticated : AuthState

    /** A sign-in call is in flight. */
    data object Authenticating : AuthState

    /** Signed in; carries the current user for /me display. */
    data class Authenticated(val user: UserDto) : AuthState

    /** The last sign-in attempt failed; [message] is user-presentable. */
    data class Error(val message: String) : AuthState
}
