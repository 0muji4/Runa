package com.runa.shared.feature.lock

import androidx.fragment.app.FragmentActivity
import java.lang.ref.WeakReference

/**
 * Holds a weak reference to the currently-resumed [FragmentActivity] so the shared
 * [AndroidBiometricAuthenticator] can present a BiometricPrompt (which requires an
 * Activity, not just the app Context that Koin holds). The single Activity registers
 * itself from its lifecycle: set on ON_RESUME, clear on ON_PAUSE. Weak so a
 * destroyed Activity is never leaked.
 */
object CurrentActivityHolder {
    private var ref: WeakReference<FragmentActivity>? = null

    fun set(activity: FragmentActivity) {
        ref = WeakReference(activity)
    }

    fun clear(activity: FragmentActivity) {
        if (ref?.get() === activity) ref = null
    }

    fun current(): FragmentActivity? = ref?.get()
}
