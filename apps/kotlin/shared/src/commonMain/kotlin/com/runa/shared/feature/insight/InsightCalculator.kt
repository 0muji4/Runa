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

/** Pure, offline aggregation of a period's diary entries into an [InsightSummary]. */
object InsightCalculator {

    fun calculate(period: InsightPeriod, entries: List<DiaryEntry>, zone: TimeZone): InsightSummary {
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
            unmoodedCount = inPeriod.size - moodCounts.values.sum(),
            moodDistribution = DiaryMood.entries.map { MoodCount(it, moodCounts[it] ?: 0) },
            // maxByOrNull keeps the first max on ties, so the earlier-declared mood wins.
            mostFrequentMood = DiaryMood.entries.filter { (moodCounts[it] ?: 0) > 0 }
                .maxByOrNull { moodCounts[it] ?: 0 },
            longestStreak = longestStreak(inPeriod.map { (date, _) -> date }),
            moonOverlap = moonOverlap(inPeriod, zone),
        )
    }

    /** Entry counts per moon phase, in [MoonPhaseKey] (synodic) order. */
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
        val distinct = dates.distinct().sorted()
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
