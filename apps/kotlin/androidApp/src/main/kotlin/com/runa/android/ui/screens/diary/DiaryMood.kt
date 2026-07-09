package com.runa.android.ui.screens.diary

import androidx.annotation.StringRes
import com.runa.android.R

/**
 * The quiet mood options. [value] is the string persisted through the API (and
 * later consumed by the insights slice); [labelRes] is the Japanese label shown
 * in the editor. Kept small and gentle on purpose.
 */
enum class DiaryMood(val value: String, @StringRes val labelRes: Int) {
    Calm("calm", R.string.diary_mood_calm),
    Gentle("gentle", R.string.diary_mood_gentle),
    Tired("tired", R.string.diary_mood_tired),
    Hopeful("hopeful", R.string.diary_mood_hopeful),
    Heavy("heavy", R.string.diary_mood_heavy);

    companion object {
        fun fromValue(value: String?): DiaryMood? = entries.firstOrNull { it.value == value }
    }
}
