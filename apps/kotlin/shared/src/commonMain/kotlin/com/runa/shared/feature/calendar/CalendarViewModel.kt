package com.runa.shared.feature.calendar

import com.runa.shared.feature.diary.SyncStatus
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.flatMapLatest
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import kotlinx.datetime.Clock
import kotlinx.datetime.TimeZone
import kotlinx.datetime.toLocalDateTime

/**
 * Drives the retrospective calendar. Holds the month on show and derives [state]
 * from the local DB stream + sync status, so it renders instantly from cache and
 * works fully offline. Android collects [state] directly; iOS observes via SKIE.
 *
 * A `factory` binding gives each open a fresh instance starting at today's month
 * (so "今日へ戻る" is the default entry point).
 */
@OptIn(ExperimentalCoroutinesApi::class)
class CalendarViewModel(
    private val repository: CalendarRepository,
    private val zone: TimeZone = TimeZone.currentSystemDefault(),
    private val scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Default),
    private val clock: Clock = Clock.System,
) {
    private val month = MutableStateFlow(currentYearMonth())

    val state: StateFlow<CalendarUiState> =
        month.flatMapLatest { ym ->
            combine(repository.observeMonth(ym.year, ym.month, zone), repository.syncStatus) { days, sync ->
                CalendarUiState.Content(
                    year = ym.year,
                    month = ym.month,
                    firstDayOfWeek = CalendarGrid.firstDayOfWeekIndex(ym.year, ym.month),
                    days = days,
                    banner = sync.toBanner(),
                )
            }
        }.stateIn(scope, SharingStarted.WhileSubscribed(5_000L), CalendarUiState.Loading)

    init {
        // Bring other devices' entries in; the local render is already showing.
        refresh()
    }

    fun showPreviousMonth() {
        month.value = month.value.previous()
        refresh()
    }

    fun showNextMonth() {
        month.value = month.value.next()
        refresh()
    }

    /** Jump back to the current month ("今日へ戻る"). */
    fun showToday() {
        month.value = currentYearMonth()
        refresh()
    }

    fun refresh() {
        val ym = month.value
        scope.launch { repository.refresh(ym.year, ym.month, zone) }
    }

    private fun currentYearMonth(): YearMonth {
        val date = clock.now().toLocalDateTime(zone).date
        return YearMonth(date.year, date.monthNumber)
    }
}

/** Year + 1-based month, with wrap-around navigation. */
data class YearMonth(val year: Int, val month: Int) {
    fun next(): YearMonth = if (month == 12) YearMonth(year + 1, 1) else YearMonth(year, month + 1)
    fun previous(): YearMonth = if (month == 1) YearMonth(year - 1, 12) else YearMonth(year, month - 1)
}

/**
 * Calendar UI state. Local-first means we almost always have [Content]; the grid
 * always renders (a month with no records is simply all-zero counts), and
 * offline/error ride along as a quiet [banner] rather than hiding the body.
 * [Loading] shows only before the first DB emission.
 */
sealed interface CalendarUiState {
    data object Loading : CalendarUiState
    data class Content(
        val year: Int,
        val month: Int,
        val firstDayOfWeek: Int,
        val days: List<CalendarDay>,
        val banner: CalendarBanner,
    ) : CalendarUiState
}

/** The quiet status line shown on the calendar. */
enum class CalendarBanner { None, Syncing, Offline, Error }

private fun SyncStatus.toBanner(): CalendarBanner = when (this) {
    SyncStatus.Idle -> CalendarBanner.None
    SyncStatus.Syncing -> CalendarBanner.Syncing
    SyncStatus.Offline -> CalendarBanner.Offline
    SyncStatus.Error -> CalendarBanner.Error
}
