package com.runa.shared.feature.insight

import com.runa.shared.feature.diary.DiaryMood
import com.runa.shared.feature.today.moon.MoonPhaseKey

/**
 * Aggregated facts of one insight period. Mood-less entries count in [daysJournaled] /
 * [entryCount] and [unmoodedCount] but not in [moodDistribution] / [mostFrequentMood].
 */
data class InsightSummary(
    val periodType: InsightPeriodType,
    /** Distinct local dates in the period with at least one entry. */
    val daysJournaled: Int,
    val entryCount: Int,
    val unmoodedCount: Int,
    /** Per-mood counts in [DiaryMood] declaration order; zero-count moods are present. */
    val moodDistribution: List<MoodCount>,
    /** Null if the period holds no moods; ties resolve to the earlier [DiaryMood]. */
    val mostFrequentMood: DiaryMood?,
    /** Longest run of consecutive journaled days within the period. */
    val longestStreak: Int,
    /** Entry counts by the moon phase of each entry's day, in synodic order. */
    val moonOverlap: List<MoonPhaseBucket>,
) {
    val isEmpty: Boolean get() = entryCount == 0
}

data class MoodCount(val mood: DiaryMood, val count: Int)

data class MoonPhaseBucket(val phaseKey: MoonPhaseKey, val count: Int)
