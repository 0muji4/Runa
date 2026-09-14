package com.runa.shared.feature.todaymoon

import com.runa.shared.feature.today.moon.MoonPhaseCalculator
import com.runa.shared.feature.today.moon.MoonPhaseKey
import kotlinx.datetime.Clock
import kotlinx.datetime.TimeZone
import kotlinx.datetime.toLocalDateTime

/** The 今日の月 payload: today's moon, its phrase, and the next principal phase. Computed on-device. */
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
