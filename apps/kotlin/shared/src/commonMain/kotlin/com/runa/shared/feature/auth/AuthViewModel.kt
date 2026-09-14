package com.runa.shared.feature.auth

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch

/** Shared auth view model; [state] is [AuthRepository.authState] verbatim and construction triggers [restore]. */
class AuthViewModel(
    private val repository: AuthRepository,
) : ViewModel() {
    val state: StateFlow<AuthState> = repository.authState

    init {
        restore()
    }

    fun restore() {
        viewModelScope.launch { repository.restoreSession() }
    }

    fun signupEmail(email: String, password: String, displayName: String?) {
        viewModelScope.launch { repository.signupEmail(email, password, displayName) }
    }

    fun loginEmail(email: String, password: String) {
        viewModelScope.launch { repository.loginEmail(email, password) }
    }

    fun loginApple(idToken: String, displayName: String?) {
        viewModelScope.launch { repository.loginApple(idToken, displayName) }
    }

    fun loginGoogle(idToken: String) {
        viewModelScope.launch { repository.loginGoogle(idToken) }
    }

    fun logout() {
        viewModelScope.launch { repository.logout() }
    }

    fun clearError() {
        repository.clearError()
    }
}
