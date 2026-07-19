package com.runa.shared.feature.settings

import com.russhwolf.settings.MapSettings
import kotlin.test.Test
import kotlin.test.assertEquals

class ThemeRepositoryTest {

    @Test
    fun defaultsToDarkWhenNothingPersisted() {
        val repo = DefaultThemeRepository(MapSettings())
        assertEquals(AppTheme.DARK, repo.observeTheme().value)
    }

    @Test
    fun setThemePersistsAndIsRestoredByAFreshRepository() {
        val settings = MapSettings()

        DefaultThemeRepository(settings).setTheme(AppTheme.PINK)

        // A brand-new repository over the SAME settings restores the selection —
        // this is what a process restart exercises.
        val restored = DefaultThemeRepository(settings)
        assertEquals(AppTheme.PINK, restored.observeTheme().value)
    }

    @Test
    fun setThemeEmitsImmediately() {
        val repo = DefaultThemeRepository(MapSettings())
        repo.setTheme(AppTheme.LIGHT)
        assertEquals(AppTheme.LIGHT, repo.observeTheme().value)
    }

    @Test
    fun unknownPersistedValueFallsBackToDark() {
        val settings = MapSettings().apply { putString("app.theme", "chartreuse") }
        assertEquals(AppTheme.DARK, DefaultThemeRepository(settings).observeTheme().value)
    }
}
