package com.runa.shared.feature.settings

import kotlinx.coroutines.flow.StateFlow

/**
 * Backs the settings top screen (19). Its only shared state is the active theme,
 * shown as the trailing value on the "テーマ" row; the app version is read natively
 * per client, and the notification / privacy-lock rows are placeholders for a later
 * feature. Kept as its own view model (rather than reusing [ThemeViewModel]) so the
 * settings-top screen resolves one dedicated holder, matching the app's screen↔VM
 * convention.
 */
class SettingsViewModel(
    themeRepository: ThemeRepository,
) {
    val theme: StateFlow<AppTheme> = themeRepository.observeTheme()
}
