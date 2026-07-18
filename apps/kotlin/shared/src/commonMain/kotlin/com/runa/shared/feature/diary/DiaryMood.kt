package com.runa.shared.feature.diary

/**
 * The quiet, poetic mood a diary entry may carry — a deliberately small, gentle
 * set (LUNA never reduces feeling to a number). [value] is the string persisted
 * through the DB and the wire (a diary entry's `mood` column / DTO field); it is
 * the stable identity, so it must not change once data exists. [labelJa] is the
 * Japanese word shown in the editor's mood picker and in the insights read-back.
 *
 * This canonical set lives in `shared` (not each platform) so the insight
 * aggregation and BOTH clients reference one source of truth — a prerequisite for
 * the cross-platform-identical results the insight feature guarantees. Runa is a
 * Japanese-only product, so the label lives here too (same reasoning as
 * [com.runa.shared.feature.today.moon.moonPhaseNameJa]), which also lets the iOS
 * side pass a [DiaryMood] straight back into Kotlin rather than switching on the
 * SKIE-bridged enum.
 *
 * Declaration order is the display / aggregation order.
 */
enum class DiaryMood(val value: String, val labelJa: String) {
    Calm("calm", "しずか"),
    Gentle("gentle", "おだやか"),
    Tired("tired", "つかれ"),
    Hopeful("hopeful", "のぞみ"),
    Heavy("heavy", "おもい");

    companion object {
        /** The [DiaryMood] for a persisted [value], or null for an unknown/absent one
         *  (older entries authored before mood was captured resolve to null). */
        fun fromValue(value: String?): DiaryMood? = entries.firstOrNull { it.value == value }
    }
}

/**
 * Free-function accessors over [DiaryMood], provided so the iOS side can drive the
 * editor's mood picker from the single shared source without depending on the
 * SKIE-bridged enum's synthesized `allCases`/property access — the same reason the
 * moon phase is read through [com.runa.shared.feature.today.moon.moonPhaseNameJa].
 */

/** All moods in display / aggregation order (for the editor's chip row). */
fun diaryMoods(): List<DiaryMood> = DiaryMood.entries

/** The persisted string value of a mood (what the editor writes / the aggregation reads). */
fun diaryMoodValue(mood: DiaryMood): String = mood.value

/** The Japanese label for a mood. */
fun diaryMoodLabelJa(mood: DiaryMood): String = mood.labelJa
