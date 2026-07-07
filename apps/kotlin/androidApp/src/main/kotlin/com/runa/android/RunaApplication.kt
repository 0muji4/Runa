package com.runa.android

import android.app.Application
import com.runa.shared.di.initKoin

/**
 * Process entry point. Starts Koin once with the app Context and the platform's
 * dev base URL. The Context is required by the shared secure store
 * (EncryptedSharedPreferences), hence the two-arg Android [initKoin] overload.
 *
 * BASE_URL is host+port only (http://10.0.2.2:8080); the shared module owns the
 * /api/v1 prefix.
 */
class RunaApplication : Application() {
    override fun onCreate() {
        super.onCreate()
        initKoin(this, BuildConfig.BASE_URL)
    }
}
