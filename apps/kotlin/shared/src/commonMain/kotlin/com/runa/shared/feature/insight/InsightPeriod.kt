package com.runa.shared.feature.insight

import kotlinx.datetime.DateTimeUnit
import kotlinx.datetime.DayOfWeek
import kotlinx.datetime.LocalDate
import kotlinx.datetime.isoDayNumber
import kotlinx.datetime.minus
import kotlinx.datetime.plus

/** Whether an insight window spans one week or one calendar month. */
enum class InsightPeriodType { Weekly, Monthly }

/**
 * A half-open day window `[start, endExclusive)` in the user's local calendar,
 * plus which kind it is. This is an **internal** aggregation type — it carries
 * [LocalDate]s and so is never exposed across the SKIE boundary (the UI receives a
 * pre-formatted label + [InsightPeriodType] instead, matching how today/diary keep
 * kotlinx-datetime off the clients' classpath).
 */
data class InsightPeriod(
    val type: InsightPeriodType,
    val start: LocalDate,
    val endExclusive: LocalDate,
) {
    /** True if [date] falls in this window (start inclusive, end exclusive). */
    fun contains(date: LocalDate): Boolean = date >= start && date < endExclusive
}

/**
 * Pure period-boundary logic for insights — no database, no repository, no network,
 * mirroring [com.runa.shared.feature.calendar.CalendarGrid]. Isolated as an `object`
 * so the DoD boundary cases (week start, month/year wrap, leap February) are
 * directly unit-testable in commonTest and so Android and iOS derive identical
 * windows from the shared layer rather than each doing platform date math.
 *
 * Week start defaults to **Sunday**, matching the calendar's 日〜土 header
 * ([com.runa.shared.feature.calendar.CalendarGrid.firstDayOfWeekIndex]); callers
 * may override it.
 */
object InsightPeriods {

    val DEFAULT_WEEK_START: DayOfWeek = DayOfWeek.SUNDAY

    /** The calendar month containing [date]: `[1st, 1st-of-next-month)`. */
    fun monthlyContaining(date: LocalDate): InsightPeriod {
        val start = LocalDate(date.year, date.monthNumber, 1)
        return InsightPeriod(InsightPeriodType.Monthly, start, start.plus(1, DateTimeUnit.MONTH))
    }

    /** The 7-day week containing [date], beginning on [weekStart]. */
    fun weeklyContaining(date: LocalDate, weekStart: DayOfWeek = DEFAULT_WEEK_START): InsightPeriod {
        val start = date.startOfWeek(weekStart)
        return InsightPeriod(InsightPeriodType.Weekly, start, start.plus(7, DateTimeUnit.DAY))
    }

    /** The window immediately before [period] (previous week / previous month). */
    fun previous(period: InsightPeriod, weekStart: DayOfWeek = DEFAULT_WEEK_START): InsightPeriod = when (period.type) {
        InsightPeriodType.Weekly -> weeklyContaining(period.start.minus(7, DateTimeUnit.DAY), weekStart)
        InsightPeriodType.Monthly -> monthlyContaining(period.start.minus(1, DateTimeUnit.MONTH))
    }

    /** The window immediately after [period] (next week / next month). */
    fun next(period: InsightPeriod, weekStart: DayOfWeek = DEFAULT_WEEK_START): InsightPeriod = when (period.type) {
        InsightPeriodType.Weekly -> weeklyContaining(period.endExclusive, weekStart)
        InsightPeriodType.Monthly -> monthlyContaining(period.endExclusive)
    }

    /** Back up to the most recent [weekStart] on or before this date. */
    private fun LocalDate.startOfWeek(weekStart: DayOfWeek): LocalDate {
        // isoDayNumber is Mon=1..Sun=7; the offset is how many days since weekStart.
        val offset = (dayOfWeek.isoDayNumber - weekStart.isoDayNumber + 7) % 7
        return minus(offset, DateTimeUnit.DAY)
    }
}
