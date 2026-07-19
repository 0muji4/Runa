package com.runa.shared.feature.settings

import com.runa.shared.feature.auth.AuthRepository
import com.runa.shared.network.ApiClient
import com.runa.shared.network.dto.ExportDto
import com.runa.shared.network.dto.UpdateMeRequest
import com.runa.shared.network.dto.UserDto

/**
 * Default [SettingsRepository]. It orchestrates the [ApiClient] and, on account
 * deletion, delegates local teardown to the components that own it: the
 * [AuthRepository] (the single source of truth for auth state + tokens) and the
 * [LocalDataCleaner] (the local database wipe). This repository does not itself
 * hold auth state — it reuses the existing session machinery rather than
 * duplicating it.
 */
class DefaultSettingsRepository(
    private val apiClient: ApiClient,
    private val authRepository: AuthRepository,
    private val localDataCleaner: LocalDataCleaner,
) : SettingsRepository {

    override suspend fun getProfile(): Result<UserDto> = runCatching { apiClient.getMe() }

    override suspend fun updateDisplayName(name: String): Result<UserDto> = runCatching {
        val updated = apiClient.updateMe(UpdateMeRequest(displayName = name))
        // Keep the app-wide user record consistent so any screen reading auth state
        // reflects the new name without a refetch.
        authRepository.updateCachedUser(updated)
        updated
    }

    override suspend fun exportData(): Result<ExportDto> = runCatching { apiClient.exportData() }

    override suspend fun deleteAccount(): Result<Unit> {
        try {
            apiClient.deleteAccount()
        } catch (e: Exception) {
            // Server-side deletion failed; keep the session intact.
            return Result.failure(e)
        }
        // Deletion succeeded: tear down local state unconditionally. A failed wipe
        // must not keep the user signed in to a now-deleted account, so end the
        // session regardless of the cleaner's outcome.
        runCatching { localDataCleaner.clearAll() }
        authRepository.endSession()
        return Result.success(Unit)
    }
}
