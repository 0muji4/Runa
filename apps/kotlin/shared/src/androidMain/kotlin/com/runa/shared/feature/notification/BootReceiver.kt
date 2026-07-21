package com.runa.shared.feature.notification

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent

/**
 * Re-arms the daily reminder after a reboot (AlarmManager schedules do not survive
 * a device restart). A no-op when the reminder is disabled.
 */
class BootReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action == Intent.ACTION_BOOT_COMPLETED) {
            AndroidLocalNotificationScheduler.rescheduleFromPreferences(context)
        }
    }
}
