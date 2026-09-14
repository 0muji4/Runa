package com.runa.shared.feature.insight

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.runa.shared.core.state.UiState
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.flatMapLatest
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import kotlinx.datetime.Clock
import kotlinx.datetime.DateTimeUnit
import kotlinx.datetime.DayOfWeek
import kotlinx.datetime.LocalDate
import kotlinx.datetime.TimeZone
import kotlinx.datetime.minus
import kotlinx.datetime.toLocalDateTime

/**
 * Drives the insight screen: holds the period on show and derives [state] from the
 * local diary stream + sync phase. [header] is separate because it must show over
 * both [UiState.Content] and [UiState.Empty].
 */
@OptIn(ExperimentalCoroutinesApi::class)
class InsightViewModel(
    private val repository: InsightRepository,
    private val zone: TimeZone = TimeZone.currentSystemDefault(),
    private val weekStart: DayOfWeek = InsightPeriods.DEFAULT_WEEK_START,
    private val clock: Clock = Clock.System,
) : ViewModel() {
    private val period = MutableStateFlow(InsightPeriods.monthlyContaining(today()))

    val header: StateFlow<InsightHeader> =
        period.map { p -> InsightHeader(periodLabel(p), p.type) }
            .stateIn(
                viewModelScope,
                SharingStarted.WhileSubscribed(5_000L),
                InsightHeader(periodLabel(period.value), period.value.type),
            )

    val state: StateFlow<UiState<Insight>> =
        period.flatMapLatest { p ->
            combine(repository.observeInsight(p, zone), repository.syncStatus) { insight, sync ->
                if (insight.summary.isEmpty) UiState.Empty else UiState.Content(insight, sync)
            }
        }.stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000L), UiState.Loading)

    init {
        refresh()
    }

    /** Switch week/month anchored to today. Same type is a no-op so the current window is not lost. */
    fun setPeriodType(type: InsightPeriodType) {
        if (period.value.type == type) return
        period.value = when (type) {
            InsightPeriodType.Weekly -> InsightPeriods.weeklyContaining(today(), weekStart)
            InsightPeriodType.Monthly -> InsightPeriods.monthlyContaining(today())
        }
        refresh()
    }

    fun showPrevious() {
        period.value = InsightPeriods.previous(period.value, weekStart)
        refresh()
    }

    fun showNext() {
        period.value = InsightPeriods.next(period.value, weekStart)
        refresh()
    }

    /** Jump back to the period containing today, keeping the week/month mode. */
    fun showCurrent() {
        period.value = when (period.value.type) {
            InsightPeriodType.Weekly -> InsightPeriods.weeklyContaining(today(), weekStart)
            InsightPeriodType.Monthly -> InsightPeriods.monthlyContaining(today())
        }
        refresh()
    }

    fun refresh() {
        viewModelScope.launch { repository.refresh() }
    }

    private fun today(): LocalDate = clock.now().toLocalDateTime(zone).date

    private fun periodLabel(p: InsightPeriod): String = when (p.type) {
        InsightPeriodType.Monthly -> "${p.start.monthNumber}月のうつろい"
        InsightPeriodType.Weekly -> {
            val last = p.endExclusive.minus(1, DateTimeUnit.DAY)
            if (p.start.monthNumber == last.monthNumber) {
                "${p.start.monthNumber}月${p.start.dayOfMonth}日〜${last.dayOfMonth}日のうつろい"
            } else {
                "${p.start.monthNumber}月${p.start.dayOfMonth}日〜${last.monthNumber}月${last.dayOfMonth}日のうつろい"
            }
        }
    }
}

/** The insight period chrome: a pre-formatted [periodLabel] and the current [periodType]. */
data class InsightHeader(
    val periodLabel: String,
    val periodType: InsightPeriodType,
)
