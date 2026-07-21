package com.runa.shared.feature.lock

import com.russhwolf.settings.Settings
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * Default [AppLockRepository] backed by multiplatform-settings. The enabled flag
 * is mirrored in an in-memory [MutableStateFlow] seeded from the persisted value
 * at construction (so the lock gate can decide before the first frame) and written
 * back on change. Same persist-and-observe shape as the theme repository.
 *
 * The flag is not sensitive (biometric enrolment and the device credential live in
 * the OS), so plain [Settings] is used — not the encrypted token store.
 */
class DefaultAppLockRepository(
    private val settings: Settings,
) : AppLockRepository {

    private val _enabled = MutableStateFlow(settings.getBoolean(KEY_ENABLED, false))

    override fun observeLockEnabled(): StateFlow<Boolean> = _enabled.asStateFlow()

    override fun setLockEnabled(enabled: Boolean) {
        settings.putBoolean(KEY_ENABLED, enabled)
        _enabled.value = enabled
    }

    private companion object {
        const val KEY_ENABLED = "lock.enabled"
    }
}
