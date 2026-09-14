package com.runa.shared.feature.gallery

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch

/** Drives the lightbox (14 画像詳細) over the same local image stream as the grid;
 *  when the focused image disappears the state becomes [Dismissed]. */
class ImageDetailViewModel(
    private val repository: GalleryRepository,
    startClientId: String,
) : ViewModel() {
    private val focused = MutableStateFlow(startClientId)

    val state: StateFlow<ImageDetailUiState> =
        combine(repository.observeImages(), focused) { images, clientId ->
            val index = images.indexOfFirst { it.clientId == clientId }
            when {
                images.isEmpty() -> ImageDetailUiState.Dismissed
                index < 0 -> ImageDetailUiState.Dismissed
                else -> ImageDetailUiState.Viewing(images, index)
            }
        }.stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000L), ImageDetailUiState.Loading)

    fun focus(clientId: String) {
        focused.value = clientId
    }

    fun delete(clientId: String) {
        viewModelScope.launch { repository.deleteImage(clientId) }
    }
}

/** Lightbox UI state over the shared image list. */
sealed interface ImageDetailUiState {
    data object Loading : ImageDetailUiState
    data class Viewing(val images: List<GalleryImage>, val index: Int) : ImageDetailUiState
    /** The focused image is gone (deleted) — the UI should close the lightbox. */
    data object Dismissed : ImageDetailUiState
}
