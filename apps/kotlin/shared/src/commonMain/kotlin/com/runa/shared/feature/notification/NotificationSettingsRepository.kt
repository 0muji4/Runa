package com.runa.shared.feature.notification

import kotlinx.coroutines.flow.StateFlow

/** Owns the persisted nightly-reminder preference (on/off + time); the server schedules the
 *  push from what [com.runa.shared.feature.push.DeviceRegistrar] registers. */
interface NotificationSettingsRepository {
    fun observeReminderEnabled(): StateFlow<Boolean>
    fun observeReminderTime(): StateFlow<ReminderTime>

    fun setReminderEnabled(enabled: Boolean)

    fun setReminderTime(time: ReminderTime)
}
