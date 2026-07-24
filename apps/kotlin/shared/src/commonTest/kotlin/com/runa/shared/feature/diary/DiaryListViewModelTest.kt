package com.runa.shared.feature.diary

import com.runa.shared.core.state.SyncPhase
import com.runa.shared.core.state.UiState
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.launchIn
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.TestScope
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import kotlinx.datetime.Instant
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertIs

/**
 * State-transition consistency for the shared [UiState] the diary list exposes:
 * empty stream → [UiState.Empty]; entries → [UiState.Content] carrying the current
 * [SyncPhase] over the body (never a body-hiding offline state); and offline →
 * online recovery flips the banner phase without dropping the content.
 *
 * The state is a `WhileSubscribed` `stateIn`, so a collector is kept active
 * ([launchIn]) for the upstream to run.
 */
class DiaryListViewModelTest {

    @Test
    fun emptyStreamYieldsEmpty() = runTest {
        val repo = FakeDiaryRepository()
        val vm = DiaryListViewModel(repo, scope(this))
        val job = vm.state.launchIn(this)
        advanceUntilIdle()

        assertEquals(UiState.Empty, vm.state.value)
        job.cancel()
    }

    @Test
    fun entriesYieldContentCarryingSyncPhase() = runTest {
        val repo = FakeDiaryRepository()
        repo.setSync(SyncPhase.Offline)
        repo.setEntries(listOf(entry("a")))
        val vm = DiaryListViewModel(repo, scope(this))
        val job = vm.state.launchIn(this)
        advanceUntilIdle()

        val content = assertIs<UiState.Content<*>>(vm.state.value)
        assertEquals(1, (content.data as List<*>).size)
        assertEquals(SyncPhase.Offline, content.sync)
        job.cancel()
    }

    @Test
    fun offlineRecoversToIdleOnReconnect() = runTest {
        val repo = FakeDiaryRepository()
        repo.setEntries(listOf(entry("a")))
        repo.setSync(SyncPhase.Offline)
        val vm = DiaryListViewModel(repo, scope(this))
        val job = vm.state.launchIn(this)
        advanceUntilIdle()
        assertEquals(SyncPhase.Offline, assertIs<UiState.Content<*>>(vm.state.value).sync)

        repo.setSync(SyncPhase.Idle)
        advanceUntilIdle()
        assertEquals(SyncPhase.Idle, assertIs<UiState.Content<*>>(vm.state.value).sync)
        job.cancel()
    }

    private fun scope(test: TestScope) = CoroutineScope(StandardTestDispatcher(test.testScheduler))

    private fun entry(id: String) = DiaryEntry(
        clientId = id,
        serverId = null,
        bodyText = "body",
        mood = null,
        createdAtEpochMs = 0L,
        updatedAtEpochMs = 0L,
        syncState = SyncState.Synced,
    )
}

/** Minimal in-memory [DiaryRepository]: a controllable entry stream + sync phase. */
private class FakeDiaryRepository : DiaryRepository {
    private val entries = MutableStateFlow<List<DiaryEntry>>(emptyList())
    private val _syncStatus = MutableStateFlow(SyncPhase.Idle)

    fun setEntries(list: List<DiaryEntry>) { entries.value = list }
    fun setSync(phase: SyncPhase) { _syncStatus.value = phase }

    override fun observeEntries(): Flow<List<DiaryEntry>> = entries
    override val syncStatus: StateFlow<SyncPhase> = _syncStatus
    override suspend fun getEntry(clientId: String): DiaryEntry? = entries.value.firstOrNull { it.clientId == clientId }
    override suspend fun createEntry(bodyText: String, mood: String?, createdAt: Instant?): DiaryEntry = error("unused")
    override suspend fun updateEntry(clientId: String, bodyText: String, mood: String?): Result<Unit> = Result.success(Unit)
    override suspend fun deleteEntry(clientId: String): Result<Unit> = Result.success(Unit)
    override suspend fun sync(): Result<Unit> = Result.success(Unit)
}
