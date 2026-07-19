package com.runa.shared.feature.gallery

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch

/**
 * Drives the lightbox (14 画像詳細). Thin: it observes the same local image stream as
 * the grid and tracks which image is focused, so swipe/next/prev page over the
 * already-loaded list and a delete flows straight through to the repository. When the
 * focused image disappears (deleted here or elsewhere) the state becomes [Dismissed]
 * so the UI closes the lightbox.
 */
class ImageDetailViewModel(
    private val repository: GalleryRepository,
    startClientId: String,
    private val scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Default),
) {
    private val focused = MutableStateFlow(startClientId)

    val state: StateFlow<ImageDetailUiState> =
        combine(repository.observeImages(), focused) { images, clientId ->
            val index = images.indexOfFirst { it.clientId == clientId }
            when {
                images.isEmpty() -> ImageDetailUiState.Dismissed
                index < 0 -> ImageDetailUiState.Dismissed
                else -> ImageDetailUiState.Viewing(images, index)
            }
        }.stateIn(scope, SharingStarted.WhileSubscribed(5_000L), ImageDetailUiState.Loading)

    /** Update which image is focused (the UI calls this as the pager swipes). */
    fun focus(clientId: String) {
        focused.value = clientId
    }

    fun delete(clientId: String) {
        scope.launch { repository.deleteImage(clientId) }
    }
}

/** Lightbox UI state over the shared image list. */
sealed interface ImageDetailUiState {
    data object Loading : ImageDetailUiState
    data class Viewing(val images: List<GalleryImage>, val index: Int) : ImageDetailUiState
    /** The focused image is gone (deleted) — the UI should close the lightbox. */
    data object Dismissed : ImageDetailUiState
}
