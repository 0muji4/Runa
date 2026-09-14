package com.runa.android.ui.screens.diary

import com.runa.shared.feature.today.moon.MoonPhaseCalculator
import com.runa.shared.feature.today.moon.moonPhaseNameJa
import java.time.DayOfWeek
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.Locale

/** Diary dates are day + moon phase in the device zone — never a clock time. */
private val dayFormatter: DateTimeFormatter =
    DateTimeFormatter.ofPattern("M月d日", Locale.JAPAN)

/** e.g. 7月4日 */
fun formatDiaryDate(epochMs: Long): String =
    Instant.ofEpochMilli(epochMs).atZone(ZoneId.systemDefault()).format(dayFormatter)

/** e.g. 日曜 */
fun formatDiaryWeekday(epochMs: Long): String {
    val dow = Instant.ofEpochMilli(epochMs).atZone(ZoneId.systemDefault()).dayOfWeek
    return when (dow) {
        DayOfWeek.MONDAY -> "月曜"
        DayOfWeek.TUESDAY -> "火曜"
        DayOfWeek.WEDNESDAY -> "水曜"
        DayOfWeek.THURSDAY -> "木曜"
        DayOfWeek.FRIDAY -> "金曜"
        DayOfWeek.SATURDAY -> "土曜"
        DayOfWeek.SUNDAY -> "日曜"
    }
}

/**
 * Backdate for a past `yyyy-MM-dd` day: local noon, so the entry never slips across
 * the date boundary in any zone.
 */
fun isoDateToNoonEpochMs(isoDate: String): Long =
    LocalDate.parse(isoDate).atTime(12, 0).atZone(ZoneId.systemDefault()).toInstant().toEpochMilli()

/** Moon phase for a diary entry's day: disc geometry + the shared Japanese name. */
data class DiaryMoon(val illumination: Float, val waxing: Boolean, val name: String)

private const val SYNODIC_HALF = 29.530588853 / 2.0

fun diaryMoonFor(epochMs: Long): DiaryMoon {
    val phase = MoonPhaseCalculator.phaseForEpochMillis(epochMs)
    return DiaryMoon(
        illumination = phase.illumination.toFloat(),
        waxing = phase.ageDays < SYNODIC_HALF,
        name = moonPhaseNameJa(phase.phaseKey),
    )
}
