package com.runa.shared.feature.notification

import platform.Foundation.NSDateComponents
import platform.UserNotifications.UNCalendarNotificationTrigger
import platform.UserNotifications.UNMutableNotificationContent
import platform.UserNotifications.UNNotificationRequest
import platform.UserNotifications.UNUserNotificationCenter

/**
 * iOS [LocalNotificationScheduler] over UNUserNotificationCenter. A single repeating
 * daily calendar trigger (hour + minute, repeats = true) is OS-managed — it
 * survives relaunch/reboot with no receiver, unlike Android's AlarmManager. Using a
 * fixed identifier means (re)scheduling replaces the previous one. Authorization is
 * requested separately in the app's onboarding (④); scheduling before authorization
 * simply won't display until granted.
 */
class IosLocalNotificationScheduler : LocalNotificationScheduler {

    private val center get() = UNUserNotificationCenter.currentNotificationCenter()

    override fun scheduleDailyReminder(time: ReminderTime) {
        cancel()

        val content = UNMutableNotificationContent().apply {
            setTitle(ReminderNotificationText.TITLE)
            setBody(ReminderNotificationText.BODY)
        }

        val components = NSDateComponents().apply {
            hour = time.hour.toLong()
            minute = time.minute.toLong()
        }
        val trigger = UNCalendarNotificationTrigger.triggerWithDateMatchingComponents(
            dateComponents = components,
            repeats = true,
        )

        val request = UNNotificationRequest.requestWithIdentifier(
            identifier = REMINDER_ID,
            content = content,
            trigger = trigger,
        )
        center.addNotificationRequest(request, withCompletionHandler = null)
    }

    override fun cancel() {
        center.removePendingNotificationRequestsWithIdentifiers(listOf(REMINDER_ID))
    }

    private companion object {
        const val REMINDER_ID = "runa.reminder.nightly"
    }
}
