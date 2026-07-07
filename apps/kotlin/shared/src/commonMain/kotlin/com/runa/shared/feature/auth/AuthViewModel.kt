package com.runa.shared.feature.auth

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch

/**
 * Shared auth view model. Mirrors [HealthzViewModel]'s shape: it owns a
 * [CoroutineScope] and exposes a [StateFlow] that Android collects directly and
 * iOS observes via SKIE.
 *
 * [state] is [AuthRepository.authState] verbatim, so the repository stays the one
 * source of truth. Construction triggers [restore] so the app boots straight into
 * the correct screen (splash → sign-in or tabs).
 */
class AuthViewModel(
    private val repository: AuthRepository,
    private val scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Default),
) {
    val state: StateFlow<AuthState> = repository.authState

    init {
        restore()
    }

    fun restore() {
        scope.launch { repository.restoreSession() }
    }

    fun signupEmail(email: String, password: String, displayName: String?) {
        scope.launch { repository.signupEmail(email, password, displayName) }
    }

    fun loginEmail(email: String, password: String) {
        scope.launch { repository.loginEmail(email, password) }
    }

    fun loginApple(idToken: String, displayName: String?) {
        scope.launch { repository.loginApple(idToken, displayName) }
    }

    fun loginGoogle(idToken: String) {
        scope.launch { repository.loginGoogle(idToken) }
    }

    fun logout() {
        scope.launch { repository.logout() }
    }

    fun clearError() {
        repository.clearError()
    }
}
