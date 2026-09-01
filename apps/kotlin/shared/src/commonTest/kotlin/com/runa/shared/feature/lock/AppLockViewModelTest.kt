package com.runa.shared.feature.lock

import com.russhwolf.settings.MapSettings
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import kotlin.test.AfterTest
import kotlin.test.BeforeTest
import kotlin.test.Test
import kotlin.test.assertEquals

/** Scriptable biometric double: choose the availability and the prompt outcome. */
private class FakeBiometricAuthenticator(
    var availabilityResult: BiometricAvailability = BiometricAvailability.AVAILABLE,
    var authResult: BiometricResult = BiometricResult.Success,
) : BiometricAuthenticator {
    var authenticateCalls = 0
    override fun availability(): BiometricAvailability = availabilityResult
    override suspend fun authenticate(): BiometricResult {
        authenticateCalls++
        return authResult
    }
}

@OptIn(ExperimentalCoroutinesApi::class)
class AppLockViewModelTest {

    // The view model now runs on viewModelScope (Dispatchers.Main), so Main has to be a
    // test dispatcher. runTest picks up its scheduler, keeping the test deterministic.
    @BeforeTest
    fun setUpMain() = Dispatchers.setMain(StandardTestDispatcher())

    @AfterTest
    fun tearDownMain() = Dispatchers.resetMain()

    private fun repo(enabled: Boolean): AppLockRepository =
        DefaultAppLockRepository(MapSettings()).apply { setLockEnabled(enabled) }

    @Test
    fun startsLockedWhenEnabled() = runTest {
        val vm = AppLockViewModel(repo(true), FakeBiometricAuthenticator())
        assertEquals(AppLockUiState.Locked, vm.state.value)
    }

    @Test
    fun startsUnlockedWhenDisabled() = runTest {
        val vm = AppLockViewModel(repo(false), FakeBiometricAuthenticator())
        assertEquals(AppLockUiState.Unlocked, vm.state.value)
    }

    @Test
    fun successfulAuthenticationUnlocks() = runTest {
        val auth = FakeBiometricAuthenticator(authResult = BiometricResult.Success)
        val vm = AppLockViewModel(repo(true), auth)

        vm.authenticate()
        advanceUntilIdle()

        assertEquals(AppLockUiState.Unlocked, vm.state.value)
        assertEquals(1, auth.authenticateCalls)
    }

    @Test
    fun failedAuthenticationStaysLocked() = runTest {
        val auth = FakeBiometricAuthenticator(authResult = BiometricResult.Failed)
        val vm = AppLockViewModel(repo(true), auth)

        vm.authenticate()
        advanceUntilIdle()

        assertEquals(AppLockUiState.Locked, vm.state.value)
    }

    @Test
    fun unavailableBiometricYieldsUnavailableState() = runTest {
        val auth = FakeBiometricAuthenticator(availabilityResult = BiometricAvailability.UNAVAILABLE)
        val vm = AppLockViewModel(repo(true), auth)

        vm.authenticate()
        advanceUntilIdle()

        assertEquals(AppLockUiState.Unavailable, vm.state.value)
        // Never even presented the prompt (availability screened it out).
        assertEquals(0, auth.authenticateCalls)
    }

    @Test
    fun disablingTheLockUnlocksImmediately() = runTest {
        val repository = repo(true)
        val vm = AppLockViewModel(repository, FakeBiometricAuthenticator())
        assertEquals(AppLockUiState.Locked, vm.state.value)

        repository.setLockEnabled(false)
        advanceUntilIdle()

        assertEquals(AppLockUiState.Unlocked, vm.state.value)
    }

    @Test
    fun returningFromBackgroundRelocksAndReprompts() = runTest {
        val auth = FakeBiometricAuthenticator(authResult = BiometricResult.Success)
        val vm = AppLockViewModel(repo(true), auth)

        vm.authenticate()
        advanceUntilIdle()
        assertEquals(AppLockUiState.Unlocked, vm.state.value)

        vm.onAppBackgrounded()
        assertEquals(AppLockUiState.Locked, vm.state.value)

        vm.onAppForegrounded()
        advanceUntilIdle()
        assertEquals(AppLockUiState.Unlocked, vm.state.value)
        assertEquals(2, auth.authenticateCalls)
    }

}
