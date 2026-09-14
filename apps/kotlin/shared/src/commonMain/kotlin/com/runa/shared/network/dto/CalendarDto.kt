package com.runa.shared.network.dto

import kotlinx.serialization.Serializable

/**
 * Response of `GET /api/v1/diary/calendar?year=&month=&tz=`. [days] lists only dates with at least
 * one entry; [CalendarDayCount.date] is the local date (yyyy-MM-dd) under the requested time zone.
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
