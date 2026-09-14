package com.runa.shared.feature.diary

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.runa.shared.core.state.UiState
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch

/**
 * Drives the diary list from the local DB stream + sync phase; offline/error ride
 * along as [UiState.Content.sync] rather than hiding the list.
 */
class DiaryListViewModel(
    private val repository: DiaryRepository,
) : ViewModel() {
    val state: StateFlow<UiState<List<DiaryEntry>>> =
        combine(repository.observeEntries(), repository.syncStatus) { entries, sync ->
            if (entries.isEmpty()) UiState.Empty else UiState.Content(entries, sync)
        }.stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000L), UiState.Loading)

    init {
        refresh()
    }

    fun refresh() {
        viewModelScope.launch { repository.sync() }
    }

    fun delete(clientId: String) {
        viewModelScope.launch { repository.deleteEntry(clientId) }
    }
}
