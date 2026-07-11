package com.runa.shared.feature.today

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

/**
 * Shared archive view model. Loads the paged song archive from the backend and
 * mirrors the local play history via [SongRepository.observeSongHistory]. The UI
 * plays a tapped song through the [com.runa.shared.feature.today.player.SongPlayerViewModel].
 */
class SongArchiveViewModel(
    private val repository: SongRepository,
    private val scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Default),
) {
    private val _state = MutableStateFlow(ArchiveUiState(isLoading = true))
    val state: StateFlow<ArchiveUiState> = _state.asStateFlow()

    private var nextCursor: String? = null

    init {
        scope.launch {
            repository.observeSongHistory().collect { history ->
                _state.value = _state.value.copy(history = history)
            }
        }
        loadNextPage(reset = true)
    }

    /** Load the next archive page (or the first page when [reset]). */
    fun loadNextPage(reset: Boolean = false) {
        scope.launch {
            _state.value = _state.value.copy(isLoading = true, error = null)
            _state.value = try {
                val page = repository.getArchive(PAGE_SIZE, if (reset) null else nextCursor)
                nextCursor = page.nextCursor
                val merged = if (reset) page.songs else _state.value.songs + page.songs
                _state.value.copy(
                    songs = merged,
                    isLoading = false,
                    canLoadMore = page.nextCursor != null,
                )
            } catch (e: Exception) {
                _state.value.copy(isLoading = false, error = e.message ?: "unknown error")
            }
        }
    }

    private companion object {
        const val PAGE_SIZE = 20
    }
}
