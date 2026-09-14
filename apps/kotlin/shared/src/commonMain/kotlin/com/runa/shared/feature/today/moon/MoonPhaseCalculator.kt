package com.runa.shared.feature.today.moon

import kotlinx.datetime.DateTimeUnit
import kotlinx.datetime.Instant
import kotlinx.datetime.LocalDate
import kotlinx.datetime.TimeZone
import kotlinx.datetime.atTime
import kotlinx.datetime.plus
import kotlinx.datetime.toInstant
import kotlinx.datetime.toLocalDateTime
import kotlin.math.PI
import kotlin.math.cos
import kotlin.math.floor

/**
 * Offline moon-phase calculator, pure `kotlin.math` so both platforms give identical results.
 * Simplified "moon age" method (Meeus, Astronomical Algorithms 2nd ed., ch. 49): age = days
 * since a reference new moon mod the mean synodic month; new/full land within ~1 day of true.
 */
object MoonPhaseCalculator {
    /** Mean synodic month (new moon to new moon), in days. */
    private const val SYNODIC_MONTH = 29.530588853

    /** Julian Day of the reference new moon (2000-01-06, ~18h UTC). */
    private const val REFERENCE_NEW_MOON_JD = 2451550.1

    /** Julian Day of the Unix epoch (1970-01-01T00:00:00Z). */
    private const val UNIX_EPOCH_JD = 2440587.5

    private const val MILLIS_PER_DAY = 86_400_000.0

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

    /** The moon phase for [date] in [zone], evaluated at local noon so it stays on the intended day. */
    fun phaseFor(date: LocalDate, zone: TimeZone): MoonPhase {
        val instant = date.atTime(hour = 12, minute = 0).toInstant(zone)
        val julianDay = instant.toEpochMilliseconds() / MILLIS_PER_DAY + UNIX_EPOCH_JD

        // Age since the reference new moon, wrapped into [0, SYNODIC_MONTH).
        var age = (julianDay - REFERENCE_NEW_MOON_JD) % SYNODIC_MONTH
        if (age < 0) age += SYNODIC_MONTH

        val fraction = age / SYNODIC_MONTH
        val illumination = ((1 - cos(2 * PI * fraction)) / 2).coerceIn(0.0, 1.0)

        // Round half-up onto eight equal buckets; 8 wraps to 0.
        val index = floor(fraction * 8 + 0.5).toInt() % 8

        return MoonPhase(
            phaseKey = PHASE_ORDER[index],
            illumination = illumination,
            ageDays = age,
        )
    }

    /** [phaseFor] on the system-zone calendar day of [epochMillis]. */
    fun phaseForEpochMillis(epochMillis: Long): MoonPhase {
        val zone = TimeZone.currentSystemDefault()
        val date = Instant.fromEpochMilliseconds(epochMillis).toLocalDateTime(zone).date
        return phaseFor(date, zone)
    }

    private val PRINCIPAL_PHASES = setOf(
        MoonPhaseKey.NEW_MOON,
        MoonPhaseKey.FIRST_QUARTER,
        MoonPhaseKey.FULL_MOON,
        MoonPhaseKey.LAST_QUARTER,
    )

    /** The next principal phase (新月 / 上弦 / 満月 / 下弦) strictly after [after], by day-by-day scan. */
    fun nextPrincipalPhase(after: LocalDate, zone: TimeZone): PrincipalPhase {
        var previousKey = phaseFor(after, zone).phaseKey
        var date = after
        repeat(40) {
            date = date.plus(1, DateTimeUnit.DAY)
            val key = phaseFor(date, zone).phaseKey
            if (key != previousKey && key in PRINCIPAL_PHASES) {
                return PrincipalPhase(date, key)
            }
            previousKey = key
        }
        // Unreachable within a synodic month; return the last day scanned as a floor.
        return PrincipalPhase(date, phaseFor(date, zone).phaseKey)
    }
}

data class PrincipalPhase(val date: LocalDate, val phaseKey: MoonPhaseKey)
