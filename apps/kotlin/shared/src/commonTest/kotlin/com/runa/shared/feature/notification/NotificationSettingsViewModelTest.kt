package com.runa.shared.feature.notification

import com.russhwolf.settings.MapSettings
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import kotlin.test.AfterTest
import kotlin.test.BeforeTest
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

    // The view model now runs on viewModelScope (Dispatchers.Main), so Main has to be a
    // test dispatcher. runTest picks up its scheduler, keeping the test deterministic.
    @BeforeTest
    fun setUpMain() = Dispatchers.setMain(UnconfinedTestDispatcher())

    @AfterTest
    fun tearDownMain() = Dispatchers.resetMain()

    @Test
    fun togglingAndChangingTimeUpdateStateAndInstructTheRepository() = runTest {
        val scheduler = RecordingScheduler()
        val repo = DefaultNotificationSettingsRepository(MapSettings(), scheduler)
        val vm = NotificationSettingsViewModel(repo)

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
