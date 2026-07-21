package com.runa.shared.feature.lock

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

/**
 * The privacy-lock gate state. This is a layer SEPARATE from authentication
 * (sign-in): even an authenticated session is hidden behind [Locked] until the
 * biometric prompt succeeds. Content is shown for [Unlocked] and, to avoid a
 * permanent lockout, also for [Unavailable].
 */
sealed interface AppLockUiState {
    /** Content is visible — either the lock is off, or auth just succeeded. */
    data object Unlocked : AppLockUiState

    /** Locked: the lock screen is shown, awaiting authentication. */
    data object Locked : AppLockUiState

    /** The biometric prompt is in progress. */
    data object Authenticating : AppLockUiState

    /** No device security available; content is revealed with a quiet notice
     *  rather than trapping the user behind a lock that can't be satisfied. */
    data object Unavailable : AppLockUiState
}

/**
 * Drives the privacy-lock gate. When the lock is enabled the app starts [Locked]
 * and re-locks whenever it returns from the background (immediate lock, no delay).
 * The native shell reports lifecycle via [onAppBackgrounded] / [onAppForegrounded]
 * and drives the prompt via [authenticate]; the biometric itself is the platform
 * [BiometricAuthenticator] (device-passcode fallback lives inside it).
 *
 * Toggling the lock OFF unlocks immediately; toggling it ON does not lock the
 * current foreground session — it takes effect on the next return to foreground —
 * so the user isn't locked out the instant they enable it.
 */
class AppLockViewModel(
    private val repository: AppLockRepository,
    private val authenticator: BiometricAuthenticator,
    private val scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Default),
) {
    private val _state = MutableStateFlow<AppLockUiState>(
        if (isLockEnabled()) AppLockUiState.Locked else AppLockUiState.Unlocked,
    )
    val state: StateFlow<AppLockUiState> = _state.asStateFlow()

    /** The persisted on/off setting, driving the 22 プライバシー・ロック toggle. */
    val lockEnabled: StateFlow<Boolean> = repository.observeLockEnabled()

    init {
        // Turning the lock off reveals content at once; turning it on is deferred to
        // the next foreground so enabling it never locks the user out mid-action.
        scope.launch {
            repository.observeLockEnabled().collect { enabled ->
                if (!enabled) _state.value = AppLockUiState.Unlocked
            }
        }
    }

    /** Re-lock on the way out (immediate policy) so returning always re-prompts and
     *  the app-switcher preview can't show private content. */
    fun onAppBackgrounded() {
        if (isLockEnabled() && _state.value == AppLockUiState.Unlocked) {
            _state.value = AppLockUiState.Locked
        }
    }

    /** On returning to foreground (and at launch), prompt if still locked. */
    fun onAppForegrounded() {
        if (_state.value == AppLockUiState.Locked) authenticate()
    }

    /** Synchronous snapshots, so the iOS gate seeds correctly before the first frame
     *  (no private content flashing behind a lock that should already be engaged). */
    fun currentState(): AppLockUiState = _state.value
    fun currentLockEnabled(): Boolean = lockEnabled.value

    /** Toggle the lock on/off from the 22 プライバシー・ロック screen. */
    fun setLockEnabled(enabled: Boolean) {
        repository.setLockEnabled(enabled)
    }

    /** Whether biometric-or-passcode can be used, so the 22 screen can warn when a
     *  device has no security set up (enabling the lock would then be ineffective). */
    fun biometricAvailable(): Boolean =
        authenticator.availability() == BiometricAvailability.AVAILABLE

    /** Present the biometric prompt. Called by the gate on appear and on retry. */
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
        scope.launch {
            _state.value = when (authenticator.authenticate()) {
                BiometricResult.Success -> AppLockUiState.Unlocked
                BiometricResult.Failed -> AppLockUiState.Locked
                BiometricResult.Unavailable -> AppLockUiState.Unavailable
            }
        }
    }

    private fun isLockEnabled(): Boolean = repository.observeLockEnabled().value
}
