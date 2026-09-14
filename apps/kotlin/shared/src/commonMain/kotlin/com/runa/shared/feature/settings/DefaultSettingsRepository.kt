package com.runa.shared.feature.settings

import com.runa.shared.feature.auth.AuthRepository
import com.runa.shared.network.ApiClient
import com.runa.shared.network.dto.ExportDto
import com.runa.shared.network.dto.UpdateMeRequest
import com.runa.shared.network.dto.UserDto

/** Default [SettingsRepository] over [ApiClient]; on account deletion delegates local
 *  teardown to [AuthRepository] and [LocalDataCleaner]. */
class DefaultSettingsRepository(
    private val apiClient: ApiClient,
    private val authRepository: AuthRepository,
    private val localDataCleaner: LocalDataCleaner,
) : SettingsRepository {

    override suspend fun getProfile(): Result<UserDto> = runCatching { apiClient.getMe() }

    override suspend fun updateDisplayName(name: String): Result<UserDto> = runCatching {
        val updated = apiClient.updateMe(UpdateMeRequest(displayName = name))
        // Keep the app-wide user record consistent without a refetch.
        authRepository.updateCachedUser(updated)
        updated
    }

    override suspend fun exportData(): Result<ExportDto> = runCatching { apiClient.exportData() }

    override suspend fun deleteAccount(): Result<Unit> {
        try {
            apiClient.deleteAccount()
        } catch (e: Exception) {
            return Result.failure(e)
        }
        // A failed wipe must not keep the user signed in to a now-deleted account,
        // so end the session regardless of the cleaner's outcome.
        runCatching { localDataCleaner.clearAll() }
        authRepository.endSession()
        return Result.success(Unit)
    }
}
