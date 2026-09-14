package com.runa.shared.feature.gallery

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.runa.shared.core.state.UiState
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import kotlin.io.encoding.Base64
import kotlin.io.encoding.ExperimentalEncodingApi

/** Drives the gallery grid (13 ギャラリー). [displayTheme] is a GALLERY-SCOPED view treatment,
 *  NOT the app-wide theme setting and NOT an image's saved [GalleryTheme]. */
class GalleryViewModel(
    private val repository: GalleryRepository,
) : ViewModel() {
    private val _displayTheme = MutableStateFlow(GalleryDisplayTheme.PINK)

    val displayTheme: StateFlow<GalleryDisplayTheme> = _displayTheme.asStateFlow()

    val state: StateFlow<UiState<List<GalleryImage>>> =
        combine(repository.observeImages(), repository.syncStatus) { images, sync ->
            if (images.isEmpty()) UiState.Empty else UiState.Content(images, sync)
        }.stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000L), UiState.Loading)

    init {
        viewModelScope.launch {
            repository.loadDisplayTheme()?.let { saved ->
                runCatching { GalleryDisplayTheme.valueOf(saved) }.getOrNull()?.let { _displayTheme.value = it }
            }
        }
        refresh()
    }

    fun setDisplayTheme(theme: GalleryDisplayTheme) {
        _displayTheme.value = theme
        viewModelScope.launch { repository.saveDisplayTheme(theme.name) }
    }

    /** Add a picked, already-normalized image; tagged with the current display theme as its saved mood. */
    fun addImage(bytes: ByteArray, width: Int, height: Int, mimeType: String) {
        viewModelScope.launch { repository.addImage(bytes, width, height, mimeType, _displayTheme.value.toSavedTheme()) }
    }

    fun deleteImage(clientId: String) {
        viewModelScope.launch { repository.deleteImage(clientId) }
    }

    fun refresh() {
        viewModelScope.launch { repository.refresh() }
    }
}

/** Gallery-scoped display treatment; deliberately separate from the app-wide theme and the per-image [GalleryTheme]. */
enum class GalleryDisplayTheme { MONOTONE, PINK }

private fun GalleryDisplayTheme.toSavedTheme(): GalleryTheme = when (this) {
    GalleryDisplayTheme.MONOTONE -> GalleryTheme.MONOTONE
    GalleryDisplayTheme.PINK -> GalleryTheme.PINK
}

/** Decodes base64 image bytes for iOS, which passes picked-image bytes as one String
 *  across Swift↔Kotlin instead of building a slow per-element `KotlinByteArray`. */
@OptIn(ExperimentalEncodingApi::class)
fun galleryDecodeBase64(value: String): ByteArray = Base64.decode(value)
