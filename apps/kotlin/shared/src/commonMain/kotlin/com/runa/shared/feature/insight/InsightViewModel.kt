package com.runa.shared.feature.insight

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
import kotlinx.datetime.DateTimeUnit
import kotlinx.datetime.DayOfWeek
import kotlinx.datetime.LocalDate
import kotlinx.datetime.TimeZone
import kotlinx.datetime.minus
import kotlinx.datetime.toLocalDateTime

/**
 * Drives the insight ("ふりかえり") screen. Holds the period on show and derives
 * [state] from the local diary stream + sync status, so it renders instantly from
 * cache and works fully offline. Android collects [state] directly; iOS observes
 * via SKIE.
 *
 * A `factory` binding gives each open a fresh instance starting at the current
 * month (matching the design's monthly "letter"); [setPeriodType] flips to the
 * week/month containing today, and [showPrevious]/[showNext]/[showCurrent] move the
 * window. The UI never sees a [LocalDate]: it gets a pre-formatted [InsightUiState]
 * carrying a period label string + [InsightPeriodType].
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

    val state: StateFlow<InsightUiState> =
        period.flatMapLatest { p ->
            combine(repository.observeInsight(p, zone), repository.syncStatus) { insight, sync ->
                val banner = sync.toBanner()
                if (insight.summary.isEmpty) {
                    InsightUiState.Empty(periodLabel(p), p.type, banner)
                } else {
                    InsightUiState.Content(insight, periodLabel(p), p.type, banner)
                }
            }
        }.stateIn(scope, SharingStarted.WhileSubscribed(5_000L), InsightUiState.Loading)

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
 * Insight UI state. Local-first means we almost always have [Content] or [Empty]
 * (a period with no records); offline/sync ride along as a quiet [banner] rather
 * than hiding the body. [Loading] shows only before the first DB emission, and
 * [Error] is reserved for a genuine failure (unreachable on the local path today,
 * but the state exists for a future server-summary path that can fail).
 */
sealed interface InsightUiState {
    data object Loading : InsightUiState
    data class Content(
        val insight: Insight,
        val periodLabel: String,
        val periodType: InsightPeriodType,
        val banner: InsightBanner,
    ) : InsightUiState
    data class Empty(
        val periodLabel: String,
        val periodType: InsightPeriodType,
        val banner: InsightBanner,
    ) : InsightUiState
    data class Error(val banner: InsightBanner) : InsightUiState
}

/** The quiet status line shown on the insight screen. */
enum class InsightBanner { None, Syncing, Offline, Error }

private fun SyncStatus.toBanner(): InsightBanner = when (this) {
    SyncStatus.Idle -> InsightBanner.None
    SyncStatus.Syncing -> InsightBanner.Syncing
    SyncStatus.Offline -> InsightBanner.Offline
    SyncStatus.Error -> InsightBanner.Error
}
