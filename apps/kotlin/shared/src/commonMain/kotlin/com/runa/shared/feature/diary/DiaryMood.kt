package com.runa.shared.feature.diary

/**
 * Mood a diary entry may carry. [value] is the key persisted in DB and wire and must
 * not change once data exists. Declaration order is the display / aggregation order.
 */
enum class DiaryMood(val value: String, val labelJa: String) {
    Calm("calm", "しずか"),
    Gentle("gentle", "おだやか"),
    Tired("tired", "つかれ"),
    Hopeful("hopeful", "のぞみ"),
    // 「おもい」は 思い / 重い と読みが割れるので和語名詞の「おもさ」にする。
    Heavy("heavy", "おもさ");

    companion object {
        /** The [DiaryMood] for a persisted [value], or null for an unknown/absent one. */
        fun fromValue(value: String?): DiaryMood? = entries.firstOrNull { it.value == value }
    }
}

/** All moods in display / aggregation order. */
fun diaryMoods(): List<DiaryMood> = DiaryMood.entries

fun diaryMoodValue(mood: DiaryMood): String = mood.value

fun diaryMoodLabelJa(mood: DiaryMood): String = mood.labelJa
