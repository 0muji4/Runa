package com.runa.shared.feature.notification

import kotlinx.coroutines.flow.StateFlow

/** Owns the persisted nightly-reminder preference (on/off + time); the setters also
 *  keep the OS schedule in sync through [LocalNotificationScheduler]. */
interface NotificationSettingsRepository {
    fun observeReminderEnabled(): StateFlow<Boolean>
    fun observeReminderTime(): StateFlow<ReminderTime>

    fun setReminderEnabled(enabled: Boolean)

    fun setReminderTime(time: ReminderTime)
}
