package com.runa.shared.feature.today

/**
 * UI state for the home screen (the day's quote + moon + song). Mirrors the
 * shape of the other feature states (Loading / content / error) with an extra
 * [Offline] case so the UI can show a quiet "offline" hint while still rendering
 * the cached copy and the (always-computed) moon.
 */
sealed interface HomeUiState {
    data object Loading : HomeUiState

    /** Fresh content from the backend. */
    data class Content(val today: Today) : HomeUiState

    /** Cached quote/song (may be null) plus the freshly computed moon, offline. */
    data class Offline(val today: Today) : HomeUiState

    data class Error(val message: String) : HomeUiState
}
