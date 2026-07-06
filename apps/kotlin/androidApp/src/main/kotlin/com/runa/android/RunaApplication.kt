package com.runa.android

import android.app.Application
import com.runa.shared.di.initKoin

/**
 * Process entry point. Starts Koin once with the platform's dev base URL.
 *
 * BASE_URL is host+port only (http://10.0.2.2:8080); the shared module owns the
 * /api/v1 prefix.
 */
class RunaApplication : Application() {
    override fun onCreate() {
        super.onCreate()
        initKoin(BuildConfig.BASE_URL)
    }
}
