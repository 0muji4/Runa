package com.runa.shared.feature.health

import com.runa.shared.network.ApiClient
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
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
 * NOTE: this deliberately owns its own [CoroutineScope] rather than depending on
 * a platform lifecycle, so the same instance works from both platforms. A future
 * refactor may introduce explicit lifecycle disposal; tracked as tech debt.
 */
class HealthzViewModel(
    private val apiClient: ApiClient,
    private val scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Default),
) {
    private val _state = MutableStateFlow<HealthzUiState>(HealthzUiState.Loading)
    val state: StateFlow<HealthzUiState> = _state.asStateFlow()

    init {
        check()
    }

    fun check() {
        scope.launch {
            _state.value = HealthzUiState.Loading
            _state.value = try {
                HealthzUiState.Ok(apiClient.healthz().status)
            } catch (e: Exception) {
                HealthzUiState.Error(e.message ?: "unknown error")
            }
        }
    }
}
