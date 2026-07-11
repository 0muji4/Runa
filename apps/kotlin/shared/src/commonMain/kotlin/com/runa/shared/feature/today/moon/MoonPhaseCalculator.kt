package com.runa.shared.feature.today.moon

import kotlinx.datetime.LocalDate
import kotlinx.datetime.TimeZone
import kotlinx.datetime.atTime
import kotlinx.datetime.toInstant
import kotlin.math.PI
import kotlin.math.cos
import kotlin.math.floor

/**
 * Offline moon-phase calculator — the domain core of the "today" feature.
 *
 * It is deliberately pure `kotlin.math` over [Double] in `commonMain` with NO
 * platform APIs, so Android (Kotlin/JVM) and iOS (Kotlin/Native) run the exact
 * same IEEE-754 arithmetic and therefore return byte-identical results. That
 * cross-platform identity is asserted by the reference-date suite in commonTest.
 *
 * Algorithm (Meeus-simplified "moon age" method): the age of the moon is the time
 * elapsed since a known reference new moon, taken modulo the mean synodic month.
 * The illuminated fraction follows from the phase angle, and the phase bucket is
 * the age rounded to one of eight equal segments. No ephemeris or network is
 * needed, so the home screen's moon works fully offline.
 *
 * Sources:
 *  - Jean Meeus, *Astronomical Algorithms* (2nd ed.), ch. 49 (phases of the Moon)
 *    and the mean synodic month 29.530588853 days.
 *  - Reference new moon epoch JD 2451550.1 (2000-01-06, the first new moon of
 *    2000), the constant used by the common simplified implementations.
 *
 * Accuracy: new/full moons land within ~1 day of the true instant across the app's
 * date range — ample for the phase icon, name, illumination and 月齢 the home shows.
 */
object MoonPhaseCalculator {
    /** Mean synodic month (new moon to new moon), in days. */
    private const val SYNODIC_MONTH = 29.530588853

    /** Julian Day of the reference new moon (2000-01-06, ~18h UTC). */
    private const val REFERENCE_NEW_MOON_JD = 2451550.1

    /** Julian Day of the Unix epoch (1970-01-01T00:00:00Z). */
    private const val UNIX_EPOCH_JD = 2440587.5

    private const val MILLIS_PER_DAY = 86_400_000.0

    /** The eight phase buckets in synodic order; index 0 and 8 both wrap to new. */
    private val PHASE_ORDER = listOf(
        MoonPhaseKey.NEW_MOON,
        MoonPhaseKey.WAXING_CRESCENT,
        MoonPhaseKey.FIRST_QUARTER,
        MoonPhaseKey.WAXING_GIBBOUS,
        MoonPhaseKey.FULL_MOON,
        MoonPhaseKey.WANING_GIBBOUS,
        MoonPhaseKey.LAST_QUARTER,
        MoonPhaseKey.WANING_CRESCENT,
    )

    /**
     * The moon phase for [date] as seen in [zone]. The day is represented at local
     * noon, a stable mid-day instant that keeps the result on the intended
     * calendar day regardless of the observer's offset.
     */
    fun phaseFor(date: LocalDate, zone: TimeZone): MoonPhase {
        val instant = date.atTime(hour = 12, minute = 0).toInstant(zone)
        val julianDay = instant.toEpochMilliseconds() / MILLIS_PER_DAY + UNIX_EPOCH_JD

        // Age since the reference new moon, wrapped into [0, SYNODIC_MONTH).
        var age = (julianDay - REFERENCE_NEW_MOON_JD) % SYNODIC_MONTH
        if (age < 0) age += SYNODIC_MONTH

        val fraction = age / SYNODIC_MONTH
        val illumination = ((1 - cos(2 * PI * fraction)) / 2).coerceIn(0.0, 1.0)

        // Round the fraction onto one of eight equal buckets (half-up); 8 wraps to 0.
        val index = floor(fraction * 8 + 0.5).toInt() % 8

        return MoonPhase(
            phaseKey = PHASE_ORDER[index],
            illumination = illumination,
            ageDays = age,
        )
    }
}
