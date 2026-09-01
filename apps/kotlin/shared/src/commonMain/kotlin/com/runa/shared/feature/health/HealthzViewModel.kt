package com.runa.shared.feature.health

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.runa.shared.network.ApiClient
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

/**
 * Drives the health-check probe. Shared across Android (consumed directly) and
 * iOS (exposed to Swift via SKIE, which maps [StateFlow] to an observable).
 *
 * It runs an initial [check] on construction so the Home tab shows a result
 * without any explicit trigger from the UI.
 *
 * Work runs on [androidx.lifecycle.viewModelScope], so it is cancelled when the
 * view model is cleared — on Android by the ViewModelStore, on iOS by the observable
 * calling `clear()` in its `deinit`.
 */
class HealthzViewModel(
    private val apiClient: ApiClient,
) : ViewModel() {
    private val _state = MutableStateFlow<HealthzUiState>(HealthzUiState.Loading)
    val state: StateFlow<HealthzUiState> = _state.asStateFlow()

    init {
        check()
    }

    fun check() {
        viewModelScope.launch {
            _state.value = HealthzUiState.Loading
            _state.value = try {
                HealthzUiState.Ok(apiClient.healthz().status)
            } catch (e: Exception) {
                HealthzUiState.Error(e.message ?: "unknown error")
            }
        }
    }
}
