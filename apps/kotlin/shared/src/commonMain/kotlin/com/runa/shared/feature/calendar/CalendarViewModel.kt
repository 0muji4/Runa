package com.runa.shared.feature.calendar

import com.runa.shared.core.state.UiState
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
 * (the shared [UiState]) from the local DB stream + sync phase, so it renders
 * instantly from cache and works fully offline. Local-first means the grid always
 * renders (a month with no records is simply all-zero counts), so the state is
 * effectively [UiState.Loading] then [UiState.Content]; offline/error ride along as
 * [UiState.Content.sync] rather than hiding the body. Android collects [state]
 * directly; iOS observes via SKIE.
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

    val state: StateFlow<UiState<CalendarMonth>> =
        month.flatMapLatest { ym ->
            combine(repository.observeMonth(ym.year, ym.month, zone), repository.syncStatus) { days, sync ->
                UiState.Content(
                    CalendarMonth(
                        year = ym.year,
                        month = ym.month,
                        firstDayOfWeek = CalendarGrid.firstDayOfWeekIndex(ym.year, ym.month),
                        days = days,
                    ),
                    sync,
                )
            }
        }.stateIn(scope, SharingStarted.WhileSubscribed(5_000L), UiState.Loading)

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
 * The month a calendar screen renders: the [days] grid plus its layout metadata
 * ([year]/[month] and the [firstDayOfWeek] index for the leading blank cells). This
 * is the payload carried by [UiState.Content].
 */
data class CalendarMonth(
    val year: Int,
    val month: Int,
    val firstDayOfWeek: Int,
    val days: List<CalendarDay>,
)
