package com.runa.shared.feature.today

import com.runa.shared.core.state.SyncPhase
import com.runa.shared.core.state.UiState
import com.runa.shared.core.state.toAppError
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
 * [TodayRepository] and exposes the shared [UiState] Android collects directly and
 * iOS observes through SKIE. Runs an initial [load] on construction so the home
 * shows content without an explicit trigger.
 *
 * The home always has renderable content — the moon is computed locally and never
 * needs the network — so offline is carried as [SyncPhase.Offline] on
 * [UiState.Content] (a quiet banner over the cached quote/song), not a body-hiding
 * state. [UiState.Failure] is defensive only (the repository already falls back to
 * cache on a network failure).
 */
class HomeViewModel(
    private val repository: TodayRepository,
    private val scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Default),
) {
    private val _state = MutableStateFlow<UiState<Today>>(UiState.Loading)
    val state: StateFlow<UiState<Today>> = _state.asStateFlow()

    init {
        load()
    }

    /** (Re)load today for the current local date. Offline falls back to cache. */
    fun load() {
        scope.launch {
            _state.value = UiState.Loading
            _state.value = try {
                val zone = TimeZone.currentSystemDefault()
                val date = Clock.System.now().toLocalDateTime(zone).date
                val today = repository.getToday(date, zone)
                UiState.Content(today, if (today.isOffline) SyncPhase.Offline else SyncPhase.Idle)
            } catch (e: Exception) {
                UiState.Failure(e.toAppError())
            }
        }
    }
}
