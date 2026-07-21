package com.runa.shared.feature.notification

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent

/**
 * Fires when the daily reminder alarm goes off: posts the notification and re-arms
 * the alarm for the next day (AlarmManager one-shots don't repeat). Reading the
 * time from the persisted preference keeps the reschedule correct even if the
 * process was killed between fires.
 */
class ReminderReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        AndroidLocalNotificationScheduler.postReminder(context)
        AndroidLocalNotificationScheduler.rescheduleFromPreferences(context)
    }

    companion object {
        const val ACTION_REMIND = "com.runa.shared.action.NIGHTLY_REMIND"
    }
}
