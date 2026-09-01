package com.runa.shared.feature.settings

import androidx.lifecycle.ViewModel
import kotlinx.coroutines.flow.StateFlow

/**
 * Drives the theme picker (screen 20) and feeds the app root the active theme so a
 * change recolors every screen at once. [theme] re-exports the repository's
 * StateFlow verbatim; [select] persists a new choice (applied immediately).
 *
 * Nothing here needs [androidx.lifecycle.viewModelScope]: persistence is synchronous
 * and observation is a plain StateFlow, so this view model holds no long-lived
 * resources of its own.
 */
class ThemeViewModel(
    private val repository: ThemeRepository,
) : ViewModel() {
    val theme: StateFlow<AppTheme> = repository.observeTheme()

    fun select(theme: AppTheme) {
        repository.setTheme(theme)
    }

    /** Synchronous current theme id for a flash-free initial read (iOS seeds its
     *  observable with this before the flow starts emitting). */
    fun currentThemeId(): String = theme.value.id

    /** Select by the stable [AppTheme.id] string. Lets the iOS client drive theme
     *  selection without depending on the bridged enum's case names. */
    fun selectId(id: String) {
        repository.setTheme(AppTheme.fromId(id))
    }
}
