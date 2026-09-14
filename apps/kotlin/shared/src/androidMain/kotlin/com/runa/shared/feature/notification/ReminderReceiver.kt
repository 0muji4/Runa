package com.runa.shared.feature.notification

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent

/** Posts the reminder when the alarm fires and re-arms it for the next day (one-shots don't repeat). */
class ReminderReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        AndroidLocalNotificationScheduler.postReminder(context)
        AndroidLocalNotificationScheduler.rescheduleFromPreferences(context)
    }

    companion object {
        const val ACTION_REMIND = "com.runa.shared.action.NIGHTLY_REMIND"
    }
}
