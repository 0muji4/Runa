package com.runa.shared.feature.insight

import com.runa.shared.feature.diary.DiaryRepository
import com.runa.shared.feature.diary.SyncStatus
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.mapLatest
import kotlinx.datetime.TimeZone

/** An insight period's computed facts paired with its composed read-back text. */
data class Insight(
    val summary: InsightSummary,
    val narrative: InsightNarrative,
)

/**
 * The insight boundary the UI depends on. Like [com.runa.shared.feature.calendar.CalendarRepository]
 * it is **local-first**: [observeInsight] is composed purely from the on-device
 * diary DB ([InsightCalculator]) plus a [SummaryComposer], so the whole screen
 * renders with no network. The only network touch is [refresh], which reconciles
 * other devices' entries via the existing diary sync.
 *
 * A future server-side summariser or cross-device aggregation hides behind this
 * interface (swap the injected [SummaryComposer], or provide an alternative
 * implementation) — the view model and both UIs never change.
 */
interface InsightRepository {

    /** Live [Insight] for [period], recomputed on every local diary write and every
     *  applied server change. No network. */
    fun observeInsight(period: InsightPeriod, zone: TimeZone): Flow<Insight>

    /** Bring other devices' entries in via the diary sync (never on the render path;
     *  offline is a no-op that leaves the local render intact). */
    suspend fun refresh(): Result<Unit>

    /** The diary sync status, surfaced as the insight screen's quiet banner. */
    val syncStatus: StateFlow<SyncStatus>
}

/**
 * Default [InsightRepository]. Adds no new persistence: it observes the existing
 * [DiaryRepository] entry stream, folds each period with [InsightCalculator], and
 * composes the read-back with [composer] (the rule-based, offline default).
 */
class DefaultInsightRepository(
    private val diaryRepository: DiaryRepository,
    private val composer: SummaryComposer = RuleBasedSummaryComposer,
) : InsightRepository {

    override val syncStatus: StateFlow<SyncStatus> = diaryRepository.syncStatus

    @OptIn(ExperimentalCoroutinesApi::class)
    override fun observeInsight(period: InsightPeriod, zone: TimeZone): Flow<Insight> =
        diaryRepository.observeEntries().mapLatest { entries ->
            val summary = InsightCalculator.calculate(period, entries, zone)
            Insight(summary, composer.compose(summary))
        }

    override suspend fun refresh(): Result<Unit> = diaryRepository.sync()
}
