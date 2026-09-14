package com.runa.shared.feature.insight

import kotlinx.datetime.DateTimeUnit
import kotlinx.datetime.DayOfWeek
import kotlinx.datetime.LocalDate
import kotlinx.datetime.isoDayNumber
import kotlinx.datetime.minus
import kotlinx.datetime.plus

enum class InsightPeriodType { Weekly, Monthly }

/** A half-open local-day window `[start, endExclusive)`. Not exposed to the UI (carries [LocalDate]). */
data class InsightPeriod(
    val type: InsightPeriodType,
    val start: LocalDate,
    val endExclusive: LocalDate,
) {
    fun contains(date: LocalDate): Boolean = date >= start && date < endExclusive
}

/** Pure period-boundary logic. Week start defaults to Sunday, matching the calendar's 日〜土 header. */
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

    fun previous(period: InsightPeriod, weekStart: DayOfWeek = DEFAULT_WEEK_START): InsightPeriod = when (period.type) {
        InsightPeriodType.Weekly -> weeklyContaining(period.start.minus(7, DateTimeUnit.DAY), weekStart)
        InsightPeriodType.Monthly -> monthlyContaining(period.start.minus(1, DateTimeUnit.MONTH))
    }

    fun next(period: InsightPeriod, weekStart: DayOfWeek = DEFAULT_WEEK_START): InsightPeriod = when (period.type) {
        InsightPeriodType.Weekly -> weeklyContaining(period.endExclusive, weekStart)
        InsightPeriodType.Monthly -> monthlyContaining(period.endExclusive)
    }

    private fun LocalDate.startOfWeek(weekStart: DayOfWeek): LocalDate {
        // isoDayNumber is Mon=1..Sun=7; the offset is how many days since weekStart.
        val offset = (dayOfWeek.isoDayNumber - weekStart.isoDayNumber + 7) % 7
        return minus(offset, DateTimeUnit.DAY)
    }
}
