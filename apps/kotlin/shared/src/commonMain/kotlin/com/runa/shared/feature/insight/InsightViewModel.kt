package com.runa.shared.feature.insight

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
 * Drives the insight ("ふりかえり") screen. Holds the period on show and derives the
 * shared [UiState] from the local diary stream + sync phase, so it renders instantly
 * from cache and works fully offline. Android collects the flows directly; iOS
 * observes via SKIE.
 *
 * The period chrome (label + week/month toggle) is exposed as a separate [header]
 * flow because it must show across both [UiState.Content] and [UiState.Empty] — the
 * generic [state] only drives the letter body. Local-first means we almost always
 * have [UiState.Content] or [UiState.Empty] (a period with no records); offline/sync
 * ride along as [UiState.Content.sync] rather than hiding the body.
 *
 * A `factory` binding gives each open a fresh instance starting at the current
 * month (matching the design's monthly "letter"); [setPeriodType] flips to the
 * week/month containing today, and [showPrevious]/[showNext]/[showCurrent] move the
 * window.
 */
@OptIn(ExperimentalCoroutinesApi::class)
class InsightViewModel(
    private val repository: InsightRepository,
    private val zone: TimeZone = TimeZone.currentSystemDefault(),
    private val weekStart: DayOfWeek = InsightPeriods.DEFAULT_WEEK_START,
    private val clock: Clock = Clock.System,
    private val scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Default),
) {
    private val period = MutableStateFlow(InsightPeriods.monthlyContaining(today()))

    /** The always-present period chrome (label + week/month mode) shown above the body. */
    val header: StateFlow<InsightHeader> =
        period.map { p -> InsightHeader(periodLabel(p), p.type) }
            .stateIn(
                scope,
                SharingStarted.WhileSubscribed(5_000L),
                InsightHeader(periodLabel(period.value), period.value.type),
            )

    val state: StateFlow<UiState<Insight>> =
        period.flatMapLatest { p ->
            combine(repository.observeInsight(p, zone), repository.syncStatus) { insight, sync ->
                if (insight.summary.isEmpty) UiState.Empty else UiState.Content(insight, sync)
            }
        }.stateIn(scope, SharingStarted.WhileSubscribed(5_000L), UiState.Loading)

    init {
        // Bring other devices' entries in; the local render is already showing.
        refresh()
    }

    /** Switch week/month, anchored to the period that contains today. Tapping the
     *  already-selected type is a no-op — re-anchoring to "now" is the period label's
     *  job ([showCurrent]) — so the current window isn't lost on a redundant tap. */
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

    /** Jump back to the period containing today, keeping the current week/month mode. */
    fun showCurrent() {
        period.value = when (period.value.type) {
            InsightPeriodType.Weekly -> InsightPeriods.weeklyContaining(today(), weekStart)
            InsightPeriodType.Monthly -> InsightPeriods.monthlyContaining(today())
        }
        refresh()
    }

    fun refresh() {
        scope.launch { repository.refresh() }
    }

    private fun today(): LocalDate = clock.now().toLocalDateTime(zone).date

    /** Quiet, pre-formatted period label shown above the heading. */
    private fun periodLabel(p: InsightPeriod): String = when (p.type) {
        InsightPeriodType.Monthly -> "${p.start.monthNumber}月のふりかえり"
        InsightPeriodType.Weekly -> {
            val last = p.endExclusive.minus(1, DateTimeUnit.DAY)
            if (p.start.monthNumber == last.monthNumber) {
                "${p.start.monthNumber}月${p.start.dayOfMonth}日〜${last.dayOfMonth}日のふりかえり"
            } else {
                "${p.start.monthNumber}月${p.start.dayOfMonth}日〜${last.monthNumber}月${last.dayOfMonth}日のふりかえり"
            }
        }
    }
}

/**
 * The insight period chrome: a pre-formatted [periodLabel] and the current
 * [periodType]. Kept separate from [UiState] because the selector + label are shown
 * over both the content and the empty body.
 */
data class InsightHeader(
    val periodLabel: String,
    val periodType: InsightPeriodType,
)
