package com.runa.shared.feature.settings

import com.runa.shared.network.dto.ExportDto
import com.runa.shared.network.dto.UserDto
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertIs
import kotlin.test.assertTrue

/** In-memory [SettingsRepository] driving the view-model state machine tests. */
private class FakeSettingsRepository(
    var profile: UserDto = UserDto(id = "u1", displayName = "Runa", authProvider = "email"),
    var deleteResult: Result<Unit> = Result.success(Unit),
) : SettingsRepository {
    var deleteCalled = false

    override suspend fun getProfile(): Result<UserDto> = Result.success(profile)

    override suspend fun updateDisplayName(name: String): Result<UserDto> {
        profile = profile.copy(displayName = name)
        return Result.success(profile)
    }

    override suspend fun exportData(): Result<ExportDto> =
        Result.success(ExportDto(exportedAt = "2026-07-20T00:00:00Z", schemaVersion = 1, user = profile))

    override suspend fun deleteAccount(): Result<Unit> {
        deleteCalled = true
        return deleteResult
    }
}

class AccountViewModelTest {

    @Test
    fun editAndSaveNameUpdatesProfileAndExitsEdit() = runTest {
        val repo = FakeSettingsRepository()
        val vm = AccountViewModel(repo, CoroutineScope(StandardTestDispatcher(testScheduler)))
        advanceUntilIdle()

        vm.startEditName()
        vm.onDisplayNameChange("月子")
        vm.saveName()
        advanceUntilIdle()

        val state = vm.state.value
        assertEquals("月子", state.profile?.displayName)
        assertFalse(state.isEditingName)
    }

    @Test
    fun emptyNameIsRejectedWithoutSaving() = runTest {
        val repo = FakeSettingsRepository()
        val vm = AccountViewModel(repo, CoroutineScope(StandardTestDispatcher(testScheduler)))
        advanceUntilIdle()

        vm.startEditName()
        vm.onDisplayNameChange("   ")
        vm.saveName()
        advanceUntilIdle()

        assertEquals("名前を入力してください", vm.state.value.nameError)
        assertTrue(vm.state.value.isEditingName)
    }

    @Test
    fun exportPreparesJsonAndText() = runTest {
        val repo = FakeSettingsRepository()
        val vm = AccountViewModel(repo, CoroutineScope(StandardTestDispatcher(testScheduler)))
        advanceUntilIdle()

        vm.export()
        advanceUntilIdle()

        val export = vm.state.value.export
        assertIs<ExportStatus.Ready>(export)
        assertTrue(export.json.contains("schema_version"))
        assertTrue(export.text.contains("LUNA データエクスポート"))
    }

    @Test
    fun confirmDeleteCallsRepositoryAndReachesDeleted() = runTest {
        val repo = FakeSettingsRepository()
        val vm = AccountViewModel(repo, CoroutineScope(StandardTestDispatcher(testScheduler)))
        advanceUntilIdle()

        vm.requestDelete()
        assertEquals(DeletionStatus.Confirming, vm.state.value.deletion)

        vm.confirmDelete()
        advanceUntilIdle()

        assertTrue(repo.deleteCalled)
        assertEquals(DeletionStatus.Deleted, vm.state.value.deletion)
    }
}
