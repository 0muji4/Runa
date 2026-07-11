package com.runa.shared.feature.today.moon

/**
 * The eight canonical lunar phases, in synodic order from new moon back to new
 * moon. The UI maps each key to an icon and a localized name (新月 / 三日月 /
 * 上弦 / 十三夜 / 満月 / …／下弦 / 晦), so the shared layer stays free of
 * presentation strings.
 */
enum class MoonPhaseKey {
    NEW_MOON,
    WAXING_CRESCENT,
    FIRST_QUARTER,
    WAXING_GIBBOUS,
    FULL_MOON,
    WANING_GIBBOUS,
    LAST_QUARTER,
    WANING_CRESCENT,
}
