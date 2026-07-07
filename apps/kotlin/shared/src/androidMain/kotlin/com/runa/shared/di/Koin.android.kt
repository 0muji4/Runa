package com.runa.shared.di

import android.content.Context
import com.runa.shared.platform.platformModule
import org.koin.android.ext.koin.androidContext
import org.koin.core.context.startKoin

/**
 * Android DI entry point. Unlike the common [initKoin] it also supplies an
 * `androidContext(...)`, which the Android [platformModule] needs to build the
 * EncryptedSharedPreferences-backed secure store.
 *
 * Called once from `RunaApplication.onCreate()`.
 *
 * @param context the application Context.
 * @param baseUrl host+port only (e.g. http://10.0.2.2:8080), no /api/v1 suffix.
 */
fun initKoin(context: Context, baseUrl: String) {
    startKoin {
        androidContext(context.applicationContext)
        modules(sharedModule(baseUrl), platformModule())
    }
}
