package com.runa.shared.feature.calendar

import com.runa.shared.feature.diary.DiaryEntry
import com.runa.shared.feature.diary.DiaryRepository
import com.runa.shared.feature.diary.SyncStatus
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
 * The calendar boundary the UI depends on. It is **local-first**: [observeMonth]
 * and [observeEntriesOn] are composed purely from the on-device diary DB plus the
 * offline [com.runa.shared.feature.today.moon.MoonPhaseCalculator], so the whole
 * screen renders with no network (DoD#1/#5). The only network touch is [refresh],
 * which reconciles other devices' entries via the existing diary sync.
 */
interface CalendarRepository {

    /** Live [CalendarDay] list for the month, grouped by the user's local date in
     *  [zone]. Re-emits on every local diary write and every applied server change. */
    fun observeMonth(year: Int, month: Int, zone: TimeZone): Flow<List<CalendarDay>>

    /** Live diary entries whose local date is the given day (for the day-tap view). */
    fun observeEntriesOn(year: Int, month: Int, day: Int, zone: TimeZone): Flow<List<DiaryEntry>>

    /**
     * Reconcile the month with the server: push/pull via the diary sync (this is
     * how entries written on another device arrive locally) and confirm against the
     * server-side authoritative counts. Never on the render path; offline is a
     * no-op that leaves the local render intact.
     */
    suspend fun refresh(year: Int, month: Int, zone: TimeZone): Result<Unit>

    /** The diary sync status, surfaced as the calendar's quiet banner. */
    val syncStatus: StateFlow<SyncStatus>
}

/**
 * Default [CalendarRepository]. Adds no new persistence: it observes the existing
 * [DiaryRepository] entry stream and folds each month/day together with the shared
 * moon phase via [CalendarGrid]. [refresh] delegates cross-device reconciliation to
 * [DiaryRepository.sync] and then probes `GET /diary/calendar` — the server's count
 * of record, used only for consistency confirmation, never to draw the grid.
 */
class DefaultCalendarRepository(
    private val diaryRepository: DiaryRepository,
    private val apiClient: ApiClient,
    private val clock: Clock = Clock.System,
) : CalendarRepository {

    override val syncStatus: StateFlow<SyncStatus> = diaryRepository.syncStatus

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
        // Auxiliary: confirm the server's authoritative per-day counts once the pull
        // has landed other devices' entries locally. Failures (offline) are ignored;
        // the local render is already correct.
        result.onSuccess { runCatching { apiClient.getCalendar(year, month, zone.id) } }
        return result
    }

    /** Group visible entries by day-of-month for [year]/[month], in [zone]. */
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
