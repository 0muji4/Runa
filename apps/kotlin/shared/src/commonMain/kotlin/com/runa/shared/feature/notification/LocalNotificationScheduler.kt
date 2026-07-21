package com.runa.shared.feature.notification

/**
 * The platform local-notification seam. The shared
 * [NotificationSettingsRepository] owns the reminder PREFERENCE (enabled + time)
 * and drives this scheduler; the actual OS scheduling is a platform implementation
 * bound in [com.runa.shared.platform.platformModule] — AlarmManager + a
 * NotificationManager receiver on Android, UNUserNotificationCenter on iOS.
 *
 * Scheduling is fire-and-forget: [scheduleDailyReminder] (re)arms a daily
 * notification at the given local time (replacing any previous one), and [cancel]
 * removes it. Neither posts a notification immediately.
 */
interface LocalNotificationScheduler {
    /** (Re)arm a daily local reminder at [time], replacing any existing schedule. */
    fun scheduleDailyReminder(time: ReminderTime)

    /** Cancel the daily reminder, if any. Idempotent. */
    fun cancel()
}
