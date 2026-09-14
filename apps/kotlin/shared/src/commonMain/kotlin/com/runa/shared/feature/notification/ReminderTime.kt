package com.runa.shared.feature.notification

/** A nightly-reminder local time-of-day. [label] is pre-formatted "HH:MM" so the
 *  UIs never need kotlinx-datetime on their classpath. */
data class ReminderTime(val hour: Int, val minute: Int) {

    val label: String
        get() = hour.toString().padStart(2, '0') + ":" + minute.toString().padStart(2, '0')

    companion object {
        val Default = ReminderTime(22, 0)

        /** The preset chips shown in 21 通知設定. */
        val Presets = listOf(ReminderTime(21, 0), ReminderTime(22, 0), ReminderTime(23, 0))

        fun of(hour: Int, minute: Int): ReminderTime =
            ReminderTime(hour.coerceIn(0, 23), minute.coerceIn(0, 59))
    }
}
