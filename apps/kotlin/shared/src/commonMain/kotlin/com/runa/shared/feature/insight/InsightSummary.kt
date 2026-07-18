package com.runa.shared.feature.insight

import com.runa.shared.feature.diary.DiaryMood
import com.runa.shared.feature.today.moon.MoonPhaseKey

/**
 * The quiet facts an insight period holds — a still映し返し, never a judgement.
 *
 * Deliberately free of kotlinx-datetime types (only [periodType] and plain counts),
 * so it rides across the SKIE boundary into both clients without exporting a
 * datetime library. Entries with no mood are counted in [daysJournaled] /
 * [entryCount] but excluded from [moodDistribution] / [mostFrequentMood] and
 * surfaced separately as [unmoodedCount] — older entries authored before mood was
 * captured are treated as "未選択", shown quietly rather than guessed at.
 */
data class InsightSummary(
    val periodType: InsightPeriodType,
    /** Distinct local dates in the period with at least one entry. */
    val daysJournaled: Int,
    /** Total entries in the period (mood-less ones included). */
    val entryCount: Int,
    /** Entries in the period with no mood recorded (excluded from the distribution). */
    val unmoodedCount: Int,
    /** Per-mood counts in [DiaryMood] declaration order; moods with zero entries are
     *  present with count 0 so the UI can render a stable, calm row. */
    val moodDistribution: List<MoodCount>,
    /** The most-recorded mood, or null if the period holds no moods at all. Ties
     *  resolve deterministically to the earlier [DiaryMood] in declaration order. */
    val mostFrequentMood: DiaryMood?,
    /** Longest run of consecutive journaled days within the period. */
    val longestStreak: Int,
    /** Entry counts bucketed by the moon phase of each entry's day, in synodic
     *  order (new → full → new). The insight screen's hero histogram. */
    val moonOverlap: List<MoonPhaseBucket>,
) {
    /** True when the period holds no entries — the screen shows its Empty state. */
    val isEmpty: Boolean get() = entryCount == 0
}

/** A mood and how many entries in the period carried it. */
data class MoodCount(val mood: DiaryMood, val count: Int)

/** A moon phase and how many of the period's entries fell on a day of that phase. */
data class MoonPhaseBucket(val phaseKey: MoonPhaseKey, val count: Int)
