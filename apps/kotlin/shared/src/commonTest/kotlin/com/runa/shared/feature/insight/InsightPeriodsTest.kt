package com.runa.shared.feature.insight

import kotlinx.datetime.DayOfWeek
import kotlinx.datetime.LocalDate
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

/**
 * Period-boundary logic: week start, month/year wrap, leap February, and the
 * half-open `[start, endExclusive)` membership. Pure commonMain, so a green run
 * proves Android and iOS derive identical windows.
 */
class InsightPeriodsTest {

    @Test
    fun monthlyWindowIsFirstToFirstOfNext() {
        val p = InsightPeriods.monthlyContaining(LocalDate(2026, 7, 15))
        assertEquals(InsightPeriodType.Monthly, p.type)
        assertEquals(LocalDate(2026, 7, 1), p.start)
        assertEquals(LocalDate(2026, 8, 1), p.endExclusive)
        assertTrue(p.contains(LocalDate(2026, 7, 1)))
        assertTrue(p.contains(LocalDate(2026, 7, 31)))
        assertFalse(p.contains(LocalDate(2026, 8, 1)), "end is exclusive")
        assertFalse(p.contains(LocalDate(2026, 6, 30)))
    }

    @Test
    fun monthlyWrapsAcrossTheYear() {
        val dec = InsightPeriods.monthlyContaining(LocalDate(2026, 12, 10))
        assertEquals(LocalDate(2027, 1, 1), dec.endExclusive)
        assertEquals(InsightPeriods.monthlyContaining(LocalDate(2027, 1, 20)), InsightPeriods.next(dec))
        assertEquals(InsightPeriods.monthlyContaining(LocalDate(2026, 11, 5)), InsightPeriods.previous(dec))
    }

    @Test
    fun leapFebruaryIncludesThe29th() {
        val leap = InsightPeriods.monthlyContaining(LocalDate(2024, 2, 10))
        assertEquals(LocalDate(2024, 3, 1), leap.endExclusive)
        assertTrue(leap.contains(LocalDate(2024, 2, 29)), "2024-02-29 is inside leap February")

        val common = InsightPeriods.monthlyContaining(LocalDate(2023, 2, 10))
        assertEquals(LocalDate(2023, 3, 1), common.endExclusive)
        assertTrue(common.contains(LocalDate(2023, 2, 28)))
    }

    @Test
    fun weeklyStartsOnSundayByDefault() {
        // 2026-07-15 is a Wednesday; its Sunday-start week begins 2026-07-12.
        val w = InsightPeriods.weeklyContaining(LocalDate(2026, 7, 15))
        assertEquals(InsightPeriodType.Weekly, w.type)
        assertEquals(DayOfWeek.SUNDAY, w.start.dayOfWeek)
        assertEquals(LocalDate(2026, 7, 12), w.start)
        assertEquals(LocalDate(2026, 7, 19), w.endExclusive)
        assertTrue(w.contains(LocalDate(2026, 7, 15)))
        assertFalse(w.contains(LocalDate(2026, 7, 19)), "end is exclusive")
        assertFalse(w.contains(LocalDate(2026, 7, 11)))
    }

    @Test
    fun weeklyRespectsAConfiguredMondayStart() {
        val w = InsightPeriods.weeklyContaining(LocalDate(2026, 7, 15), DayOfWeek.MONDAY)
        assertEquals(DayOfWeek.MONDAY, w.start.dayOfWeek)
        assertEquals(LocalDate(2026, 7, 13), w.start)
        assertEquals(LocalDate(2026, 7, 20), w.endExclusive)
    }

    @Test
    fun weeklySpansLeapFebruaryIntoMarch() {
        // 2024-02-29 (Thu); its Sunday-start week is 2024-02-25 .. 2024-03-03.
        val w = InsightPeriods.weeklyContaining(LocalDate(2024, 2, 29))
        assertEquals(LocalDate(2024, 2, 25), w.start)
        assertEquals(LocalDate(2024, 3, 3), w.endExclusive)
        assertTrue(w.contains(LocalDate(2024, 2, 29)))
        assertTrue(w.contains(LocalDate(2024, 3, 1)))
        assertEquals(w, InsightPeriods.previous(InsightPeriods.weeklyContaining(LocalDate(2024, 3, 3))))
    }

    @Test
    fun nextAndPreviousAreInverse() {
        val w = InsightPeriods.weeklyContaining(LocalDate(2026, 7, 15))
        assertEquals(w, InsightPeriods.previous(InsightPeriods.next(w)))
        val m = InsightPeriods.monthlyContaining(LocalDate(2026, 7, 15))
        assertEquals(m, InsightPeriods.previous(InsightPeriods.next(m)))
    }
}
