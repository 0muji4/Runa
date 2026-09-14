package com.runa.shared.feature.today

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.runa.shared.core.state.SyncPhase
import com.runa.shared.core.state.UiState
import com.runa.shared.core.state.toAppError
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.datetime.Clock
import kotlinx.datetime.TimeZone
import kotlinx.datetime.toLocalDateTime

/**
 * Home view model: loads today's quote + moon + song. Offline is carried as
 * [SyncPhase.Offline] on [UiState.Content]; [UiState.Failure] is defensive only.
 */
class HomeViewModel(
    private val repository: TodayRepository,
) : ViewModel() {
    private val _state = MutableStateFlow<UiState<Today>>(UiState.Loading)
    val state: StateFlow<UiState<Today>> = _state.asStateFlow()

    init {
        load()
    }

    fun load() {
        viewModelScope.launch {
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
