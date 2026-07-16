package com.runa.shared.feature.diary

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.datetime.Instant

/**
 * Drives the "write" screen for a single entry — new (clientId == null) or an
 * existing one. Editing is durable and offline-safe: the first non-blank change
 * creates the entry (pending_create), and later changes autosave on a short
 * debounce. An entry left blank is never persisted.
 */
class DiaryEditorViewModel(
    private val repository: DiaryRepository,
    clientId: String? = null,
    // Backdate for a new entry authored from the calendar's empty day (epoch-ms of
    // that day's local noon); null for a plain new entry dated now.
    private val createdAtEpochMs: Long? = null,
    private val scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Default),
    private val autosaveDelayMs: Long = 700,
) {
    // Local id of the entry being edited; assigned once the first save creates it.
    private var clientId: String? = clientId

    private val _state = MutableStateFlow(DiaryEditorState())
    val state: StateFlow<DiaryEditorState> = _state.asStateFlow()

    private var saveJob: Job? = null

    init {
        this.clientId?.let { id ->
            scope.launch {
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
        scope.launch { persist() }
    }

    private fun scheduleSave() {
        saveJob?.cancel()
        saveJob = scope.launch {
            delay(autosaveDelayMs)
            persist()
        }
    }

    private suspend fun persist() {
        val snapshot = _state.value
        if (snapshot.bodyText.isBlank()) return // never persist an empty draft
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

/** Editor UI state: the draft plus a subtle autosave indicator. */
data class DiaryEditorState(
    val bodyText: String = "",
    val mood: String? = null,
    val save: SaveStatus = SaveStatus.Editing,
)

/** Autosave lifecycle for the editor's quiet indicator. */
enum class SaveStatus { Editing, Saving, Saved, Error }
