package com.runa.shared.feature.settings

import kotlinx.coroutines.flow.StateFlow

/** Owns the persisted app-theme selection. [observeTheme] must be seeded synchronously
 *  at construction so the correct theme is available before the first frame. */
interface ThemeRepository {
    fun observeTheme(): StateFlow<AppTheme>
    fun setTheme(theme: AppTheme)
}
