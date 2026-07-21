package com.runa.shared.feature.notification

/**
 * The nightly-reminder notification copy, in shared code so both platforms show
 * the exact same quiet, on-world-view line (the app is Japanese-only, like the
 * shared moon-phase names). "月が出ました" + a gentle invitation to write.
 */
object ReminderNotificationText {
    const val TITLE = "月が出ました"
    const val BODY = "今日を、そっと綴りませんか。"
}
