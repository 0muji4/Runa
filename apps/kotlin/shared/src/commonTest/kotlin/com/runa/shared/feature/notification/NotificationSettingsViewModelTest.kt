package com.runa.shared.feature.notification

import com.russhwolf.settings.MapSettings
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.runTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

private class RecordingScheduler : LocalNotificationScheduler {
    val scheduled = mutableListOf<ReminderTime>()
    override fun scheduleDailyReminder(time: ReminderTime) {
        scheduled += time
    }
    override fun cancel() {}
}

@OptIn(ExperimentalCoroutinesApi::class)
class NotificationSettingsViewModelTest {

    @Test
    fun togglingAndChangingTimeUpdateStateAndInstructTheRepository() = runTest {
        val scheduler = RecordingScheduler()
        val repo = DefaultNotificationSettingsRepository(MapSettings(), scheduler)
        val vm = NotificationSettingsViewModel(repo, CoroutineScope(UnconfinedTestDispatcher(testScheduler)))

        // Seeded from persistence: off, 22:00, three presets.
        assertFalse(vm.state.value.enabled)
        assertEquals(ReminderTime(22, 0), vm.state.value.time)
        assertEquals(ReminderTime.Presets, vm.state.value.presets)

        vm.onToggle(true)
        assertTrue(vm.state.value.enabled)
        assertEquals(ReminderTime(22, 0), scheduler.scheduled.last())

        vm.onSelectTime(ReminderTime(23, 0))
        assertEquals(ReminderTime(23, 0), vm.state.value.time)
        assertEquals(ReminderTime(23, 0), scheduler.scheduled.last())
    }
}
