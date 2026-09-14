package com.runa.shared.feature.lock

import com.russhwolf.settings.MapSettings
import kotlin.test.Test
import kotlin.test.assertEquals

class AppLockRepositoryTest {

    @Test
    fun defaultsToDisabled() {
        assertEquals(false, DefaultAppLockRepository(MapSettings()).observeLockEnabled().value)
    }

    @Test
    fun setEnabledPersistsAndIsRestoredByAFreshRepository() {
        val settings = MapSettings()

        DefaultAppLockRepository(settings).setLockEnabled(true)

        // A new repository over the SAME settings models a process restart.
        assertEquals(true, DefaultAppLockRepository(settings).observeLockEnabled().value)
    }

    @Test
    fun setEnabledEmitsImmediately() {
        val repo = DefaultAppLockRepository(MapSettings())
        repo.setLockEnabled(true)
        assertEquals(true, repo.observeLockEnabled().value)
    }
}
