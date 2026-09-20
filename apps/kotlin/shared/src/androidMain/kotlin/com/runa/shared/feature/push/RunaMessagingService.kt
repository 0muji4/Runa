package com.runa.shared.feature.push

import android.Manifest
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.content.ContextCompat
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import com.runa.shared.R
import com.runa.shared.feature.diary.DiaryRepository
import com.runa.shared.feature.notification.ReminderNotificationText
import com.runa.shared.network.auth.TokenStore
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.runBlocking
import kotlinx.datetime.Instant
import kotlinx.datetime.TimeZone
import kotlinx.datetime.toLocalDateTime
import org.koin.core.Koin
import org.koin.core.context.GlobalContext

/**
 * Receives the server's data-only reminder ({kind, date, title, body}) and composes the notification.
 * Dropped when signed out or when the local DB already holds an entry for [date].
 */
class RunaMessagingService : FirebaseMessagingService() {

    override fun onNewToken(token: String) {
        koinOrNull()?.get<PushTokenStore>()?.set(token)
    }

    override fun onMessageReceived(message: RemoteMessage) {
        val data = message.data
        if (data["kind"] != KIND_DIARY_REMINDER) return
        val koin = koinOrNull() ?: return
        if (koin.get<TokenStore>().load() == null) return
        val date = data["date"]
        if (date != null && hasEntryOn(koin.get(), date)) return
        post(
            title = data["title"].orEmptyToNull() ?: ReminderNotificationText.TITLE,
            body = data["body"].orEmptyToNull() ?: ReminderNotificationText.BODY,
        )
    }

    // A failed local read must not veto the reminder (the server already judged) nor crash the process.
    private fun hasEntryOn(repository: DiaryRepository, isoDate: String): Boolean = runCatching {
        runBlocking {
            val zone = TimeZone.currentSystemDefault()
            repository.observeEntries().first().any { entry ->
                Instant.fromEpochMilliseconds(entry.createdAtEpochMs).toLocalDateTime(zone).date.toString() == isoDate
            }
        }
    }.getOrDefault(false)

    private fun post(title: String, body: String) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU &&
            ContextCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS) !=
            PackageManager.PERMISSION_GRANTED
        ) {
            return
        }
        ensureChannel()
        val open = Intent(Intent.ACTION_VIEW, Uri.parse(EDITOR_URI))
            .setPackage(packageName)
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_SINGLE_TOP)
        val contentIntent = PendingIntent.getActivity(
            this,
            CONTENT_REQUEST_CODE,
            open,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val notification = NotificationCompat.Builder(this, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_reminder_moon)
            .setContentTitle(title)
            .setContentText(body)
            .setPriority(NotificationCompat.PRIORITY_DEFAULT)
            .setAutoCancel(true)
            .setContentIntent(contentIntent)
            .build()
        NotificationManagerCompat.from(this).notify(NOTIFICATION_ID, notification)
    }

    private fun ensureChannel() {
        val channel = NotificationChannel(
            CHANNEL_ID,
            "夜のリマインダー",
            NotificationManager.IMPORTANCE_DEFAULT,
        ).apply { description = "静かに綴る時間のお知らせ" }
        getSystemService(NotificationManager::class.java).createNotificationChannel(channel)
    }

    private fun koinOrNull(): Koin? = GlobalContext.getOrNull()

    private fun String?.orEmptyToNull(): String? = this?.takeIf { it.isNotBlank() }

    private companion object {
        const val KIND_DIARY_REMINDER = "diary_reminder"
        const val EDITOR_URI = "runa://diary/editor"
        const val CHANNEL_ID = "runa.reminder.nightly"
        const val NOTIFICATION_ID = 4201
        const val CONTENT_REQUEST_CODE = 4203
    }
}
