package com.runa.shared.feature.notification

/**
 * A nightly-reminder time-of-day. [hour] (0–23) and [minute] (0–59) are local
 * clock components; [label] is pre-formatted "HH:MM" so the UIs never need
 * kotlinx-datetime on their classpath (same reasoning as the today slice's
 * pre-formatted dateLabel). The design (21 通知設定) offers three quiet presets plus
 * a free picker; [Presets] backs the chips and [Default] is the initial 22:00.
 */
data class ReminderTime(val hour: Int, val minute: Int) {

    /** "HH:MM", zero-padded — the big time display and the settings-row value. */
    val label: String
        get() = hour.toString().padStart(2, '0') + ":" + minute.toString().padStart(2, '0')

    companion object {
        /** The design's default nightly time: 22:00. */
        val Default = ReminderTime(22, 0)

        /** The three preset chips shown in 21 通知設定 (21:00 / 22:00 / 23:00). */
        val Presets = listOf(ReminderTime(21, 0), ReminderTime(22, 0), ReminderTime(23, 0))

        /** Build a [ReminderTime], clamping into valid clock ranges. */
        fun of(hour: Int, minute: Int): ReminderTime =
            ReminderTime(hour.coerceIn(0, 23), minute.coerceIn(0, 59))
    }
}
