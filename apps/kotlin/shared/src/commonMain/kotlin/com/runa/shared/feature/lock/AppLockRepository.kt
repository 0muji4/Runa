package com.runa.shared.feature.lock

import kotlinx.coroutines.flow.StateFlow

/** Owns the persisted privacy-lock preference. [observeLockEnabled] must be seeded
 *  synchronously at construction so the first frame is correct (no flash of unlocked content). */
interface AppLockRepository {
    fun observeLockEnabled(): StateFlow<Boolean>

    fun setLockEnabled(enabled: Boolean)
}
