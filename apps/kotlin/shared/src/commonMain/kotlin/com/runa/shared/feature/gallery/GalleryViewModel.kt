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

/**
 * Drives the gallery grid (13 ひかりの記録). Derives the shared [UiState] from the
 * local image stream + sync phase, so it renders instantly from cache and works
 * offline. Local-first means we almost always have [UiState.Content] or
 * [UiState.Empty]; offline/sync ride along as [UiState.Content.sync] rather than
 * hiding the grid.
 *
 * The [displayTheme] toggle (monotone ⇔ pink) is exposed as a separate flow because
 * it is grid chrome shown over both content and empty — it is a GALLERY-SCOPED view
 * treatment, NOT the app-wide theme setting and NOT the same as an image's saved
 * [GalleryTheme] (though a newly added image is tagged with the current display theme
 * as its saved mood). The toggle is persisted locally, defaulting to PINK per the
 * confirmed design.
 */
class GalleryViewModel(
    private val repository: GalleryRepository,
) : ViewModel() {
    private val _displayTheme = MutableStateFlow(GalleryDisplayTheme.PINK)

    /** The gallery-scoped display treatment (persisted); grid chrome, shown always. */
    val displayTheme: StateFlow<GalleryDisplayTheme> = _displayTheme.asStateFlow()

    val state: StateFlow<UiState<List<GalleryImage>>> =
        combine(repository.observeImages(), repository.syncStatus) { images, sync ->
            if (images.isEmpty()) UiState.Empty else UiState.Content(images, sync)
        }.stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000L), UiState.Loading)

    init {
        // Restore the persisted display-theme preference (async; PINK until then).
        viewModelScope.launch {
            repository.loadDisplayTheme()?.let { saved ->
                runCatching { GalleryDisplayTheme.valueOf(saved) }.getOrNull()?.let { _displayTheme.value = it }
            }
        }
        // Bring in other devices' images / refresh expired URLs; the grid already renders.
        refresh()
    }

    /** Switch the gallery-scoped display treatment (persisted). */
    fun setDisplayTheme(theme: GalleryDisplayTheme) {
        _displayTheme.value = theme
        viewModelScope.launch { repository.saveDisplayTheme(theme.name) }
    }

    /** Add a picked, already-normalized image; it is tagged with the current display
     *  theme as its saved mood. Bytes come from the platform picker (UI layer). */
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

/**
 * The gallery-scoped display treatment. This is deliberately a SEPARATE type from
 * the app-wide theme setting and from the per-image [GalleryTheme]: it only changes
 * how the grid is rendered (monotone desaturation ⇔ pink duotone), stays inside the
 * gallery, and is never confused with the global theme.
 */
enum class GalleryDisplayTheme { MONOTONE, PINK }

private fun GalleryDisplayTheme.toSavedTheme(): GalleryTheme = when (this) {
    GalleryDisplayTheme.MONOTONE -> GalleryTheme.MONOTONE
    GalleryDisplayTheme.PINK -> GalleryTheme.PINK
}

/**
 * Decode base64 image bytes into a Kotlin [ByteArray] on the Kotlin side. iOS uses
 * this so it can pass picked-image bytes as a single String across the Swift↔Kotlin
 * boundary and get back a [ByteArray] reference — avoiding a slow per-element
 * `KotlinByteArray` build in Swift — then hand it straight to [GalleryViewModel.addImage].
 * (Android passes its `ByteArray` directly and does not need this.)
 */
@OptIn(ExperimentalEncodingApi::class)
fun galleryDecodeBase64(value: String): ByteArray = Base64.decode(value)
