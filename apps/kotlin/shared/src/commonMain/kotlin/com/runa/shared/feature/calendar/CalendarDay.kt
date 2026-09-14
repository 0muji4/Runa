package com.runa.shared.feature.calendar

import com.runa.shared.feature.today.moon.MoonPhaseKey

/**
 * One cell of the retrospective calendar: a local day with its moon phase
 * ([illumination] 0.0 new .. 1.0 full) and the number of diary entries on it.
 */
data class CalendarDay(
    val year: Int,
    val month: Int,
    val day: Int,
    val phaseKey: MoonPhaseKey,
    val illumination: Double,
    val entryCount: Int,
    val isToday: Boolean,
)
