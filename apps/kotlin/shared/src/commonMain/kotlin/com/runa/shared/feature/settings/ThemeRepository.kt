package com.runa.shared.feature.settings

import kotlinx.coroutines.flow.StateFlow

/**
 * Owns the persisted app-theme selection. [observeTheme] is the single source of
 * truth the app root and the settings screens subscribe to; [setTheme] persists a
 * new choice and re-emits so the whole app recolors immediately.
 *
 * The selection is stored via multiplatform-settings, seeded synchronously at
 * construction so the correct theme is available before the first frame (no flash).
 */
interface ThemeRepository {
    fun observeTheme(): StateFlow<AppTheme>
    fun setTheme(theme: AppTheme)
}
