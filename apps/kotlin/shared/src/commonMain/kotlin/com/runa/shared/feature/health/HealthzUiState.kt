package com.runa.shared.feature.health

/** UI state for the health-check probe surfaced on the Home tab. */
sealed interface HealthzUiState {
    data object Loading : HealthzUiState
    data class Ok(val status: String) : HealthzUiState
    data class Error(val message: String) : HealthzUiState
}
