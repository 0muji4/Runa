package com.runa.shared.feature.settings

import com.runa.shared.network.dto.ExportDto
import com.runa.shared.network.dto.UserDto

/** The account-data boundary: profile read/edit, export and account deletion.
 *  A successful [deleteAccount] also tears down the local session and wipes the local database. */
interface SettingsRepository {
    /** GET /me — the caller's current profile. */
    suspend fun getProfile(): Result<UserDto>

    /** PATCH /me — update the display name; returns the updated profile. */
    suspend fun updateDisplayName(name: String): Result<UserDto>

    /** GET /me/export — the caller's full data export. */
    suspend fun exportData(): Result<ExportDto>

    /** DELETE /me — permanent account deletion, followed by local teardown. */
    suspend fun deleteAccount(): Result<Unit>
}
