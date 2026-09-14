package com.runa.shared.feature.lock

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

/** The privacy-lock gate state, a layer SEPARATE from authentication. Content is
 *  shown for [Unlocked] and — to avoid a permanent lockout — also for [Unavailable]. */
sealed interface AppLockUiState {
    data object Unlocked : AppLockUiState

    data object Locked : AppLockUiState

    data object Authenticating : AppLockUiState

    /** No device security available; content is revealed with a quiet notice. */
    data object Unavailable : AppLockUiState
}

/** Drives the privacy-lock gate; the native shell reports lifecycle via [onAppBackgrounded] / [onAppForegrounded].
 *  Toggling the lock OFF unlocks immediately; toggling it ON must not lock the current
 *  foreground session — it takes effect on the next return to foreground. */
class AppLockViewModel(
    private val repository: AppLockRepository,
    private val authenticator: BiometricAuthenticator,
) : ViewModel() {
    private val _state = MutableStateFlow<AppLockUiState>(
        if (isLockEnabled()) AppLockUiState.Locked else AppLockUiState.Unlocked,
    )
    val state: StateFlow<AppLockUiState> = _state.asStateFlow()

    val lockEnabled: StateFlow<Boolean> = repository.observeLockEnabled()

    init {
        viewModelScope.launch {
            repository.observeLockEnabled().collect { enabled ->
                if (!enabled) _state.value = AppLockUiState.Unlocked
            }
        }
    }

    /** Re-lock immediately on the way out so the app-switcher preview can't show private content. */
    fun onAppBackgrounded() {
        if (isLockEnabled() && _state.value == AppLockUiState.Unlocked) {
            _state.value = AppLockUiState.Locked
        }
    }

    fun onAppForegrounded() {
        if (_state.value == AppLockUiState.Locked) authenticate()
    }

    /** Synchronous snapshots, so the iOS gate seeds correctly before the first frame. */
    fun currentState(): AppLockUiState = _state.value
    fun currentLockEnabled(): Boolean = lockEnabled.value

    fun setLockEnabled(enabled: Boolean) {
        repository.setLockEnabled(enabled)
    }

    /** Whether biometric-or-passcode can be used (the settings screen warns when not). */
    fun biometricAvailable(): Boolean =
        authenticator.availability() == BiometricAvailability.AVAILABLE

    fun authenticate() {
        if (!isLockEnabled()) {
            _state.value = AppLockUiState.Unlocked
            return
        }
        if (authenticator.availability() == BiometricAvailability.UNAVAILABLE) {
            _state.value = AppLockUiState.Unavailable
            return
        }
        if (_state.value == AppLockUiState.Authenticating) return
        _state.value = AppLockUiState.Authenticating
        viewModelScope.launch {
            _state.value = when (authenticator.authenticate()) {
                BiometricResult.Success -> AppLockUiState.Unlocked
                BiometricResult.Failed -> AppLockUiState.Locked
                BiometricResult.Unavailable -> AppLockUiState.Unavailable
            }
        }
    }

    private fun isLockEnabled(): Boolean = repository.observeLockEnabled().value
}
