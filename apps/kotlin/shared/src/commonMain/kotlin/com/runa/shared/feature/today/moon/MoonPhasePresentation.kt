package com.runa.shared.feature.today.moon

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

/** Whether the lit limb is on the right, to orient the drawn disc; new is treated as waxing. */
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

fun moonPhaseNameJa(key: MoonPhaseKey): String = when (key) {
    MoonPhaseKey.NEW_MOON -> "新月"
    MoonPhaseKey.WAXING_CRESCENT -> "三日月"
    MoonPhaseKey.FIRST_QUARTER -> "上弦の月"
    // 「十三夜」だけだと旧暦九月十三日の一夜（後の月）を指すので「十三夜月」にする。
    MoonPhaseKey.WAXING_GIBBOUS -> "十三夜月"
    MoonPhaseKey.FULL_MOON -> "満月"
    MoonPhaseKey.WANING_GIBBOUS -> "寝待月"
    MoonPhaseKey.LAST_QUARTER -> "下弦の月"
    MoonPhaseKey.WANING_CRESCENT -> "有明月"
}
