package com.runa.shared.feature.insight

import com.runa.shared.feature.diary.DiaryEntry
import com.runa.shared.feature.diary.DiaryMood
import com.runa.shared.feature.today.moon.MoonPhaseCalculator
import com.runa.shared.feature.today.moon.MoonPhaseKey
import kotlinx.datetime.DateTimeUnit
import kotlinx.datetime.Instant
import kotlinx.datetime.LocalDate
import kotlinx.datetime.TimeZone
import kotlinx.datetime.plus
import kotlinx.datetime.toLocalDateTime

/**
 * The domain core of the insight feature — pure, offline aggregation of a period's
 * diary entries into an [InsightSummary].
 *
 * Like [MoonPhaseCalculator] it is a stateless `object` in `commonMain` using only
 * kotlin stdlib + kotlinx-datetime, so Android (Kotlin/JVM) and iOS (Kotlin/Native)
 * run the exact same logic and therefore return identical results. That
 * cross-platform identity is structural, and asserted by the commonTest suite.
 *
 * Dates come from each entry's `createdAtEpochMs`, resolved to the user's local
 * calendar day in [zone] (the same `Instant → LocalDate` idiom the calendar uses),
 * so the day/week/month boundaries honour the observer's timezone.
 */
object InsightCalculator {

    fun calculate(period: InsightPeriod, entries: List<DiaryEntry>, zone: TimeZone): InsightSummary {
        // Entries whose local day falls inside the window, paired with that day.
        val inPeriod: List<Pair<LocalDate, DiaryEntry>> = entries
            .map { localDate(it, zone) to it }
            .filter { (date, _) -> period.contains(date) }

        val moodCounts: Map<DiaryMood, Int> = inPeriod
            .mapNotNull { (_, entry) -> DiaryMood.fromValue(entry.mood) }
            .groupingBy { it }
            .eachCount()

        return InsightSummary(
            periodType = period.type,
            daysJournaled = inPeriod.mapTo(HashSet()) { (date, _) -> date }.size,
            entryCount = inPeriod.size,
            // Every in-period entry is either a known mood (in moodCounts) or未選択.
            unmoodedCount = inPeriod.size - moodCounts.values.sum(),
            moodDistribution = DiaryMood.entries.map { MoodCount(it, moodCounts[it] ?: 0) },
            // maxByOrNull keeps the first max on ties; DiaryMood.entries order makes
            // that tiebreak deterministic (earlier-declared mood wins).
            mostFrequentMood = DiaryMood.entries.filter { (moodCounts[it] ?: 0) > 0 }
                .maxByOrNull { moodCounts[it] ?: 0 },
            longestStreak = longestStreak(inPeriod.map { (date, _) -> date }),
            moonOverlap = moonOverlap(inPeriod, zone),
        )
    }

    /** Count entries per moon phase (synodic order, new → … → waning crescent). */
    private fun moonOverlap(inPeriod: List<Pair<LocalDate, DiaryEntry>>, zone: TimeZone): List<MoonPhaseBucket> {
        val counts = HashMap<MoonPhaseKey, Int>()
        for ((date, _) in inPeriod) {
            val key = MoonPhaseCalculator.phaseFor(date, zone).phaseKey
            counts[key] = (counts[key] ?: 0) + 1
        }
        return MoonPhaseKey.entries.map { MoonPhaseBucket(it, counts[it] ?: 0) }
    }

    /** Longest run of consecutive calendar days present in [dates]. */
    private fun longestStreak(dates: List<LocalDate>): Int {
        if (dates.isEmpty()) return 0
        val distinct = dates.distinct().sorted() // distinct()/sorted() are commonMain-safe
        var longest = 1
        var run = 1
        for (i in 1 until distinct.size) {
            run = if (distinct[i] == distinct[i - 1].plus(1, DateTimeUnit.DAY)) run + 1 else 1
            if (run > longest) longest = run
        }
        return longest
    }

    private fun localDate(entry: DiaryEntry, zone: TimeZone): LocalDate =
        Instant.fromEpochMilliseconds(entry.createdAtEpochMs).toLocalDateTime(zone).date
}
