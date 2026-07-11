package com.runa.shared.feature.today

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.datetime.Clock
import kotlinx.datetime.TimeZone
import kotlinx.datetime.toLocalDateTime

/**
 * Shared home view model. Composes the day's quote + moon + song via
 * [TodayRepository] and exposes a [StateFlow] Android collects directly and iOS
 * observes through SKIE. Runs an initial [load] on construction so the home shows
 * content without an explicit trigger (mirrors [com.runa.shared.feature.health.HealthzViewModel]).
 */
class HomeViewModel(
    private val repository: TodayRepository,
    private val scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Default),
) {
    private val _state = MutableStateFlow<HomeUiState>(HomeUiState.Loading)
    val state: StateFlow<HomeUiState> = _state.asStateFlow()

    init {
        load()
    }

    /** (Re)load today for the current local date. Offline falls back to cache. */
    fun load() {
        scope.launch {
            _state.value = HomeUiState.Loading
            _state.value = try {
                val zone = TimeZone.currentSystemDefault()
                val date = Clock.System.now().toLocalDateTime(zone).date
                val today = repository.getToday(date, zone)
                if (today.isOffline) HomeUiState.Offline(today) else HomeUiState.Content(today)
            } catch (e: Exception) {
                HomeUiState.Error(e.message ?: "unknown error")
            }
        }
    }
}
