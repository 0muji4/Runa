package com.runa.shared.feature.today.moon

/** Moon phase for a day: [illumination] 0.0 (new) .. 1.0 (full); [ageDays] 月齢 0.0 .. ~29.53. */
data class MoonPhase(
    val phaseKey: MoonPhaseKey,
    val illumination: Double,
    val ageDays: Double,
)
