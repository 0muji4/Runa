package com.runa.shared.feature.calendar

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.runa.shared.core.state.UiState
import kotlinx.coroutines.ExperimentalCoroutinesApi
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
 * Drives the retrospective calendar: holds the month on show and derives [state] from
 * the local DB stream + sync phase. Offline/error ride along as [UiState.Content.sync].
 */
@OptIn(ExperimentalCoroutinesApi::class)
class CalendarViewModel(
    private val repository: CalendarRepository,
    private val zone: TimeZone = TimeZone.currentSystemDefault(),
    private val clock: Clock = Clock.System,
) : ViewModel() {
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
        }.stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000L), UiState.Loading)

    init {
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

    fun showToday() {
        month.value = currentYearMonth()
        refresh()
    }

    fun refresh() {
        val ym = month.value
        viewModelScope.launch { repository.refresh(ym.year, ym.month, zone) }
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

/** The month a calendar screen renders; [firstDayOfWeek] is the count of leading blank cells. */
data class CalendarMonth(
    val year: Int,
    val month: Int,
    val firstDayOfWeek: Int,
    val days: List<CalendarDay>,
)
