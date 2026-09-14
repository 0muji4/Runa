package com.runa.shared.feature.settings

import androidx.lifecycle.ViewModel
import kotlinx.coroutines.flow.StateFlow

/** Backs the settings top screen (19); its only shared state is the active theme. */
class SettingsViewModel(
    themeRepository: ThemeRepository,
) : ViewModel() {
    val theme: StateFlow<AppTheme> = themeRepository.observeTheme()
}
