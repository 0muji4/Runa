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
 * Drives the health-check probe; runs an initial [check] on construction.
 * Koin で `single` 束縛なのでアプリ寿命で生き続ける。画面ごとに捨てたい view model は
 * `factory` 束縛にし、Android は koinViewModel()、iOS は [com.runa.shared.platform.ViewModelOwner] に預けること。
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
