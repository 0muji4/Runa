package com.runa.shared.feature.push

import android.content.Context
import com.google.firebase.FirebaseApp
import com.google.firebase.FirebaseOptions

/** Manual Firebase bootstrap (no google-services plugin); must run before Koin so the token fetch finds the app. */
object RunaFirebase {

    /** Returns false (and does nothing) when any option is blank or an app already exists. */
    fun initialize(
        context: Context,
        projectId: String,
        applicationId: String,
        apiKey: String,
        senderId: String,
    ): Boolean {
        if (projectId.isBlank() || applicationId.isBlank() || apiKey.isBlank() || senderId.isBlank()) return false
        if (FirebaseApp.getApps(context).isNotEmpty()) return false
        val options = FirebaseOptions.Builder()
            .setProjectId(projectId)
            .setApplicationId(applicationId)
            .setApiKey(apiKey)
            .setGcmSenderId(senderId)
            .build()
        FirebaseApp.initializeApp(context, options)
        return true
    }
}
