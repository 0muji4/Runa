package com.runa.shared.feature.diary

import com.runa.shared.core.state.UiState
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch

/**
 * Drives the diary list. [state] is the shared [UiState]: it is derived from the
 * local DB stream and the repository's [com.runa.shared.core.state.SyncPhase], so it
 * renders instantly from cache and never blocks on the network. Local-first means we
 * almost always have [UiState.Content] or [UiState.Empty]; offline/error ride along
 * as [UiState.Content.sync] (the quiet banner) rather than hiding the list.
 * [UiState.Loading] shows only before the first DB emission. Android collects it
 * directly; iOS observes via SKIE.
 */
class DiaryListViewModel(
    private val repository: DiaryRepository,
    private val scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Default),
) {
    val state: StateFlow<UiState<List<DiaryEntry>>> =
        combine(repository.observeEntries(), repository.syncStatus) { entries, sync ->
            if (entries.isEmpty()) UiState.Empty else UiState.Content(entries, sync)
        }.stateIn(scope, SharingStarted.WhileSubscribed(5_000L), UiState.Loading)

    init {
        // Kick a sync when the list opens; the repository also auto-syncs on
        // connectivity changes.
        refresh()
    }

    /** Pull-to-refresh / on-resume entry point. */
    fun refresh() {
        scope.launch { repository.sync() }
    }

    /** Soft-delete an entry (used by the detail screen's delete action). */
    fun delete(clientId: String) {
        scope.launch { repository.deleteEntry(clientId) }
    }
}
