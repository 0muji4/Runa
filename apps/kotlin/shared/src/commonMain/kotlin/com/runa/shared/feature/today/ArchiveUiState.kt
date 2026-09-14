package com.runa.shared.feature.today

import com.runa.shared.core.state.AppError
import com.runa.shared.network.dto.SongDto

/** UI state for the song archive: paged [songs] (newest first) plus the local play [history]. */
data class ArchiveUiState(
    val songs: List<SongDto> = emptyList(),
    val history: List<SongHistoryEntry> = emptyList(),
    val isLoading: Boolean = false,
    val canLoadMore: Boolean = false,
    val error: AppError? = null,
)
