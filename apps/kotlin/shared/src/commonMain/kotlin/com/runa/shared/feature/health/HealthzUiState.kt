package com.runa.shared.feature.health

/**
 * UI state for the health-check probe surfaced on the Home tab.
 *
 * This is the ONLY feature state in the walking skeleton; every product feature
 * (today's song, diary, gallery) will add its own state type alongside this one.
 */
sealed interface HealthzUiState {
    data object Loading : HealthzUiState
    data class Ok(val status: String) : HealthzUiState
    data class Error(val message: String) : HealthzUiState
}
