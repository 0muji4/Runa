package com.runa.shared.feature.notification

import com.russhwolf.settings.MapSettings
import kotlin.test.Test
import kotlin.test.assertEquals

class NotificationSettingsRepositoryTest {

    @Test
    fun defaultsToDisabledAt2200() {
        val repo = DefaultNotificationSettingsRepository(MapSettings())
        assertEquals(false, repo.observeReminderEnabled().value)
        assertEquals(ReminderTime(22, 0), repo.observeReminderTime().value)
    }

    @Test
    fun togglingEmitsImmediately() {
        val repo = DefaultNotificationSettingsRepository(MapSettings())

        repo.setReminderEnabled(true)
        assertEquals(true, repo.observeReminderEnabled().value)

        repo.setReminderEnabled(false)
        assertEquals(false, repo.observeReminderEnabled().value)
    }

    @Test
    fun changingTimeEmitsImmediately() {
        val repo = DefaultNotificationSettingsRepository(MapSettings())

        repo.setReminderTime(ReminderTime(23, 0))

        assertEquals(ReminderTime(23, 0), repo.observeReminderTime().value)
    }

    @Test
    fun persistsAndIsRestoredByAFreshRepository() {
        val settings = MapSettings()

        DefaultNotificationSettingsRepository(settings).apply {
            setReminderEnabled(true)
            setReminderTime(ReminderTime(21, 0))
        }

        // A new repository over the SAME settings models a process restart.
        val restored = DefaultNotificationSettingsRepository(settings)
        assertEquals(true, restored.observeReminderEnabled().value)
        assertEquals(ReminderTime(21, 0), restored.observeReminderTime().value)
    }

    @Test
    fun persistedKeysStayStable() {
        val settings = MapSettings().apply {
            putBoolean("notif.reminder.enabled", true)
            putInt("notif.reminder.hour", 21)
            putInt("notif.reminder.minute", 30)
        }

        val repo = DefaultNotificationSettingsRepository(settings)

        assertEquals(true, repo.observeReminderEnabled().value)
        assertEquals(ReminderTime(21, 30), repo.observeReminderTime().value)
    }
}
