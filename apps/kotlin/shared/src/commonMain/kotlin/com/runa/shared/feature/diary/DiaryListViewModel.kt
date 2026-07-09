package com.runa.shared.feature.diary

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch

/**
 * Drives the diary list. [state] is derived from the local DB stream and the
 * repository's [SyncStatus], so it renders instantly from cache and never blocks
 * on the network. Android collects it directly; iOS observes via SKIE.
 */
class DiaryListViewModel(
    private val repository: DiaryRepository,
    private val scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Default),
) {
    val state: StateFlow<DiaryListState> =
        combine(repository.observeEntries(), repository.syncStatus) { entries, sync ->
            val banner = sync.toBanner()
            if (entries.isEmpty()) DiaryListState.Empty(banner) else DiaryListState.Content(entries, banner)
        }.stateIn(scope, SharingStarted.WhileSubscribed(5_000L), DiaryListState.Loading)

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

/**
 * List UI state. Local-first means we almost always have [Content] or [Empty];
 * offline/error are carried as a subtle [SyncBanner] over them rather than as
 * body-hiding states. [Loading] only shows before the first DB emission.
 */
sealed interface DiaryListState {
    data object Loading : DiaryListState
    data class Content(val entries: List<DiaryEntry>, val banner: SyncBanner) : DiaryListState
    data class Empty(val banner: SyncBanner) : DiaryListState
}

/** The quiet status line shown above the list. */
enum class SyncBanner { None, Syncing, Offline, Error }

private fun SyncStatus.toBanner(): SyncBanner = when (this) {
    SyncStatus.Idle -> SyncBanner.None
    SyncStatus.Syncing -> SyncBanner.Syncing
    SyncStatus.Offline -> SyncBanner.Offline
    SyncStatus.Error -> SyncBanner.Error
}
