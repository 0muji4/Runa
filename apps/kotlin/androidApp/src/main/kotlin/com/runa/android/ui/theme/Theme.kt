package com.runa.android.ui.theme

import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.runtime.Composable

/**
 * Runa theme. Dark ONLY by design — there is no light variant. The color scheme
 * is derived directly from [RunaColors] so Android and iOS stay in lockstep.
 */
private val RunaDarkColorScheme = darkColorScheme(
    primary = RunaColors.Accent,
    onPrimary = RunaColors.Background,
    secondary = RunaColors.SubAccent,
    onSecondary = RunaColors.Background,
    background = RunaColors.Background,
    onBackground = RunaColors.Body,
    surface = RunaColors.Surface,
    onSurface = RunaColors.Body,
    surfaceVariant = RunaColors.Surface,
    onSurfaceVariant = RunaColors.Subtle,
)

@Composable
fun RunaTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = RunaDarkColorScheme,
        typography = RunaTypography,
        content = content,
    )
}
