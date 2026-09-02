package com.runa.shared.feature.gallery

import com.runa.shared.core.state.SyncPhase
import com.runa.shared.core.state.UiState
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.launchIn
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import kotlin.test.AfterTest
import kotlin.test.BeforeTest
import kotlin.test.Test
import kotlin.test.assertEquals

private class FakeGalleryRepository(private val savedTheme: String? = null) : GalleryRepository {
    val images = MutableStateFlow<List<GalleryImage>>(emptyList())
    var persistedTheme: String? = null

    override fun observeImages(): Flow<List<GalleryImage>> = images
    override val syncStatus: StateFlow<SyncPhase> = MutableStateFlow(SyncPhase.Idle)
    override suspend fun addImage(bytes: ByteArray, width: Int, height: Int, mimeType: String, theme: GalleryTheme) = Unit
    override suspend fun deleteImage(clientId: String) = Unit
    override suspend fun refresh(): Result<Unit> = Result.success(Unit)
    override suspend fun loadDisplayTheme(): String? = savedTheme
    override suspend fun saveDisplayTheme(value: String) {
        persistedTheme = value
    }
}

/** ギャラリーの空状態と、画面固有の表示テーマの復元・永続化を固定する。 */
class GalleryViewModelTest {

    @BeforeTest
    fun setUpMain() = Dispatchers.setMain(StandardTestDispatcher())

    @AfterTest
    fun tearDownMain() = Dispatchers.resetMain()

    @Test
    fun noImagesYieldsEmpty() = runTest {
        val vm = GalleryViewModel(FakeGalleryRepository())
        val job = vm.state.launchIn(this)
        advanceUntilIdle()

        assertEquals(UiState.Empty, vm.state.value)
        job.cancel()
    }

    @Test
    fun persistedDisplayThemeIsRestoredOnOpen() = runTest {
        val vm = GalleryViewModel(FakeGalleryRepository(savedTheme = "MONOTONE"))
        advanceUntilIdle()

        assertEquals(GalleryDisplayTheme.MONOTONE, vm.displayTheme.value)
    }

    @Test
    fun unknownPersistedThemeFallsBackToTheDefault() = runTest {
        val vm = GalleryViewModel(FakeGalleryRepository(savedTheme = "SEPIA"))
        advanceUntilIdle()

        assertEquals(GalleryDisplayTheme.PINK, vm.displayTheme.value, "壊れた値でも既定に落ちるだけで落ちない")
    }

    @Test
    fun switchingDisplayThemePersistsIt() = runTest {
        val repo = FakeGalleryRepository()
        val vm = GalleryViewModel(repo)
        advanceUntilIdle()

        vm.setDisplayTheme(GalleryDisplayTheme.MONOTONE)
        advanceUntilIdle()

        assertEquals(GalleryDisplayTheme.MONOTONE, vm.displayTheme.value)
        assertEquals("MONOTONE", repo.persistedTheme)
    }
}
