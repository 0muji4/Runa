package com.runa.shared.feature.today.moon

import kotlinx.datetime.LocalDate
import kotlinx.datetime.TimeZone
import kotlin.math.abs
import kotlin.math.min
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

/** Expected values come from real astronomical new/full/quarter moons (UTC). */
class MoonPhaseCalculatorTest {

    private val utc = TimeZone.UTC
    private val synodic = 29.530588853

    private fun phase(date: String) =
        MoonPhaseCalculator.phaseFor(LocalDate.parse(date), utc)

    private fun ageFromNew(ageDays: Double) = min(ageDays, synodic - ageDays)

    @Test
    fun knownNewMoonsAreClassifiedAsNew() {
        // 2000-01-06 is the calculator's own epoch.
        for (date in listOf("2000-01-06", "2024-01-11", "2024-12-01", "2025-03-29")) {
            val p = phase(date)
            assertEquals(MoonPhaseKey.NEW_MOON, p.phaseKey, "phaseKey for $date")
            assertTrue(p.illumination < 0.05, "illumination for $date was ${p.illumination}")
            assertTrue(ageFromNew(p.ageDays) < 1.0, "age-from-new for $date was ${p.ageDays}")
        }
    }

    @Test
    fun knownFullMoonsAreClassifiedAsFull() {
        for (date in listOf("2024-01-25", "2024-12-15", "2025-03-14")) {
            val p = phase(date)
            assertEquals(MoonPhaseKey.FULL_MOON, p.phaseKey, "phaseKey for $date")
            assertTrue(p.illumination > 0.95, "illumination for $date was ${p.illumination}")
            assertTrue(abs(p.ageDays - synodic / 2) < 1.1, "age for $date was ${p.ageDays}")
        }
    }

    @Test
    fun knownQuartersAreClassified() {
        val first = phase("2024-12-08")
        assertEquals(MoonPhaseKey.FIRST_QUARTER, first.phaseKey)
        assertTrue(first.illumination in 0.40..0.65, "first-quarter illum ${first.illumination}")

        val last = phase("2024-12-22")
        assertEquals(MoonPhaseKey.LAST_QUARTER, last.phaseKey)
        assertTrue(last.illumination in 0.40..0.65, "last-quarter illum ${last.illumination}")
    }

    @Test
    fun illuminationIncreasesMonotonicallyWhileWaxing() {
        // From the day after the 2024-12-01 new moon up to the 2024-12-15 full moon.
        var previous = -1.0
        for (day in 2..15) {
            val illum = phase("2024-12-${day.toString().padStart(2, '0')}").illumination
            assertTrue(illum > previous, "illumination dropped on 2024-12-$day: $illum <= $previous")
            previous = illum
        }
    }

    @Test
    fun ageAdvancesRoughlyOneDayPerDay() {
        val d1 = phase("2024-12-05").ageDays
        val d2 = phase("2024-12-06").ageDays
        assertTrue(abs((d2 - d1) - 1.0) < 0.01, "one-day age delta was ${d2 - d1}")
    }

    @Test
    fun pinnedValueGuardsCrossPlatformDeterminism() {
        // Computed by the same algorithm at 2024-12-15 noon UTC; a target whose Double math diverges fails here.
        val p = phase("2024-12-15")
        assertEquals(14.478633, p.ageDays, 1e-4, "pinned ageDays")
        assertEquals(0.999070, p.illumination, 1e-4, "pinned illumination")
    }

    @Test
    fun phaseIsStableForTheSameInput() {
        assertEquals(phase("2026-07-11"), phase("2026-07-11"))
    }
}
