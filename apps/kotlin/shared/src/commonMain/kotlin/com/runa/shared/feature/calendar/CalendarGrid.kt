package com.runa.shared.feature.calendar

import com.runa.shared.feature.today.moon.MoonPhaseCalculator
import kotlinx.datetime.DateTimeUnit
import kotlinx.datetime.LocalDate
import kotlinx.datetime.TimeZone
import kotlinx.datetime.daysUntil
import kotlinx.datetime.isoDayNumber
import kotlinx.datetime.plus

/**
 * Pure month-grid logic for the calendar — no database, no repository, no network.
 * Isolated here so the DoD cases (leap year, month length, the weekday a month
 * starts on) are directly unit-testable in commonTest, and so Android and iOS
 * derive the exact same grid layout from the shared layer rather than each doing
 * platform date math.
 */
object CalendarGrid {

    /** The number of days in [month] of [year], leap years included. */
    fun daysInMonth(year: Int, month: Int): Int {
        val first = LocalDate(year, month, 1)
        return first.daysUntil(first.plus(1, DateTimeUnit.MONTH))
    }

    /**
     * The column (0 = Sunday .. 6 = Saturday) that the 1st of [month] lands in,
     * i.e. the number of empty leading cells before day 1. The design's weekday
     * header runs 日〜土, so Sunday is column 0.
     */
    fun firstDayOfWeekIndex(year: Int, month: Int): Int =
        // isoDayNumber is Mon=1..Sun=7; mod 7 maps Sun→0, Mon→1, … Sat→6.
        LocalDate(year, month, 1).dayOfWeek.isoDayNumber % 7

    /**
     * Build every [CalendarDay] of the month: the moon phase for each day (via the
     * shared [MoonPhaseCalculator], computed at local noon in [zone]), the diary
     * [entryCountByDay] (keyed by day-of-month), and whether the day is [today].
     */
    fun build(
        year: Int,
        month: Int,
        zone: TimeZone,
        today: LocalDate,
        entryCountByDay: Map<Int, Int>,
    ): List<CalendarDay> = (1..daysInMonth(year, month)).map { day ->
        val date = LocalDate(year, month, day)
        val phase = MoonPhaseCalculator.phaseFor(date, zone)
        CalendarDay(
            year = year,
            month = month,
            day = day,
            phaseKey = phase.phaseKey,
            illumination = phase.illumination,
            entryCount = entryCountByDay[day] ?: 0,
            isToday = date == today,
        )
    }
}
