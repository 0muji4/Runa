package com.runa.shared.feature.settings

import com.runa.shared.network.dto.ExportDto
import com.runa.shared.network.dto.UserDto

/**
 * The account-data boundary: profile read/edit, self-service export and account
 * deletion. Theme selection is a separate concern ([ThemeRepository]) — this one
 * is the network-backed account surface.
 *
 * All methods return [Result] so callers surface success/failure inline. On a
 * successful [deleteAccount] the local session is torn down (tokens cleared,
 * auth state dropped to unauthenticated) and the local database is wiped.
 */
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
