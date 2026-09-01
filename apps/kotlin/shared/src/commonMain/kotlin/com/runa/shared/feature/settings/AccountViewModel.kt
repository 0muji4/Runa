package com.runa.shared.feature.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.runa.shared.core.state.toJaMessage
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/**
 * Drives the account-data screen (23): profile display, display-name editing,
 * data export and account deletion. Exposes a single [state] the UI renders; each
 * action mutates one facet of [AccountUiState] and leaves the others untouched.
 *
 * Deletion does not navigate here — on success the underlying repository ends the
 * session, so the app root's auth-state observer returns to sign-in. [state] still
 * reports [DeletionStatus.Deleted] for any UI that wants to reflect it.
 */
class AccountViewModel(
    private val repository: SettingsRepository,
) : ViewModel() {
    private val _state = MutableStateFlow(AccountUiState())
    val state: StateFlow<AccountUiState> = _state.asStateFlow()

    init {
        loadProfile()
    }

    /** (Re)load the profile. Also resets a stale export status from a prior visit. */
    fun loadProfile() {
        _state.update { it.copy(isLoadingProfile = true, loadError = null, export = ExportStatus.Idle) }
        viewModelScope.launch {
            repository.getProfile()
                .onSuccess { user -> _state.update { it.copy(profile = user, isLoadingProfile = false) } }
                .onFailure { e ->
                    _state.update {
                        it.copy(isLoadingProfile = false, loadError = e.toJaMessage("プロフィールを取得できませんでした。"))
                    }
                }
        }
    }

    // ---- display name editing ----

    fun startEditName() {
        _state.update {
            it.copy(isEditingName = true, displayNameDraft = it.profile?.displayName ?: "", nameError = null)
        }
    }

    fun onDisplayNameChange(value: String) {
        _state.update { it.copy(displayNameDraft = value) }
    }

    fun cancelEditName() {
        _state.update { it.copy(isEditingName = false, nameError = null) }
    }

    fun saveName() {
        val name = _state.value.displayNameDraft.trim()
        if (name.isEmpty()) {
            _state.update { it.copy(nameError = "名前を入力してください。") }
            return
        }
        _state.update { it.copy(isSavingName = true, nameError = null) }
        viewModelScope.launch {
            repository.updateDisplayName(name)
                .onSuccess { user ->
                    _state.update { it.copy(profile = user, isEditingName = false, isSavingName = false) }
                }
                .onFailure { e ->
                    _state.update { it.copy(isSavingName = false, nameError = e.toJaMessage("保存できませんでした。")) }
                }
        }
    }

    // ---- export ----

    fun export() {
        _state.update { it.copy(export = ExportStatus.InProgress) }
        viewModelScope.launch {
            repository.exportData()
                .onSuccess { dto ->
                    val ready = ExportStatus.Ready(
                        json = SettingsExport.toJson(dto),
                        text = SettingsExport.toPlainText(dto),
                    )
                    _state.update { it.copy(export = ready) }
                }
                .onFailure { e ->
                    _state.update { it.copy(export = ExportStatus.Error(e.toJaMessage("エクスポートできませんでした。"))) }
                }
        }
    }

    /** Dismiss the export result (after the share sheet closes). */
    fun clearExport() {
        _state.update { it.copy(export = ExportStatus.Idle) }
    }

    // ---- deletion ----

    fun requestDelete() {
        _state.update { it.copy(deletion = DeletionStatus.Confirming) }
    }

    fun cancelDelete() {
        _state.update { it.copy(deletion = DeletionStatus.Idle) }
    }

    fun confirmDelete() {
        _state.update { it.copy(deletion = DeletionStatus.InProgress) }
        viewModelScope.launch {
            repository.deleteAccount()
                .onSuccess { _state.update { it.copy(deletion = DeletionStatus.Deleted) } }
                .onFailure { e ->
                    _state.update { it.copy(deletion = DeletionStatus.Error(e.toJaMessage("削除できませんでした。"))) }
                }
        }
    }
}
