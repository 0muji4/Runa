package com.runa.shared.feature.todaymoon

import androidx.lifecycle.ViewModel
import com.runa.shared.core.state.UiState
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.datetime.TimeZone

/**
 * Shared view model for 15 今日の月. The payload is a pure, offline computation, so
 * [load] is synchronous and [state] settles to [UiState.Content] at once (a
 * [UiState.Failure] never occurs — the moon needs no network). Android collects
 * [state]; iOS observes it through SKIE.
 */
class TodayMoonViewModel(
    private val repository: TodayMoonRepository,
    private val zone: TimeZone = TimeZone.currentSystemDefault(),
) : ViewModel() {
    private val _state = MutableStateFlow<UiState<TodayMoon>>(UiState.Loading)
    val state: StateFlow<UiState<TodayMoon>> = _state.asStateFlow()

    init {
        load()
    }

    fun load() {
        _state.value = UiState.Content(repository.getTodayMoon(zone))
    }
}
