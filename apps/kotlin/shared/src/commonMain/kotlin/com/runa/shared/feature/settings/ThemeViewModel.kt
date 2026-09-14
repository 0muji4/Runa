package com.runa.shared.feature.settings

import androidx.lifecycle.ViewModel
import kotlinx.coroutines.flow.StateFlow

/** Drives the theme picker (screen 20) and feeds the app root the active theme. */
class ThemeViewModel(
    private val repository: ThemeRepository,
) : ViewModel() {
    val theme: StateFlow<AppTheme> = repository.observeTheme()

    fun select(theme: AppTheme) {
        repository.setTheme(theme)
    }

    /** Synchronous current theme id, so iOS can seed its observable flash-free. */
    fun currentThemeId(): String = theme.value.id

    /** Select by the stable [AppTheme.id]; lets iOS avoid depending on the bridged enum's case names. */
    fun selectId(id: String) {
        repository.setTheme(AppTheme.fromId(id))
    }
}
