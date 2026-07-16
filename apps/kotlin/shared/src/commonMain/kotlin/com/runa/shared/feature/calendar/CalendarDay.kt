package com.runa.shared.feature.calendar

import com.runa.shared.feature.today.moon.MoonPhaseKey

/**
 * One cell of the retrospective calendar (12 ふりかえりカレンダー): a calendar day
 * with its locally computed moon phase and how many diary entries fall on it.
 *
 * The date is carried as plain [year]/[month]/[day] ints (not a
 * kotlinx.datetime.LocalDate) so the UI layer need not depend on a datetime
 * library — the same reasoning as the diary slice's epoch-millis Longs.
 *
 * Per the confirmed design the moon is drawn only where [entryCount] > 0 ("記録の
 * ある日に、月あかり"), but the phase is computed for every day so both platforms
 * stay byte-identical and the rule lives entirely in the UI.
 *
 * @property phaseKey the canonical moon phase for the day (drives the disc).
 * @property illumination the lit fraction 0.0 (new) .. 1.0 (full).
 * @property entryCount local diary entries whose local date is this day.
 * @property isToday whether this is the user's current local date.
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
