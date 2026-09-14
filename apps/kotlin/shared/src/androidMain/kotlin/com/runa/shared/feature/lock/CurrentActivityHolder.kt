package com.runa.shared.feature.lock

import androidx.fragment.app.FragmentActivity
import java.lang.ref.WeakReference

/**
 * Weak reference to the currently-resumed [FragmentActivity] for [AndroidBiometricAuthenticator].
 * The Activity sets it on ON_RESUME and clears it on ON_PAUSE.
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
