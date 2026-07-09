package com.runa.shared.feature.diary

/**
 * A diary entry as the UI sees it. This is the domain model, distinct from the
 * wire [com.runa.shared.network.dto.DiaryEntryDto] and the SQLDelight row.
 *
 * [clientId] is the stable local identity (present from the moment of authoring);
 * [serverId] is null until the create has been synced. [syncState] lets the UI
 * show a subtle "not yet synced" affordance if it wants to.
 *
 * Timestamps are epoch-millis Longs (not kotlinx.datetime types) so the UI can
 * format them with each platform's native date API without the shared module
 * having to export a datetime library across the boundary.
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
