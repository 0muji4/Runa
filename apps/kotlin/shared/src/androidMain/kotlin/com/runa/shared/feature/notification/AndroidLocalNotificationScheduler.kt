package com.runa.shared.feature.notification

import android.app.AlarmManager
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import com.runa.shared.R
import java.util.Calendar

/**
 * Android [LocalNotificationScheduler]. Arms a daily local reminder with
 * AlarmManager and posts it from [ReminderReceiver]. Uses `setAndAllowWhileIdle`
 * (inexact, allow-while-idle) rather than an exact alarm, so no
 * `SCHEDULE_EXACT_ALARM` permission is needed — a gentle nightly nudge does not
 * require to-the-second precision. The alarm fires once; the receiver re-arms the
 * next day (and [BootReceiver] re-arms after a reboot), which is why the scheduler
 * exposes [rescheduleFromPreferences] reading the same `runa_settings` store the
 * repository writes.
 */
class AndroidLocalNotificationScheduler(
    private val context: Context,
) : LocalNotificationScheduler {

    override fun scheduleDailyReminder(time: ReminderTime) {
        ensureChannel(context)
        val alarm = context.getSystemService(Context.ALARM_SERVICE) as AlarmManager
        alarm.setAndAllowWhileIdle(
            AlarmManager.RTC_WAKEUP,
            nextTriggerMillis(time.hour, time.minute),
            reminderPendingIntent(context),
        )
    }

    override fun cancel() {
        val alarm = context.getSystemService(Context.ALARM_SERVICE) as AlarmManager
        alarm.cancel(reminderPendingIntent(context))
    }

    companion object {
        private const val CHANNEL_ID = "runa.reminder.nightly"
        private const val NOTIFICATION_ID = 4201
        private const val ALARM_REQUEST_CODE = 4202
        private const val CONTENT_REQUEST_CODE = 4203

        // Same store + keys the DefaultNotificationSettingsRepository persists to, so
        // a receiver (alarm fire / boot) can re-arm without the shared graph.
        private const val PREFS_NAME = "runa_settings"
        private const val KEY_ENABLED = "notif.reminder.enabled"
        private const val KEY_HOUR = "notif.reminder.hour"
        private const val KEY_MINUTE = "notif.reminder.minute"

        /** Re-arm the daily reminder from the persisted preference (used by the alarm
         *  receiver to schedule the next day and by the boot receiver after reboot).
         *  A no-op when the reminder is disabled. */
        fun rescheduleFromPreferences(context: Context) {
            val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            if (!prefs.getBoolean(KEY_ENABLED, false)) return
            val time = ReminderTime.of(
                hour = prefs.getInt(KEY_HOUR, ReminderTime.Default.hour),
                minute = prefs.getInt(KEY_MINUTE, ReminderTime.Default.minute),
            )
            AndroidLocalNotificationScheduler(context).scheduleDailyReminder(time)
        }

        /** Build + post the reminder notification. Silently a no-op if the user has
         *  not granted POST_NOTIFICATIONS (API 33+) — the reminder never crashes. */
        fun postReminder(context: Context) {
            ensureChannel(context)
            val launch = context.packageManager.getLaunchIntentForPackage(context.packageName)
            val contentIntent = launch?.let {
                PendingIntent.getActivity(context, CONTENT_REQUEST_CODE, it, immutableFlags())
            }
            val notification = NotificationCompat.Builder(context, CHANNEL_ID)
                .setSmallIcon(R.drawable.ic_reminder_moon)
                .setContentTitle(ReminderNotificationText.TITLE)
                .setContentText(ReminderNotificationText.BODY)
                .setPriority(NotificationCompat.PRIORITY_DEFAULT)
                .setAutoCancel(true)
                .apply { contentIntent?.let { setContentIntent(it) } }
                .build()
            NotificationManagerCompat.from(context).notify(NOTIFICATION_ID, notification)
        }

        private fun reminderPendingIntent(context: Context): PendingIntent {
            val intent = Intent(context, ReminderReceiver::class.java).setAction(ReminderReceiver.ACTION_REMIND)
            return PendingIntent.getBroadcast(context, ALARM_REQUEST_CODE, intent, immutableFlags())
        }

        private fun immutableFlags(): Int =
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE

        private fun ensureChannel(context: Context) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                "夜のリマインド",
                NotificationManager.IMPORTANCE_DEFAULT,
            ).apply { description = "静かに綴る時間のお知らせ" }
            val manager = context.getSystemService(NotificationManager::class.java)
            manager.createNotificationChannel(channel)
        }

        private fun nextTriggerMillis(hour: Int, minute: Int): Long {
            val now = Calendar.getInstance()
            val next = Calendar.getInstance().apply {
                set(Calendar.HOUR_OF_DAY, hour)
                set(Calendar.MINUTE, minute)
                set(Calendar.SECOND, 0)
                set(Calendar.MILLISECOND, 0)
            }
            if (!next.after(now)) next.add(Calendar.DAY_OF_YEAR, 1)
            return next.timeInMillis
        }
    }
}
