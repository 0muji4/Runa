package com.runa.shared.feature.calendar

import com.runa.shared.feature.today.moon.MoonPhaseCalculator
import kotlinx.datetime.LocalDate
import kotlinx.datetime.TimeZone
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

/**
 * Pure month-grid tests (DoD: month length, leap year, the weekday a month starts
 * on, and that each cell carries the shared moon phase). No DB or network.
 */
class CalendarGridTest {

    private val utc = TimeZone.UTC

    @Test
    fun daysInMonthHandlesLeapYears() {
        assertEquals(29, CalendarGrid.daysInMonth(2024, 2), "Feb 2024 (leap)")
        assertEquals(28, CalendarGrid.daysInMonth(2025, 2), "Feb 2025")
        assertEquals(31, CalendarGrid.daysInMonth(2026, 7), "Jul 2026")
        assertEquals(30, CalendarGrid.daysInMonth(2026, 4), "Apr 2026")
        assertEquals(31, CalendarGrid.daysInMonth(2026, 12), "Dec 2026")
    }

    @Test
    fun firstDayOfWeekMatchesTheDesign() {
        // The design shows July 2026 starting under 水 (Wed): column 3 (Sun=0).
        assertEquals(3, CalendarGrid.firstDayOfWeekIndex(2026, 7), "Jul 2026 = Wed")
        // Feb 1 2024 is a Thursday → column 4.
        assertEquals(4, CalendarGrid.firstDayOfWeekIndex(2024, 2), "Feb 2024 = Thu")
        // Feb 1 2026 is a Sunday → column 0.
        assertEquals(0, CalendarGrid.firstDayOfWeekIndex(2026, 2), "Feb 2026 = Sun")
    }

    @Test
    fun buildProducesEveryDayWithItsMoonAndCounts() {
        val days = CalendarGrid.build(
            year = 2026, month = 7, zone = utc,
            today = LocalDate(2026, 7, 4),
            entryCountByDay = mapOf(1 to 2, 4 to 1),
        )

        assertEquals(31, days.size)
        assertEquals(1, days.first().day)
        assertEquals(31, days.last().day)

        // Counts reflect the map; other days are 0.
        assertEquals(2, days.first { it.day == 1 }.entryCount)
        assertEquals(1, days.first { it.day == 4 }.entryCount)
        assertEquals(0, days.first { it.day == 2 }.entryCount)

        // isToday only on the 4th.
        assertTrue(days.first { it.day == 4 }.isToday)
        assertFalse(days.first { it.day == 3 }.isToday)

        // Each cell carries the shared calculator's phase for that day.
        for (cell in days) {
            val expected = MoonPhaseCalculator.phaseFor(LocalDate(2026, 7, cell.day), utc)
            assertEquals(expected.phaseKey, cell.phaseKey, "phase for day ${cell.day}")
            assertEquals(expected.illumination, cell.illumination, "illum for day ${cell.day}")
        }
    }
}
