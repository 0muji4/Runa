package com.runa.shared.feature.insight

import com.runa.shared.core.state.SyncPhase
import com.runa.shared.feature.diary.DiaryRepository
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
 * Insight boundary for the UI. Local-first: [observeInsight] is computed from the
 * on-device diary DB only; [refresh] is the sole network touch.
 */
interface InsightRepository {

    /** Live [Insight] for [period], recomputed on every local diary change. */
    fun observeInsight(period: InsightPeriod, zone: TimeZone): Flow<Insight>

    /** Bring other devices' entries in via the diary sync; offline is a no-op. */
    suspend fun refresh(): Result<Unit>

    val syncStatus: StateFlow<SyncPhase>
}

/** Default [InsightRepository]: folds the [DiaryRepository] stream through [InsightCalculator] + [composer]. */
class DefaultInsightRepository(
    private val diaryRepository: DiaryRepository,
    private val composer: SummaryComposer = RuleBasedSummaryComposer,
) : InsightRepository {

    override val syncStatus: StateFlow<SyncPhase> = diaryRepository.syncStatus

    @OptIn(ExperimentalCoroutinesApi::class)
    override fun observeInsight(period: InsightPeriod, zone: TimeZone): Flow<Insight> =
        diaryRepository.observeEntries().mapLatest { entries ->
            val summary = InsightCalculator.calculate(period, entries, zone)
            Insight(summary, composer.compose(summary))
        }

    override suspend fun refresh(): Result<Unit> = diaryRepository.sync()
}
