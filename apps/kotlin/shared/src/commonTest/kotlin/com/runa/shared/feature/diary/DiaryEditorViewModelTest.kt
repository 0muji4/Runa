package com.runa.shared.feature.diary

import com.runa.shared.core.state.SyncPhase
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import kotlinx.datetime.Instant
import kotlin.test.AfterTest
import kotlin.test.BeforeTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

/** createEntry / updateEntry の呼ばれ方を記録する [DiaryRepository]。 */
private class RecordingDiaryRepository(private var stored: DiaryEntry? = null) : DiaryRepository {
    val createdBodies = mutableListOf<String>()
    val createdAt = mutableListOf<Instant?>()
    val updatedBodies = mutableListOf<Pair<String, String>>()
    private var seq = 0

    override fun observeEntries(): Flow<List<DiaryEntry>> = MutableStateFlow(emptyList())
    override val syncStatus: StateFlow<SyncPhase> = MutableStateFlow(SyncPhase.Idle)

    override suspend fun getEntry(clientId: String): DiaryEntry? =
        stored?.takeIf { it.clientId == clientId }

    override suspend fun createEntry(bodyText: String, mood: String?, createdAt: Instant?): DiaryEntry {
        createdBodies += bodyText
        this.createdAt += createdAt
        seq += 1
        return DiaryEntry(
            clientId = "created-$seq",
            serverId = null,
            bodyText = bodyText,
            mood = mood,
            createdAtEpochMs = 0L,
            updatedAtEpochMs = 0L,
            syncState = SyncState.PendingCreate,
        )
    }

    override suspend fun updateEntry(clientId: String, bodyText: String, mood: String?): Result<Unit> {
        updatedBodies += clientId to bodyText
        return Result.success(Unit)
    }

    override suspend fun deleteEntry(clientId: String): Result<Unit> = Result.success(Unit)
    override suspend fun sync(): Result<Unit> = Result.success(Unit)
}

/**
 * 自動保存の状態機械をピン留めする。#186 まで、日記エディタは回転で view model ごと
 * 作り直され `clientId` が null に戻るため、書き足すたびに新しい日記ができていた。
 * ここでは view model 単体として「最初の保存だけが create、以降は同じ id への update」
 * になることを固定する。
 */
class DiaryEditorViewModelTest {

    @BeforeTest
    fun setUpMain() = Dispatchers.setMain(StandardTestDispatcher())

    @AfterTest
    fun tearDownMain() = Dispatchers.resetMain()

    private fun entry(id: String, body: String) = DiaryEntry(
        clientId = id,
        serverId = null,
        bodyText = body,
        mood = null,
        createdAtEpochMs = 0L,
        updatedAtEpochMs = 0L,
        syncState = SyncState.Synced,
    )

    @Test
    fun blankDraftIsNeverPersisted() = runTest {
        val repo = RecordingDiaryRepository()
        val vm = DiaryEditorViewModel(repo)

        vm.onBodyChange("   ")
        advanceUntilIdle()

        assertTrue(repo.createdBodies.isEmpty(), "空白だけの下書きは保存しない")
        assertTrue(repo.updatedBodies.isEmpty())
    }

    @Test
    fun rapidTypingDebouncesIntoASingleCreate() = runTest {
        val repo = RecordingDiaryRepository()
        val vm = DiaryEditorViewModel(repo)

        vm.onBodyChange("月")
        vm.onBodyChange("月あ")
        vm.onBodyChange("月あかり")
        advanceUntilIdle()

        assertEquals(listOf("月あかり"), repo.createdBodies, "debounce 後に最後の内容が 1 回だけ保存される")
    }

    @Test
    fun continuedTypingUpdatesTheSameEntryInsteadOfCreatingAnother() = runTest {
        val repo = RecordingDiaryRepository()
        val vm = DiaryEditorViewModel(repo)

        vm.onBodyChange("ひとつめ")
        advanceUntilIdle()
        vm.onBodyChange("ひとつめとふたつめ")
        advanceUntilIdle()

        assertEquals(1, repo.createdBodies.size, "2 回目以降に日記が増えてはいけない")
        assertEquals(listOf("created-1" to "ひとつめとふたつめ"), repo.updatedBodies)
    }

    @Test
    fun existingEntryLoadsAndSavesGoToUpdate() = runTest {
        val repo = RecordingDiaryRepository(stored = entry("c1", "もとの本文"))
        val vm = DiaryEditorViewModel(repo, clientId = "c1")
        advanceUntilIdle()

        assertEquals("もとの本文", vm.state.value.bodyText)
        assertEquals(SaveStatus.Saved, vm.state.value.save)

        vm.onBodyChange("書き足した")
        advanceUntilIdle()

        assertTrue(repo.createdBodies.isEmpty(), "既存の日記を開いたら create は呼ばれない")
        assertEquals(listOf("c1" to "書き足した"), repo.updatedBodies)
    }

    @Test
    fun saveNowFlushesWithoutWaitingForTheDebounce() = runTest {
        val repo = RecordingDiaryRepository()
        val vm = DiaryEditorViewModel(repo, autosaveDelayMs = 10_000)

        vm.onBodyChange("いますぐ")
        vm.saveNow()
        advanceUntilIdle()

        assertEquals(listOf("いますぐ"), repo.createdBodies)
    }

    @Test
    fun backdatedNewEntryPassesTheGivenCreatedAt() = runTest {
        val noon = 1_700_000_000_000L
        val repo = RecordingDiaryRepository()
        val vm = DiaryEditorViewModel(repo, createdAtEpochMs = noon)

        vm.onBodyChange("空の日に綴る")
        advanceUntilIdle()

        assertEquals(Instant.fromEpochMilliseconds(noon), repo.createdAt.single())
    }
}
