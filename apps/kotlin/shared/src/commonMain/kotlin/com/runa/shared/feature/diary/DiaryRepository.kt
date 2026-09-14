package com.runa.shared.feature.diary

import com.runa.shared.core.state.SyncPhase
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.datetime.Instant

/**
 * Diary boundary for the UI. Local-first: mutations persist locally as pending_* and
 * return at once; the network is only ever touched inside [sync].
 */
interface DiaryRepository {

    /** Live, newest-first list from the local DB. */
    fun observeEntries(): Flow<List<DiaryEntry>>

    /** One entry by its local id, or null if unknown/deleted. */
    suspend fun getEntry(clientId: String): DiaryEntry?

    /** Persist a new entry locally and return it; [createdAt] backdates it (null = now). */
    suspend fun createEntry(bodyText: String, mood: String?, createdAt: Instant? = null): DiaryEntry

    suspend fun updateEntry(clientId: String, bodyText: String, mood: String?): Result<Unit>

    /** Soft-delete locally (pending_delete), or hard-drop if never synced. */
    suspend fun deleteEntry(clientId: String): Result<Unit>

    /** Push all pending changes, then pull the server delta. Idempotent; overlapping calls coalesce. */
    suspend fun sync(): Result<Unit>

    val syncStatus: StateFlow<SyncPhase>
}
