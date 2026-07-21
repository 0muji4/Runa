package com.runa.shared.feature.notification

import kotlinx.coroutines.flow.StateFlow

/**
 * Owns the persisted nightly-reminder preference (on/off + time) and keeps the OS
 * schedule in sync with it. [observeReminderEnabled] / [observeReminderTime] are
 * the single source of truth the settings screen subscribes to; the setters
 * persist the new value, re-emit, AND instruct the platform
 * [LocalNotificationScheduler] (schedule on enable / at a new time, cancel on
 * disable), so a preference change is reflected in the actual OS notification.
 *
 * Values are stored via multiplatform-settings, seeded synchronously at
 * construction so the settings screen shows the saved state with no flash.
 */
interface NotificationSettingsRepository {
    fun observeReminderEnabled(): StateFlow<Boolean>
    fun observeReminderTime(): StateFlow<ReminderTime>

    /** Persist the on/off choice, re-emit, and schedule or cancel accordingly. */
    fun setReminderEnabled(enabled: Boolean)

    /** Persist the reminder time, re-emit, and (when enabled) reschedule. */
    fun setReminderTime(time: ReminderTime)
}
