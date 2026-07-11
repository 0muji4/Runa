package com.runa.shared.feature.today

import com.runa.shared.network.dto.SongDto

/**
 * UI state for the "past songs" archive. [songs] is the paged archive (newest
 * first); [history] is the local play log shown alongside it. [canLoadMore] drives
 * the paging affordance.
 */
data class ArchiveUiState(
    val songs: List<SongDto> = emptyList(),
    val history: List<SongHistoryEntry> = emptyList(),
    val isLoading: Boolean = false,
    val canLoadMore: Boolean = false,
    val error: String? = null,
)
