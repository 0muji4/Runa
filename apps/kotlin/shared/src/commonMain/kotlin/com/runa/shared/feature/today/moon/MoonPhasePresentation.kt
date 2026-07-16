package com.runa.shared.feature.today.moon

/**
 * Presentation helpers for a moon phase. These live in shared (not the UI layer)
 * for two reasons: the glyph is a universal emoji, and Runa is a single-locale
 * (Japanese) app, so keeping the name here guarantees Android and iOS show the
 * exact same label without each re-deriving it — and it lets the iOS side pass a
 * [MoonPhaseKey] straight back into Kotlin instead of switching on the SKIE enum.
 */

/** A moon-phase emoji glyph for the phase. */
fun moonPhaseGlyph(key: MoonPhaseKey): String = when (key) {
    MoonPhaseKey.NEW_MOON -> "🌑"
    MoonPhaseKey.WAXING_CRESCENT -> "🌒"
    MoonPhaseKey.FIRST_QUARTER -> "🌓"
    MoonPhaseKey.WAXING_GIBBOUS -> "🌔"
    MoonPhaseKey.FULL_MOON -> "🌕"
    MoonPhaseKey.WANING_GIBBOUS -> "🌖"
    MoonPhaseKey.LAST_QUARTER -> "🌗"
    MoonPhaseKey.WANING_CRESCENT -> "🌘"
}

/**
 * Whether the phase is waxing (its lit limb on the right), used to orient the
 * drawn moon disc. Derived from the phase key so Android and iOS pick the same
 * side. New/full are (un)lit so the side is cosmetic; new is treated as waxing.
 */
fun moonIsWaxing(key: MoonPhaseKey): Boolean = when (key) {
    MoonPhaseKey.NEW_MOON,
    MoonPhaseKey.WAXING_CRESCENT,
    MoonPhaseKey.FIRST_QUARTER,
    MoonPhaseKey.WAXING_GIBBOUS -> true
    MoonPhaseKey.FULL_MOON,
    MoonPhaseKey.WANING_GIBBOUS,
    MoonPhaseKey.LAST_QUARTER,
    MoonPhaseKey.WANING_CRESCENT -> false
}

/** The Japanese name for the phase (Runa is a Japanese-only product). */
fun moonPhaseNameJa(key: MoonPhaseKey): String = when (key) {
    MoonPhaseKey.NEW_MOON -> "新月"
    MoonPhaseKey.WAXING_CRESCENT -> "三日月"
    MoonPhaseKey.FIRST_QUARTER -> "上弦の月"
    MoonPhaseKey.WAXING_GIBBOUS -> "十三夜"
    MoonPhaseKey.FULL_MOON -> "満月"
    MoonPhaseKey.WANING_GIBBOUS -> "寝待月"
    MoonPhaseKey.LAST_QUARTER -> "下弦の月"
    MoonPhaseKey.WANING_CRESCENT -> "有明月"
}
