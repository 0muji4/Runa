package com.runa.shared.feature.today.moon

/**
 * The computed moon phase for a given day.
 *
 * @property phaseKey the canonical phase bucket (drives the UI icon + name).
 * @property illumination the lit fraction of the disc, 0.0 (new) .. 1.0 (full).
 * @property ageDays the moon's age (月齢) in days, 0.0 .. ~29.53.
 */
data class MoonPhase(
    val phaseKey: MoonPhaseKey,
    val illumination: Double,
    val ageDays: Double,
)
