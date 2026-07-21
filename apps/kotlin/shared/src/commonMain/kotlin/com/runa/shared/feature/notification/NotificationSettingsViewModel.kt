package com.runa.shared.feature.notification

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.launch

/**
 * Immutable snapshot for the 通知設定 (21) screen: whether the nightly reminder is
 * on, the chosen time, and the preset chips to offer. Independent fields (not a
 * sealed hierarchy) because the toggle and the time coexist on one screen.
 */
data class NotificationUiState(
    val enabled: Boolean,
    val time: ReminderTime,
    val presets: List<ReminderTime> = ReminderTime.Presets,
)

/**
 * Drives the 通知設定 (21) screen. Thin over [NotificationSettingsRepository]: it
 * folds the enabled + time streams into one [NotificationUiState] and forwards the
 * toggle / time-selection intents. The repository handles persistence AND the OS
 * schedule, so this view model stays UI-only. Android collects [state] directly;
 * iOS observes it via SKIE.
 *
 * A plain [MutableStateFlow] fed by an init collector (rather than `stateIn` with
 * `WhileSubscribed`) so `state.value` is always current for the platform observers
 * and tests, without needing an active subscriber.
 */
class NotificationSettingsViewModel(
    private val repository: NotificationSettingsRepository,
    scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Default),
) {
    private val _state = MutableStateFlow(
        NotificationUiState(
            enabled = repository.observeReminderEnabled().value,
            time = repository.observeReminderTime().value,
        ),
    )
    val state: StateFlow<NotificationUiState> = _state.asStateFlow()

    init {
        scope.launch {
            combine(
                repository.observeReminderEnabled(),
                repository.observeReminderTime(),
            ) { enabled, time -> NotificationUiState(enabled = enabled, time = time) }
                .collect { _state.value = it }
        }
    }

    /** Synchronous current snapshot, so the iOS observable can seed without a flash. */
    fun currentState(): NotificationUiState = _state.value

    /** Turn the nightly reminder on/off. */
    fun onToggle(enabled: Boolean) = repository.setReminderEnabled(enabled)

    /** Choose a reminder time (a preset chip or a free picker value). */
    fun onSelectTime(time: ReminderTime) = repository.setReminderTime(time)
}
