package com.runa.shared.feature.notification

/** The platform local-notification seam, bound in [com.runa.shared.platform.platformModule].
 *  Fire-and-forget: neither method posts a notification immediately. */
interface LocalNotificationScheduler {
    /** (Re)arm a daily local reminder at [time], replacing any existing schedule. */
    fun scheduleDailyReminder(time: ReminderTime)

    /** Cancel the daily reminder, if any. Idempotent. */
    fun cancel()
}
