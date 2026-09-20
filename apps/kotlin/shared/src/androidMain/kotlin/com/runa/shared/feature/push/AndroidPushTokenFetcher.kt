package com.runa.shared.feature.push

import android.content.Context
import com.google.firebase.FirebaseApp
import com.google.firebase.messaging.FirebaseMessaging

/** Seeds [PushTokenStore] with the current FCM token at startup; rotations arrive via [RunaMessagingService]. */
class AndroidPushTokenFetcher(
    private val context: Context,
    private val store: PushTokenStore,
) {
    fun start() {
        if (FirebaseApp.getApps(context).isEmpty()) return
        FirebaseMessaging.getInstance().token.addOnSuccessListener { store.set(it) }
    }
}
