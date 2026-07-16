package com.runa.shared.feature.todaymoon

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.datetime.TimeZone

/**
 * Shared view model for 15 今日の月. The payload is a pure, offline computation, so
 * [load] is synchronous and [state] settles to [TodayMoonUiState.Content] at once.
 * Android collects [state]; iOS observes it through SKIE.
 */
class TodayMoonViewModel(
    private val repository: TodayMoonRepository,
    private val zone: TimeZone = TimeZone.currentSystemDefault(),
) {
    private val _state = MutableStateFlow<TodayMoonUiState>(TodayMoonUiState.Loading)
    val state: StateFlow<TodayMoonUiState> = _state.asStateFlow()

    init {
        load()
    }

    fun load() {
        _state.value = TodayMoonUiState.Content(repository.getTodayMoon(zone))
    }
}

sealed interface TodayMoonUiState {
    data object Loading : TodayMoonUiState
    data class Content(val moon: TodayMoon) : TodayMoonUiState
}
