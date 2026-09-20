package com.runa.android

import android.app.Application
import com.runa.shared.di.initKoin
import com.runa.shared.feature.push.RunaFirebase

/** BASE_URL is host+port only; the shared module owns the /api/v1 prefix. */
class RunaApplication : Application() {
    override fun onCreate() {
        super.onCreate()
        // Before initKoin: the eager FCM token fetch looks for an initialized FirebaseApp.
        RunaFirebase.initialize(
            context = this,
            projectId = BuildConfig.FIREBASE_PROJECT_ID,
            applicationId = BuildConfig.FIREBASE_APP_ID,
            apiKey = BuildConfig.FIREBASE_API_KEY,
            senderId = BuildConfig.FIREBASE_SENDER_ID,
        )
        initKoin(this, BuildConfig.BASE_URL)
    }
}
