package com.runa.shared.feature.todaymoon

import com.runa.shared.feature.today.moon.MoonPhaseCalculator
import com.runa.shared.feature.today.moon.MoonPhaseKey
import kotlinx.datetime.Clock
import kotlinx.datetime.TimeZone
import kotlinx.datetime.toLocalDateTime

/**
 * The composed payload for 15 今日の月: today's moon (phase, illumination, 月齢) plus
 * a quiet, phase-specific line and the next principal phase to come. Everything is
 * computed on-device — no network — so the screen works fully offline (DoD#5).
 *
 * [dateLabel] / [nextPhaseDateLabel] are pre-formatted (e.g. "7月4日") so the UIs
 * need not depend on kotlinx-datetime (matching the diary/today conventions).
 */
data class TodayMoon(
    val dateLabel: String,
    val phaseKey: MoonPhaseKey,
    val illumination: Double,
    val ageDays: Double,
    val phrase: String,
    val nextPhaseDateLabel: String,
    val nextPhaseKey: MoonPhaseKey,
)

/** Loads the 今日の月 payload. Pure and offline. */
interface TodayMoonRepository {
    fun getTodayMoon(zone: TimeZone): TodayMoon
}

/**
 * Default [TodayMoonRepository]. Reuses the shared [MoonPhaseCalculator] for both
 * the current phase and the next principal phase, and [MoonPhrases] for the line.
 */
class DefaultTodayMoonRepository(
    private val clock: Clock = Clock.System,
) : TodayMoonRepository {

    override fun getTodayMoon(zone: TimeZone): TodayMoon {
        val today = clock.now().toLocalDateTime(zone).date
        val phase = MoonPhaseCalculator.phaseFor(today, zone)
        val next = MoonPhaseCalculator.nextPrincipalPhase(today, zone)
        return TodayMoon(
            dateLabel = "${today.monthNumber}月${today.dayOfMonth}日",
            phaseKey = phase.phaseKey,
            illumination = phase.illumination,
            ageDays = phase.ageDays,
            phrase = MoonPhrases.phraseFor(phase.phaseKey),
            nextPhaseDateLabel = "${next.date.monthNumber}月${next.date.dayOfMonth}日",
            nextPhaseKey = next.phaseKey,
        )
    }
}
