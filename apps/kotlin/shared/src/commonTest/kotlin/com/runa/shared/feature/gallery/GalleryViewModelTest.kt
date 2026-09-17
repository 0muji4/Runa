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

private class FakeGalleryRepository : GalleryRepository {
    val images = MutableStateFlow<List<GalleryImage>>(emptyList())

    override fun observeImages(): Flow<List<GalleryImage>> = images
    override val syncStatus: StateFlow<SyncPhase> = MutableStateFlow(SyncPhase.Idle)
    override suspend fun addImage(bytes: ByteArray, width: Int, height: Int, mimeType: String) = Unit
    override suspend fun deleteImage(clientId: String) = Unit
    override suspend fun refresh(): Result<Unit> = Result.success(Unit)
}

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
}
