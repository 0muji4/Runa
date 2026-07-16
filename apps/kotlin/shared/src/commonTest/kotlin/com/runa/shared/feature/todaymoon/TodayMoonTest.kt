package com.runa.shared.feature.todaymoon

import com.runa.shared.feature.today.moon.MoonPhaseCalculator
import com.runa.shared.feature.today.moon.MoonPhaseKey
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant
import kotlinx.datetime.LocalDate
import kotlinx.datetime.TimeZone
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

/**
 * 今日の月 composition + the phrase table + the "next principal phase" scan. All
 * offline / pure math, so a green run proves Android and iOS agree.
 */
class TodayMoonTest {

    private val utc = TimeZone.UTC

    @Test
    fun everyPhaseHasATwoLinePhrase() {
        for (key in MoonPhaseKey.entries) {
            val phrase = MoonPhrases.phraseFor(key)
            assertTrue(phrase.isNotBlank(), "phrase for $key")
            assertTrue(phrase.contains('\n'), "phrase for $key should be two lines")
        }
        // The one line fixed by the confirmed design.
        assertEquals("満ちた月は、\n手ばなすための夜。", MoonPhrases.phraseFor(MoonPhaseKey.FULL_MOON))
    }

    @Test
    fun getTodayMoonComposesTheFullMoonDay() {
        // A verified full moon (see MoonPhaseCalculatorTest): 2024-12-15, 月齢 ~14.5.
        val repo = DefaultTodayMoonRepository(clock = FixedClock(Instant.parse("2024-12-15T03:00:00Z")))
        val moon = repo.getTodayMoon(utc)

        assertEquals("12月15日", moon.dateLabel)
        assertEquals(MoonPhaseKey.FULL_MOON, moon.phaseKey)
        assertTrue(moon.illumination > 0.95, "illumination ${moon.illumination}")
        assertTrue(moon.ageDays in 14.0..15.0, "月齢 ${moon.ageDays}")
        assertEquals(MoonPhrases.phraseFor(MoonPhaseKey.FULL_MOON), moon.phrase)
        // Next principal phase after this full moon is the last quarter (2024-12-21).
        assertEquals(MoonPhaseKey.LAST_QUARTER, moon.nextPhaseKey)
        assertEquals("12月21日", moon.nextPhaseDateLabel)
    }

    @Test
    fun nextPrincipalPhaseAfterNewMoonIsFirstQuarter() {
        // 2024-12-01 is a verified new moon; the next principal phase is the first quarter.
        val next = MoonPhaseCalculator.nextPrincipalPhase(LocalDate(2024, 12, 1), utc)
        assertEquals(MoonPhaseKey.FIRST_QUARTER, next.phaseKey)
        assertTrue(next.date > LocalDate(2024, 12, 1), "must be strictly after")
    }
}

private class FixedClock(private val instant: Instant) : Clock {
    override fun now(): Instant = instant
}
