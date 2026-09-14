package com.runa.shared.feature.calendar

import com.runa.shared.core.state.SyncPhase
import com.runa.shared.feature.diary.DiaryEntry
import com.runa.shared.feature.diary.DiaryRepository
import com.runa.shared.network.ApiClient
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.map
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant
import kotlinx.datetime.LocalDate
import kotlinx.datetime.TimeZone
import kotlinx.datetime.toLocalDateTime

/**
 * Calendar boundary for the UI. Local-first: the observe streams come from the
 * on-device diary DB only; [refresh] is the sole network touch and never blocks render.
 */
interface CalendarRepository {

    /** Live [CalendarDay] list for the month, grouped by local date in [zone]. */
    fun observeMonth(year: Int, month: Int, zone: TimeZone): Flow<List<CalendarDay>>

    fun observeEntriesOn(year: Int, month: Int, day: Int, zone: TimeZone): Flow<List<DiaryEntry>>

    /** Push/pull via the diary sync so other devices' entries arrive; offline is a no-op. */
    suspend fun refresh(year: Int, month: Int, zone: TimeZone): Result<Unit>

    val syncStatus: StateFlow<SyncPhase>
}

/** Default [CalendarRepository]: folds the [DiaryRepository] stream through [CalendarGrid]; no own persistence. */
class DefaultCalendarRepository(
    private val diaryRepository: DiaryRepository,
    private val apiClient: ApiClient,
    private val clock: Clock = Clock.System,
) : CalendarRepository {

    override val syncStatus: StateFlow<SyncPhase> = diaryRepository.syncStatus

    override fun observeMonth(year: Int, month: Int, zone: TimeZone): Flow<List<CalendarDay>> =
        diaryRepository.observeEntries().map { entries ->
            val today = clock.now().toLocalDateTime(zone).date
            CalendarGrid.build(year, month, zone, today, countByDay(entries, year, month, zone))
        }

    override fun observeEntriesOn(year: Int, month: Int, day: Int, zone: TimeZone): Flow<List<DiaryEntry>> =
        diaryRepository.observeEntries().map { entries ->
            entries.filter { localDate(it, zone).let { d -> d.year == year && d.monthNumber == month && d.dayOfMonth == day } }
        }

    override suspend fun refresh(year: Int, month: Int, zone: TimeZone): Result<Unit> {
        val result = diaryRepository.sync()
        // Server counts are a consistency probe only, never used to draw; failures ignored.
        result.onSuccess { runCatching { apiClient.getCalendar(year, month, zone.id) } }
        return result
    }

    private fun countByDay(entries: List<DiaryEntry>, year: Int, month: Int, zone: TimeZone): Map<Int, Int> {
        val counts = HashMap<Int, Int>()
        for (entry in entries) {
            val date = localDate(entry, zone)
            if (date.year == year && date.monthNumber == month) {
                counts[date.dayOfMonth] = (counts[date.dayOfMonth] ?: 0) + 1
            }
        }
        return counts
    }

    private fun localDate(entry: DiaryEntry, zone: TimeZone): LocalDate =
        Instant.fromEpochMilliseconds(entry.createdAtEpochMs).toLocalDateTime(zone).date
}
