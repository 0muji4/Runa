package com.runa.shared.network.dto

import kotlinx.serialization.Serializable

/**
 * Response of `GET /api/v1/diary/calendar?year=&month=&tz=` — the server-side
 * authoritative count of diary entries per local date for the month. It is an
 * auxiliary consistency check only; the calendar is rendered from the local DB
 * (see [com.runa.shared.feature.calendar.CalendarRepository]).
 *
 * [days] lists only dates that have at least one entry; [CalendarDayCount.date] is
 * the user's local date (yyyy-MM-dd) under the requested time zone.
 */
@Serializable
data class DiaryCalendarResponse(
    val year: Int,
    val month: Int,
    val days: List<CalendarDayCount> = emptyList(),
)

@Serializable
data class CalendarDayCount(
    val date: String,
    val count: Int,
)
