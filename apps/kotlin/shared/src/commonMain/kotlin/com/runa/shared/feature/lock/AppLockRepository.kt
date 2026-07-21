package com.runa.shared.feature.lock

import kotlinx.coroutines.flow.StateFlow

/**
 * Owns the persisted privacy-lock preference. When enabled, the app requires a
 * successful biometric (Face ID / BiometricPrompt) authentication — with the
 * device passcode as fallback — before revealing content on launch/resume.
 *
 * [observeLockEnabled] is the single source of truth the lock gate and the 22
 * プライバシー・ロック screen subscribe to. Persisted via multiplatform-settings and
 * seeded synchronously at construction so the first frame is correct (no flash of
 * unlocked content when the lock is on).
 */
interface AppLockRepository {
    fun observeLockEnabled(): StateFlow<Boolean>

    /** Persist the on/off choice and re-emit. */
    fun setLockEnabled(enabled: Boolean)
}
