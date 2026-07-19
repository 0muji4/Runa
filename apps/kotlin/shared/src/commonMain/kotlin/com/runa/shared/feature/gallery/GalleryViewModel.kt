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
import kotlin.io.encoding.Base64
import kotlin.io.encoding.ExperimentalEncodingApi

/**
 * Drives the gallery grid (13 ひかりの記録). Derives [state] from the local image
 * stream + sync status, so it renders instantly from cache and works offline, and
 * holds the client-only display-theme toggle.
 *
 * The display theme (monotone ⇔ pink) is a GALLERY-SCOPED view treatment that
 * re-grades the whole grid — it is NOT the app-wide theme setting, and NOT the same
 * as an image's saved [GalleryTheme] (though a newly added image is tagged with the
 * current display theme as its saved mood). The toggle is persisted locally so it is
 * remembered, defaulting to PINK per the confirmed design.
 */
class GalleryViewModel(
    private val repository: GalleryRepository,
    private val scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Default),
) {
    private val displayTheme = MutableStateFlow(GalleryDisplayTheme.PINK)

    val state: StateFlow<GalleryUiState> =
        combine(repository.observeImages(), repository.syncStatus, displayTheme) { images, sync, theme ->
            val banner = sync.toBanner()
            if (images.isEmpty()) {
                GalleryUiState.Empty(theme, banner)
            } else {
                GalleryUiState.Content(images, theme, banner)
            }
        }.stateIn(scope, SharingStarted.WhileSubscribed(5_000L), GalleryUiState.Loading)

    init {
        // Restore the persisted display-theme preference (async; PINK until then).
        scope.launch {
            repository.loadDisplayTheme()?.let { saved ->
                runCatching { GalleryDisplayTheme.valueOf(saved) }.getOrNull()?.let { displayTheme.value = it }
            }
        }
        // Bring in other devices' images / refresh expired URLs; the grid already renders.
        refresh()
    }

    /** Switch the gallery-scoped display treatment (persisted). */
    fun setDisplayTheme(theme: GalleryDisplayTheme) {
        displayTheme.value = theme
        scope.launch { repository.saveDisplayTheme(theme.name) }
    }

    /** Add a picked, already-normalized image; it is tagged with the current display
     *  theme as its saved mood. Bytes come from the platform picker (UI layer). */
    fun addImage(bytes: ByteArray, width: Int, height: Int, mimeType: String) {
        scope.launch { repository.addImage(bytes, width, height, mimeType, displayTheme.value.toSavedTheme()) }
    }

    fun deleteImage(clientId: String) {
        scope.launch { repository.deleteImage(clientId) }
    }

    fun refresh() {
        scope.launch { repository.refresh() }
    }
}

/**
 * Gallery grid UI state. Local-first means we almost always have [Content] or
 * [Empty]; offline/sync ride along as a quiet [banner] rather than hiding the grid.
 */
sealed interface GalleryUiState {
    data object Loading : GalleryUiState
    data class Content(
        val images: List<GalleryImage>,
        val displayTheme: GalleryDisplayTheme,
        val banner: GalleryBanner,
    ) : GalleryUiState
    data class Empty(
        val displayTheme: GalleryDisplayTheme,
        val banner: GalleryBanner,
    ) : GalleryUiState
    data class Error(val banner: GalleryBanner) : GalleryUiState
}

/**
 * The gallery-scoped display treatment. This is deliberately a SEPARATE type from
 * the app-wide theme setting and from the per-image [GalleryTheme]: it only changes
 * how the grid is rendered (monotone desaturation ⇔ pink duotone), stays inside the
 * gallery, and is never confused with the global theme.
 */
enum class GalleryDisplayTheme { MONOTONE, PINK }

/** The quiet status line shown on the gallery screen. */
enum class GalleryBanner { None, Syncing, Offline, Error }

private fun GallerySyncStatus.toBanner(): GalleryBanner = when (this) {
    GallerySyncStatus.Idle -> GalleryBanner.None
    GallerySyncStatus.Syncing -> GalleryBanner.Syncing
    GallerySyncStatus.Offline -> GalleryBanner.Offline
    GallerySyncStatus.Error -> GalleryBanner.Error
}

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
