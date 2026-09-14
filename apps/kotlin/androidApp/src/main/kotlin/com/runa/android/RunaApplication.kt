package com.runa.android

import android.app.Application
import com.runa.shared.di.initKoin

/** BASE_URL is host+port only; the shared module owns the /api/v1 prefix. */
class RunaApplication : Application() {
    override fun onCreate() {
        super.onCreate()
        initKoin(this, BuildConfig.BASE_URL)
    }
}
