package com.runa.shared.feature.settings

import com.runa.shared.network.dto.UserDto

/** State for the account-data screen (23): independent facets (profile, name edit, export, deletion). */
data class AccountUiState(
    val profile: UserDto? = null,
    val isLoadingProfile: Boolean = true,
    val loadError: String? = null,
    val isEditingName: Boolean = false,
    val displayNameDraft: String = "",
    val isSavingName: Boolean = false,
    val nameError: String? = null,
    val export: ExportStatus = ExportStatus.Idle,
    val deletion: DeletionStatus = DeletionStatus.Idle,
)

/** Progress of a data export; [Ready] carries both renderings so the UI can offer text or JSON. */
sealed interface ExportStatus {
    data object Idle : ExportStatus
    data object InProgress : ExportStatus
    data class Ready(val json: String, val text: String) : ExportStatus
    data class Error(val message: String) : ExportStatus
}

/** The account-deletion flow. [Deleted] is terminal; the app root observes auth state and returns to sign-in. */
sealed interface DeletionStatus {
    data object Idle : DeletionStatus
    data object Confirming : DeletionStatus
    data object InProgress : DeletionStatus
    data object Deleted : DeletionStatus
    data class Error(val message: String) : DeletionStatus
}
