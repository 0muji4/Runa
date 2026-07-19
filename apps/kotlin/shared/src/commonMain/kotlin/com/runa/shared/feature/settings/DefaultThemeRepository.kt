package com.runa.shared.feature.settings

import com.russhwolf.settings.Settings
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * Default [ThemeRepository] backed by multiplatform-settings.
 *
 * The current selection is mirrored in an in-memory [MutableStateFlow] (seeded
 * from the persisted value at construction, so startup applies the saved theme
 * with no flash) and written back on every change. This mirrors the gallery
 * display-theme precedent: persist the value, observe an in-memory flow — no
 * observable-settings dependency needed for a single, self-written key.
 */
class DefaultThemeRepository(
    private val settings: Settings,
) : ThemeRepository {

    private val _theme = MutableStateFlow(AppTheme.fromId(settings.getStringOrNull(KEY_THEME)))

    override fun observeTheme(): StateFlow<AppTheme> = _theme.asStateFlow()

    override fun setTheme(theme: AppTheme) {
        settings.putString(KEY_THEME, theme.id)
        _theme.value = theme
    }

    private companion object {
        const val KEY_THEME = "app.theme"
    }
}
