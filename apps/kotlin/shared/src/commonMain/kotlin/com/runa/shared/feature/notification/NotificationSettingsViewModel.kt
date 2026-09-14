package com.runa.shared.feature.notification

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.launch

/** Snapshot for the 通知設定 (21) screen: reminder on/off, the chosen time, and the preset chips. */
data class NotificationUiState(
    val enabled: Boolean,
    val time: ReminderTime,
    val presets: List<ReminderTime> = ReminderTime.Presets,
)

/** Drives the 通知設定 (21) screen. A plain [MutableStateFlow] fed by an init collector
 *  (not `stateIn` + `WhileSubscribed`) so `state.value` is always current without an active subscriber. */
class NotificationSettingsViewModel(
    private val repository: NotificationSettingsRepository,
) : ViewModel() {
    private val _state = MutableStateFlow(
        NotificationUiState(
            enabled = repository.observeReminderEnabled().value,
            time = repository.observeReminderTime().value,
        ),
    )
    val state: StateFlow<NotificationUiState> = _state.asStateFlow()

    init {
        viewModelScope.launch {
            combine(
                repository.observeReminderEnabled(),
                repository.observeReminderTime(),
            ) { enabled, time -> NotificationUiState(enabled = enabled, time = time) }
                .collect { _state.value = it }
        }
    }

    /** Synchronous current snapshot, so the iOS observable can seed without a flash. */
    fun currentState(): NotificationUiState = _state.value

    fun onToggle(enabled: Boolean) = repository.setReminderEnabled(enabled)

    fun onSelectTime(time: ReminderTime) = repository.setReminderTime(time)
}
