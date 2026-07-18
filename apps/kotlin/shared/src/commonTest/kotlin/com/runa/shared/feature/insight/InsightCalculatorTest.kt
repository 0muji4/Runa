package com.runa.shared.feature.insight

import com.runa.shared.feature.diary.DiaryEntry
import com.runa.shared.feature.diary.DiaryMood
import com.runa.shared.feature.diary.SyncState
import com.runa.shared.feature.today.moon.MoonPhaseKey
import kotlinx.datetime.Instant
import kotlinx.datetime.LocalDate
import kotlinx.datetime.TimeZone
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull
import kotlin.test.assertTrue

/**
 * The aggregation core: day counts, mood distribution, streak, mood-less exclusion,
 * moon overlap, period boundaries, and timezone grouping — pure commonMain, so a
 * green run proves Android and iOS compute identical insights.
 */
class InsightCalculatorTest {

    private val utc = TimeZone.UTC
    private val tokyo = TimeZone.of("Asia/Tokyo") // UTC+9
    private val december2024 = InsightPeriods.monthlyContaining(LocalDate(2024, 12, 15))

    @Test
    fun countsDaysEntriesMoodsAndStreak() {
        val entries = listOf(
            entry("2024-12-10T09:00:00Z", "calm"),
            entry("2024-12-10T20:00:00Z", "calm"),   // same day → 1 journaled day, 2 entries
            entry("2024-12-11T09:00:00Z", "gentle"),
            entry("2024-12-12T09:00:00Z", null),      // mood未選択
            entry("2024-12-20T09:00:00Z", "calm"),
            entry("2024-11-30T09:00:00Z", "heavy"),   // before the period → excluded
            entry("2025-01-01T09:00:00Z", "tired"),   // after the period → excluded
        )
        val s = InsightCalculator.calculate(december2024, entries, utc)

        assertEquals(4, s.daysJournaled, "distinct in-period days: 10, 11, 12, 20")
        assertEquals(5, s.entryCount, "in-period entries (mood-less included)")
        assertEquals(1, s.unmoodedCount, "the 12th has no mood")
        assertEquals(3, s.moodDistribution.first { it.mood == DiaryMood.Calm }.count)
        assertEquals(1, s.moodDistribution.first { it.mood == DiaryMood.Gentle }.count)
        assertEquals(0, s.moodDistribution.first { it.mood == DiaryMood.Heavy }.count, "out-of-period heavy excluded")
        assertEquals(DiaryMood.Calm, s.mostFrequentMood)
        assertEquals(3, s.longestStreak, "10→11→12 consecutive")
    }

    @Test
    fun moodDistributionCoversAllMoodsInDeclarationOrder() {
        val s = InsightCalculator.calculate(december2024, listOf(entry("2024-12-10T09:00:00Z", "calm")), utc)
        assertEquals(DiaryMood.entries, s.moodDistribution.map { it.mood }, "stable, full, declaration-ordered")
    }

    @Test
    fun moodlessEntriesAreExcludedFromDistributionButStillCounted() {
        val entries = listOf(
            entry("2024-12-05T09:00:00Z", null),
            entry("2024-12-06T09:00:00Z", null),
        )
        val s = InsightCalculator.calculate(december2024, entries, utc)
        assertEquals(2, s.daysJournaled)
        assertEquals(2, s.entryCount)
        assertEquals(2, s.unmoodedCount)
        assertTrue(s.moodDistribution.all { it.count == 0 }, "no moods recorded")
        assertNull(s.mostFrequentMood, "no mood → no most-frequent")
    }

    @Test
    fun mostFrequentMoodTieBreaksByDeclarationOrder() {
        // calm(1) and gentle(1) tie → the earlier-declared mood (Calm) wins deterministically.
        val entries = listOf(
            entry("2024-12-11T09:00:00Z", "gentle"),
            entry("2024-12-10T09:00:00Z", "calm"),
        )
        val s = InsightCalculator.calculate(december2024, entries, utc)
        assertEquals(DiaryMood.Calm, s.mostFrequentMood)
    }

    @Test
    fun moonOverlapBucketsEntriesByPhaseInSynodicOrder() {
        // Verified phases: 2024-12-15 = full moon, 2024-12-08 = first quarter.
        val entries = listOf(
            entry("2024-12-15T12:00:00Z", "calm"),
            entry("2024-12-15T18:00:00Z", "gentle"),
            entry("2024-12-08T12:00:00Z", null),
        )
        val s = InsightCalculator.calculate(december2024, entries, utc)

        assertEquals(MoonPhaseKey.entries, s.moonOverlap.map { it.phaseKey }, "synodic order, all 8 buckets")
        assertEquals(2, s.moonOverlap.first { it.phaseKey == MoonPhaseKey.FULL_MOON }.count)
        assertEquals(1, s.moonOverlap.first { it.phaseKey == MoonPhaseKey.FIRST_QUARTER }.count)
        assertEquals(s.entryCount, s.moonOverlap.sumOf { it.count }, "every entry lands in exactly one bucket")
    }

    @Test
    fun groupingUsesTheLocalDateOfTheZone() {
        // 2024-11-30T20:00Z is still November in UTC but already 2024-12-01 in Tokyo.
        val entries = listOf(entry("2024-11-30T20:00:00Z", "calm"))

        val inUtc = InsightCalculator.calculate(december2024, entries, utc)
        assertEquals(0, inUtc.entryCount, "UTC keeps it in November → outside December")

        val inTokyo = InsightCalculator.calculate(december2024, entries, tokyo)
        assertEquals(1, inTokyo.entryCount, "Tokyo moves it to December 1")
        assertEquals(1, inTokyo.daysJournaled)
    }

    @Test
    fun emptyPeriodIsAllZero() {
        val s = InsightCalculator.calculate(december2024, emptyList(), utc)
        assertTrue(s.isEmpty)
        assertEquals(0, s.daysJournaled)
        assertEquals(0, s.entryCount)
        assertEquals(0, s.unmoodedCount)
        assertEquals(0, s.longestStreak)
        assertNull(s.mostFrequentMood)
        assertTrue(s.moodDistribution.all { it.count == 0 })
        assertTrue(s.moonOverlap.all { it.count == 0 })
    }

    // ---- helpers ----

    private fun entry(instant: String, mood: String?): DiaryEntry {
        val ms = Instant.parse(instant).toEpochMilliseconds()
        return DiaryEntry(
            clientId = instant, serverId = null, bodyText = "夜のことば", mood = mood,
            createdAtEpochMs = ms, updatedAtEpochMs = ms, syncState = SyncState.Synced,
        )
    }
}
