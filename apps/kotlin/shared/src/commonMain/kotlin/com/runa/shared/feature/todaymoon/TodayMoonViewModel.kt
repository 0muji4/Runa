package com.runa.shared.feature.todaymoon

import androidx.lifecycle.ViewModel
import com.runa.shared.core.state.UiState
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.datetime.TimeZone

/** View model for 今日の月; the payload is pure and offline, so [load] is synchronous. */
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
