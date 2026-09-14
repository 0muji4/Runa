package com.runa.shared.feature.diary

/**
 * Domain model of a diary entry. [clientId] is the stable local identity;
 * [serverId] is null until the create has been synced.
 */
data class DiaryEntry(
    val clientId: String,
    val serverId: String?,
    val bodyText: String,
    val mood: String?,
    val createdAtEpochMs: Long,
    val updatedAtEpochMs: Long,
    val syncState: SyncState,
)

/** Local-only sync lifecycle of an entry, persisted in the `sync_state` column. */
enum class SyncState {
    Synced,
    PendingCreate,
    PendingUpdate,
    PendingDelete,
}
