package com.runa.android.ui.theme

import androidx.compose.runtime.Composable
import androidx.compose.runtime.Immutable
import androidx.compose.runtime.ReadOnlyComposable
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color
import com.runa.shared.feature.settings.AppTheme

/**
 * Runa semantic color tokens. The seven roles are the single source of truth every
 * screen references (never a raw hex); a theme change swaps the whole scheme.
 *
 * Values are IDENTICAL to the iOS `RunaColors` and to the canonical table in the
 * repo README — the three clients must not drift. Property names are capitalized so
 * existing call sites (`RunaColors.Background`) keep compiling unchanged after the
 * move from a static `object` to this injected scheme.
 */
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

/** 夜（ダーク・既定）— the look the app was designed around first. */
val RunaDarkColors = RunaColorScheme(
    Background = Color(0xFF0E0E12),
    Surface = Color(0xFF16161C),
    Heading = Color(0xFFF5F3EF),
    Body = Color(0xFFC8C6CE),
    Subtle = Color(0xFF9A9AA5),
    Accent = Color(0xFFF4A9C0),
    SubAccent = Color(0xFFE8E2D0),
)

/** あさ（ライト）— bright cream base, dark text, soft pink accent. Values beyond the
 *  cream background are derived from the design swatch + spec (pending sign-off). */
val RunaLightColors = RunaColorScheme(
    Background = Color(0xFFFAF7F5),
    Surface = Color(0xFFFFFFFF),
    Heading = Color(0xFF2A2620),
    Body = Color(0xFF4E483F),
    Subtle = Color(0xFF8C8579),
    Accent = Color(0xFFE79CB6),
    SubAccent = Color(0xFFC9B8A0),
)

/** ピンク×ピンク — dark base with the accent pink pushed further. Derived from the
 *  spec (dark base + strong accent), pending sign-off. */
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

/** The active scheme, provided by [RunaTheme]. Static because a theme change should
 *  recompose the whole tree (like MaterialTheme's own color local). */
val LocalRunaColors = staticCompositionLocalOf { RunaDarkColors }

/**
 * Idiomatic accessor mirroring `MaterialTheme.colorScheme`: call sites keep writing
 * `RunaColors.Background` and transparently read the active theme's token. Usable
 * only in composable scope; non-composable draw code (e.g. MoonArt) holds its own
 * fixed constants instead.
 */
val RunaColors: RunaColorScheme
    @Composable @ReadOnlyComposable get() = LocalRunaColors.current
