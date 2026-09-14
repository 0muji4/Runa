package com.runa.shared.feature.diary

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.datetime.Instant

/**
 * Drives the editor for one entry (new when clientId == null). The first non-blank
 * change creates it; later changes autosave on a debounce. A blank draft is never persisted.
 */
class DiaryEditorViewModel(
    private val repository: DiaryRepository,
    clientId: String? = null,
    // Backdate for a new entry from the calendar (that day's local noon); null = now.
    private val createdAtEpochMs: Long? = null,
    private val autosaveDelayMs: Long = 700,
) : ViewModel() {
    private var clientId: String? = clientId

    private val _state = MutableStateFlow(DiaryEditorState())
    val state: StateFlow<DiaryEditorState> = _state.asStateFlow()

    private var saveJob: Job? = null

    init {
        this.clientId?.let { id ->
            viewModelScope.launch {
                repository.getEntry(id)?.let { e ->
                    _state.value = DiaryEditorState(bodyText = e.bodyText, mood = e.mood, save = SaveStatus.Saved)
                }
            }
        }
    }

    fun onBodyChange(text: String) {
        _state.value = _state.value.copy(bodyText = text, save = editingIfNeeded())
        scheduleSave()
    }

    fun onMoodChange(mood: String?) {
        _state.value = _state.value.copy(mood = mood, save = editingIfNeeded())
        scheduleSave()
    }

    /** Flush any pending autosave immediately (call when leaving the screen). */
    fun saveNow() {
        saveJob?.cancel()
        viewModelScope.launch { persist() }
    }

    private fun scheduleSave() {
        saveJob?.cancel()
        saveJob = viewModelScope.launch {
            delay(autosaveDelayMs)
            persist()
        }
    }

    private suspend fun persist() {
        val snapshot = _state.value
        if (snapshot.bodyText.isBlank()) return
        _state.value = snapshot.copy(save = SaveStatus.Saving)

        val result = runCatching {
            val id = clientId
            if (id == null) {
                val createdAt = createdAtEpochMs?.let { Instant.fromEpochMilliseconds(it) }
                clientId = repository.createEntry(snapshot.bodyText, snapshot.mood, createdAt).clientId
            } else {
                repository.updateEntry(id, snapshot.bodyText, snapshot.mood).getOrThrow()
            }
        }
        // Only settle to Saved if the user hasn't typed more since this snapshot.
        val stillCurrent = _state.value.bodyText == snapshot.bodyText && _state.value.mood == snapshot.mood
        _state.value = _state.value.copy(
            save = when {
                result.isFailure -> SaveStatus.Error
                stillCurrent -> SaveStatus.Saved
                else -> SaveStatus.Editing
            },
        )
    }

    private fun editingIfNeeded(): SaveStatus =
        if (_state.value.save == SaveStatus.Saved) SaveStatus.Editing else _state.value.save
}

/** Editor UI state: the draft plus the autosave indicator. */
data class DiaryEditorState(
    val bodyText: String = "",
    val mood: String? = null,
    val save: SaveStatus = SaveStatus.Editing,
)

enum class SaveStatus { Editing, Saving, Saved, Error }
