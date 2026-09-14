package com.runa.shared.feature.notification

import com.russhwolf.settings.Settings
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/** Default [NotificationSettingsRepository] backed by multiplatform-settings; every
 *  mutation re-issues the OS schedule through [LocalNotificationScheduler]. */
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
        // While off, changing the time only records the preference; the OS schedule is untouched.
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
