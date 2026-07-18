package com.runa.shared.feature.insight

import com.runa.shared.feature.diary.DiaryEntry
import com.runa.shared.feature.diary.DiaryRepository
import com.runa.shared.feature.diary.SyncState
import com.runa.shared.feature.diary.SyncStatus
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.runTest
import kotlinx.datetime.Instant
import kotlinx.datetime.LocalDate
import kotlinx.datetime.TimeZone
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

/**
 * The repository wires the calculator + composer over the local diary stream, and
 * the summariser is swappable behind the interface (DoD#5). Fully in commonMain
 * over a fake diary — no DB, no network.
 */
class InsightRepositoryTest {

    private val utc = TimeZone.UTC
    private val december2024 = InsightPeriods.monthlyContaining(LocalDate(2024, 12, 15))

    @Test
    fun observeInsightAggregatesTheLocalDiaryStream() = runTest {
        val diary = FakeDiaryRepository()
        diary.setEntries(listOf(entry("2024-12-10T09:00:00Z", "calm"), entry("2024-12-11T09:00:00Z", "gentle")))

        val insight = DefaultInsightRepository(diary).observeInsight(december2024, utc).first()

        assertEquals(2, insight.summary.daysJournaled)
        assertTrue(insight.narrative.body.isNotBlank(), "rule-based body is produced offline")
    }

    @Test
    fun summariserIsSwappableBehindTheInterface() = runTest {
        val diary = FakeDiaryRepository()
        diary.setEntries(listOf(entry("2024-12-10T09:00:00Z", "calm")))
        // A stand-in for a future server-LLM summariser.
        val serverLike = object : SummaryComposer {
            override suspend fun compose(summary: InsightSummary) = InsightNarrative("server-composed", null)
        }

        val insight = DefaultInsightRepository(diary, composer = serverLike).observeInsight(december2024, utc).first()

        assertEquals("server-composed", insight.narrative.body)
        assertEquals(1, insight.summary.daysJournaled, "metrics still come from the local calculator")
    }

    @Test
    fun refreshDelegatesToDiarySync() = runTest {
        val diary = FakeDiaryRepository()
        val result = DefaultInsightRepository(diary).refresh()
        assertTrue(result.isSuccess)
        assertEquals(1, diary.syncCalls)
    }

    // ---- helpers ----

    private fun entry(instant: String, mood: String?): DiaryEntry {
        val ms = Instant.parse(instant).toEpochMilliseconds()
        return DiaryEntry(instant, null, "x", mood, ms, ms, SyncState.Synced)
    }
}

/** Minimal in-memory [DiaryRepository]: a controllable entry stream + sync counter. */
private class FakeDiaryRepository : DiaryRepository {
    private val entries = MutableStateFlow<List<DiaryEntry>>(emptyList())
    private val _syncStatus = MutableStateFlow(SyncStatus.Idle)
    var syncCalls = 0
        private set

    fun setEntries(list: List<DiaryEntry>) { entries.value = list }

    override fun observeEntries(): Flow<List<DiaryEntry>> = entries
    override val syncStatus: StateFlow<SyncStatus> = _syncStatus
    override suspend fun getEntry(clientId: String): DiaryEntry? = entries.value.firstOrNull { it.clientId == clientId }
    override suspend fun createEntry(bodyText: String, mood: String?, createdAt: Instant?): DiaryEntry = error("unused")
    override suspend fun updateEntry(clientId: String, bodyText: String, mood: String?): Result<Unit> = Result.success(Unit)
    override suspend fun deleteEntry(clientId: String): Result<Unit> = Result.success(Unit)
    override suspend fun sync(): Result<Unit> { syncCalls++; return Result.success(Unit) }
}
