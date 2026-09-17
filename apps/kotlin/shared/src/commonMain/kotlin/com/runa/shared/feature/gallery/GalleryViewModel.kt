package com.runa.shared.feature.gallery

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.runa.shared.core.state.UiState
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import kotlin.io.encoding.Base64
import kotlin.io.encoding.ExperimentalEncodingApi

/** Drives the gallery grid (13 ギャラリー); photos render as they are, with no color grade. */
class GalleryViewModel(
    private val repository: GalleryRepository,
) : ViewModel() {
    val state: StateFlow<UiState<List<GalleryImage>>> =
        combine(repository.observeImages(), repository.syncStatus) { images, sync ->
            if (images.isEmpty()) UiState.Empty else UiState.Content(images, sync)
        }.stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000L), UiState.Loading)

    init {
        refresh()
    }

    /** Add a picked, already-normalized image. */
    fun addImage(bytes: ByteArray, width: Int, height: Int, mimeType: String) {
        viewModelScope.launch { repository.addImage(bytes, width, height, mimeType) }
    }

    fun deleteImage(clientId: String) {
        viewModelScope.launch { repository.deleteImage(clientId) }
    }

    fun refresh() {
        viewModelScope.launch { repository.refresh() }
    }
}

/** Decodes base64 image bytes for iOS, which passes picked-image bytes as one String
 *  across Swift↔Kotlin instead of building a slow per-element `KotlinByteArray`. */
@OptIn(ExperimentalEncodingApi::class)
fun galleryDecodeBase64(value: String): ByteArray = Base64.decode(value)
