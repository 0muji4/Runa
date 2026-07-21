package com.runa.shared.feature.notification

import com.russhwolf.settings.MapSettings
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

/** Records scheduler calls so the repository's OS-schedule instructions can be asserted. */
private class FakeLocalNotificationScheduler : LocalNotificationScheduler {
    val scheduled = mutableListOf<ReminderTime>()
    var cancelCount = 0
    override fun scheduleDailyReminder(time: ReminderTime) {
        scheduled += time
    }
    override fun cancel() {
        cancelCount++
    }
}

class NotificationSettingsRepositoryTest {

    @Test
    fun defaultsToDisabledAt2200() {
        val repo = DefaultNotificationSettingsRepository(MapSettings(), FakeLocalNotificationScheduler())
        assertEquals(false, repo.observeReminderEnabled().value)
        assertEquals(ReminderTime(22, 0), repo.observeReminderTime().value)
    }

    @Test
    fun enablingSchedulesAtTheCurrentTime() {
        val scheduler = FakeLocalNotificationScheduler()
        val repo = DefaultNotificationSettingsRepository(MapSettings(), scheduler)

        repo.setReminderEnabled(true)

        assertEquals(ReminderTime(22, 0), scheduler.scheduled.last())
    }

    @Test
    fun disablingCancels() {
        val scheduler = FakeLocalNotificationScheduler()
        val repo = DefaultNotificationSettingsRepository(MapSettings(), scheduler)

        repo.setReminderEnabled(true)
        repo.setReminderEnabled(false)

        assertEquals(false, repo.observeReminderEnabled().value)
        assertTrue(scheduler.cancelCount >= 1)
    }

    @Test
    fun changingTimeReschedulesWhenEnabled() {
        val scheduler = FakeLocalNotificationScheduler()
        val repo = DefaultNotificationSettingsRepository(MapSettings(), scheduler)

        repo.setReminderEnabled(true)
        repo.setReminderTime(ReminderTime(23, 0))

        assertEquals(ReminderTime(23, 0), repo.observeReminderTime().value)
        assertEquals(ReminderTime(23, 0), scheduler.scheduled.last())
    }

    @Test
    fun changingTimeWhileDisabledDoesNotSchedule() {
        val scheduler = FakeLocalNotificationScheduler()
        val repo = DefaultNotificationSettingsRepository(MapSettings(), scheduler)

        repo.setReminderTime(ReminderTime(21, 0))

        assertEquals(ReminderTime(21, 0), repo.observeReminderTime().value)
        assertTrue(scheduler.scheduled.isEmpty())
    }

    @Test
    fun persistsAndIsRestoredByAFreshRepository() {
        val settings = MapSettings()

        DefaultNotificationSettingsRepository(settings, FakeLocalNotificationScheduler()).apply {
            setReminderEnabled(true)
            setReminderTime(ReminderTime(21, 0))
        }

        // A brand-new repository over the SAME settings restores the preference —
        // what a process restart exercises.
        val restored = DefaultNotificationSettingsRepository(settings, FakeLocalNotificationScheduler())
        assertEquals(true, restored.observeReminderEnabled().value)
        assertEquals(ReminderTime(21, 0), restored.observeReminderTime().value)
    }
}
