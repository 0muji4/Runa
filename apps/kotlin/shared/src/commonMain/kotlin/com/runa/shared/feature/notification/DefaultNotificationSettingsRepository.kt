package com.runa.shared.feature.notification

import com.russhwolf.settings.Settings
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * Default [NotificationSettingsRepository] backed by multiplatform-settings and a
 * platform [LocalNotificationScheduler].
 *
 * The enabled flag and time are mirrored in in-memory [MutableStateFlow]s (seeded
 * from persisted values at construction, so the settings screen renders the saved
 * state with no flash) and written back on every change. This mirrors the theme /
 * gallery display-theme precedent. On every mutation the repository re-issues the
 * OS schedule so the on-device notification always matches the stored preference.
 */
class DefaultNotificationSettingsRepository(
    private val settings: Settings,
    private val scheduler: LocalNotificationScheduler,
) : NotificationSettingsRepository {

    private val _enabled = MutableStateFlow(settings.getBoolean(KEY_ENABLED, false))
    private val _time = MutableStateFlow(loadTime())

    override fun observeReminderEnabled(): StateFlow<Boolean> = _enabled.asStateFlow()
    override fun observeReminderTime(): StateFlow<ReminderTime> = _time.asStateFlow()

    override fun setReminderEnabled(enabled: Boolean) {
        settings.putBoolean(KEY_ENABLED, enabled)
        _enabled.value = enabled
        if (enabled) scheduler.scheduleDailyReminder(_time.value) else scheduler.cancel()
    }

    override fun setReminderTime(time: ReminderTime) {
        settings.putInt(KEY_HOUR, time.hour)
        settings.putInt(KEY_MINUTE, time.minute)
        _time.value = time
        // Only touch the OS schedule when the reminder is on; changing the time
        // while off just records the preference for when it's next enabled.
        if (_enabled.value) scheduler.scheduleDailyReminder(time)
    }

    private fun loadTime(): ReminderTime = ReminderTime.of(
        hour = settings.getInt(KEY_HOUR, ReminderTime.Default.hour),
        minute = settings.getInt(KEY_MINUTE, ReminderTime.Default.minute),
    )

    private companion object {
        const val KEY_ENABLED = "notif.reminder.enabled"
        const val KEY_HOUR = "notif.reminder.hour"
        const val KEY_MINUTE = "notif.reminder.minute"
    }
}
