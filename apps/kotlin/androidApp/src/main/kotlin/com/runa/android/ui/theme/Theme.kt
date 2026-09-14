package com.runa.android.ui.theme

import android.app.Activity
import androidx.compose.material3.ColorScheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.SideEffect
import androidx.compose.ui.platform.LocalView
import androidx.core.view.WindowCompat
import com.runa.shared.feature.settings.AppTheme

/** Provides [LocalRunaColors] for [theme] and derives the Material3 [ColorScheme] from it. */
@Composable
fun RunaTheme(
    theme: AppTheme = AppTheme.DARK,
    content: @Composable () -> Unit,
) {
    val colors = runaColorsFor(theme)

    val view = LocalView.current
    if (!view.isInEditMode) {
        SideEffect {
            val window = (view.context as Activity).window
            WindowCompat.getInsetsController(window, view).isAppearanceLightStatusBars =
                theme == AppTheme.LIGHT
        }
    }

    CompositionLocalProvider(LocalRunaColors provides colors) {
        MaterialTheme(
            colorScheme = materialSchemeFor(theme, colors),
            typography = RunaTypography,
            content = content,
        )
    }
}

/** Light uses [lightColorScheme] so unmapped roles get light defaults; dark and pink share dark. */
private fun materialSchemeFor(theme: AppTheme, c: RunaColorScheme): ColorScheme =
    if (theme == AppTheme.LIGHT) {
        lightColorScheme(
            primary = c.Accent,
            onPrimary = c.Heading,
            secondary = c.SubAccent,
            onSecondary = c.Heading,
            background = c.Background,
            onBackground = c.Body,
            surface = c.Surface,
            onSurface = c.Body,
            surfaceVariant = c.Surface,
            onSurfaceVariant = c.Subtle,
        )
    } else {
        darkColorScheme(
            primary = c.Accent,
            onPrimary = c.Background,
            secondary = c.SubAccent,
            onSecondary = c.Background,
            background = c.Background,
            onBackground = c.Body,
            surface = c.Surface,
            onSurface = c.Body,
            surfaceVariant = c.Surface,
            onSurfaceVariant = c.Subtle,
        )
    }
