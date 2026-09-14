package com.runa.android.ui.theme

import androidx.compose.runtime.Composable
import androidx.compose.runtime.Immutable
import androidx.compose.runtime.ReadOnlyComposable
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color
import com.runa.shared.feature.settings.AppTheme

/** Semantic color tokens; values are shared with iOS/README (drift-guarded by hack/). */
@Immutable
data class RunaColorScheme(
    val Background: Color,
    val Surface: Color,
    val Heading: Color,
    val Body: Color,
    val Subtle: Color,
    val Accent: Color,
    val SubAccent: Color,
)

/** 夜（ダーク・既定） */
val RunaDarkColors = RunaColorScheme(
    Background = Color(0xFF0E0E12),
    Surface = Color(0xFF16161C),
    Heading = Color(0xFFF5F3EF),
    Body = Color(0xFFC8C6CE),
    Subtle = Color(0xFF9A9AA5),
    Accent = Color(0xFFF4A9C0),
    SubAccent = Color(0xFFE8E2D0),
)

/** あさ（ライト） */
val RunaLightColors = RunaColorScheme(
    Background = Color(0xFFFAF7F5),
    Surface = Color(0xFFFFFFFF),
    Heading = Color(0xFF2A2620),
    Body = Color(0xFF4E483F),
    Subtle = Color(0xFF8C8579),
    Accent = Color(0xFFE79CB6),
    SubAccent = Color(0xFFC9B8A0),
)

/** ピンク×ピンク */
val RunaPinkColors = RunaColorScheme(
    Background = Color(0xFF141017),
    Surface = Color(0xFF1E1622),
    Heading = Color(0xFFF6EEF2),
    Body = Color(0xFFD6C4CE),
    Subtle = Color(0xFFA08E99),
    Accent = Color(0xFFF4A9C0),
    SubAccent = Color(0xFFE8B7C8),
)

/** Maps the shared [AppTheme] selection to its native color scheme. */
fun runaColorsFor(theme: AppTheme): RunaColorScheme = when (theme) {
    AppTheme.DARK -> RunaDarkColors
    AppTheme.LIGHT -> RunaLightColors
    AppTheme.PINK -> RunaPinkColors
}

/** Provided by [RunaTheme]. Static so a theme change recomposes the whole tree. */
val LocalRunaColors = staticCompositionLocalOf { RunaDarkColors }

/** Active theme's tokens; composable scope only (non-composable draw code holds its own constants). */
val RunaColors: RunaColorScheme
    @Composable @ReadOnlyComposable get() = LocalRunaColors.current
