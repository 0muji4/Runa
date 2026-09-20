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

@OptIn(ExperimentalCoroutinesApi::class)
class NotificationSettingsViewModelTest {

    // The view model runs on viewModelScope (Dispatchers.Main), so Main must be a test dispatcher.
    @BeforeTest
    fun setUpMain() = Dispatchers.setMain(UnconfinedTestDispatcher())

    @AfterTest
    fun tearDownMain() = Dispatchers.resetMain()

    @Test
    fun togglingAndChangingTimeUpdateStateAndTheRepository() = runTest {
        val repo = DefaultNotificationSettingsRepository(MapSettings())
        val vm = NotificationSettingsViewModel(repo)

        assertFalse(vm.state.value.enabled)
        assertEquals(ReminderTime(22, 0), vm.state.value.time)
        assertEquals(ReminderTime.Presets, vm.state.value.presets)

        vm.onToggle(true)
        assertTrue(vm.state.value.enabled)
        assertTrue(repo.observeReminderEnabled().value)

        vm.onSelectTime(ReminderTime(23, 0))
        assertEquals(ReminderTime(23, 0), vm.state.value.time)
        assertEquals(ReminderTime(23, 0), repo.observeReminderTime().value)
    }
}
