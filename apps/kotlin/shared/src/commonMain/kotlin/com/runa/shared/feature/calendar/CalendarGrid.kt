package com.runa.shared.feature.calendar

import com.runa.shared.feature.today.moon.MoonPhaseCalculator
import kotlinx.datetime.DateTimeUnit
import kotlinx.datetime.LocalDate
import kotlinx.datetime.TimeZone
import kotlinx.datetime.daysUntil
import kotlinx.datetime.isoDayNumber
import kotlinx.datetime.plus

/** Pure month-grid logic (no DB, no network) so the layout is unit-testable. */
object CalendarGrid {

    fun daysInMonth(year: Int, month: Int): Int {
        val first = LocalDate(year, month, 1)
        return first.daysUntil(first.plus(1, DateTimeUnit.MONTH))
    }

    /** Column (0 = Sunday .. 6 = Saturday) of the 1st of [month] = leading blank cells. */
    fun firstDayOfWeekIndex(year: Int, month: Int): Int =
        // isoDayNumber is Mon=1..Sun=7; mod 7 maps Sun→0, Mon→1, … Sat→6.
        LocalDate(year, month, 1).dayOfWeek.isoDayNumber % 7

    /** Build every [CalendarDay] of the month; [entryCountByDay] is keyed by day-of-month. */
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
