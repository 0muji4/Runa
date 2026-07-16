package com.runa.shared.feature.diary

import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.datetime.Instant

/**
 * The diary boundary the UI depends on. It is local-first: [observeEntries]
 * streams the on-device database, so writes render instantly and reads work
 * offline. Mutations persist locally (as pending_*) and return immediately; the
 * network is only ever touched inside [sync].
 */
interface DiaryRepository {

    /** Live, newest-first list from the local DB. Re-emits on every local write
     *  and every applied server change. */
    fun observeEntries(): Flow<List<DiaryEntry>>

    /** One entry by its local id, or null if unknown/deleted. */
    suspend fun getEntry(clientId: String): DiaryEntry?

    /** Create an entry: persist locally as pending_create (rendered at once) and
     *  return it; a background push follows. [createdAt] backdates the entry (used
     *  by the calendar's "write on this day" flow); null uses the current clock. */
    suspend fun createEntry(bodyText: String, mood: String?, createdAt: Instant? = null): DiaryEntry

    /** Replace an entry's body/mood locally, marking it pending; a push follows. */
    suspend fun updateEntry(clientId: String, bodyText: String, mood: String?): Result<Unit>

    /** Soft-delete locally (pending_delete) — or hard-drop if never synced — then
     *  push. */
    suspend fun deleteEntry(clientId: String): Result<Unit>

    /** Push all pending changes, then pull the server delta since last_synced_at.
     *  Idempotent and safe to call repeatedly; overlapping calls coalesce. */
    suspend fun sync(): Result<Unit>

    /** Coarse status of the last/ongoing sync, for the list banner. */
    val syncStatus: StateFlow<SyncStatus>
}

/** What a sync is currently doing, surfaced as the list's subtle banner. */
enum class SyncStatus {
    /** Nothing in flight; last sync (if any) succeeded. */
    Idle,

    /** A sync is running. */
    Syncing,

    /** The last sync could not reach the server (connectivity). */
    Offline,

    /** The last sync reached the server but failed (non-connectivity error). */
    Error,
}
