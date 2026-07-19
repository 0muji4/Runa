package com.runa.shared.feature.gallery

import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.StateFlow

/**
 * The gallery boundary the UI depends on. Local-first, like the diary:
 * [observeImages] streams the on-device DB (so the grid renders instantly and
 * offline), and mutations persist locally then reconcile with the store/server
 * inside [refresh]. Uploads are queued as bytes and pushed on the next sync /
 * connectivity return; the presigned-URL exchange happens transparently inside.
 */
interface GalleryRepository {

    /** Live, newest-first grid from the local DB. Re-emits on every local write,
     *  every applied server change, and every upload-progress tick. */
    fun observeImages(): Flow<List<GalleryImage>>

    /** Queue a picked image: persist its bytes locally (rendered at once as
     *  "uploading") and return; a background upload → register follows. [theme] is
     *  the saved per-image mood. */
    suspend fun addImage(bytes: ByteArray, width: Int, height: Int, mimeType: String, theme: GalleryTheme)

    /** Delete an image: drop it locally if never uploaded, else mark pending-delete
     *  and push (soft-delete + async object removal server-side). */
    suspend fun deleteImage(clientId: String)

    /** Push queued uploads/deletes, then pull the server list — bringing in other
     *  devices' images, refreshing expired view URLs, and reconciling remote
     *  deletions. Overlapping calls coalesce. */
    suspend fun refresh(): Result<Unit>

    /** Coarse status of the last/ongoing sync, for the grid banner. */
    val syncStatus: StateFlow<GallerySyncStatus>

    /** The persisted gallery display-theme toggle (enum name), or null if unset.
     *  This is a client-only view preference, stored in the gallery meta table. */
    suspend fun loadDisplayTheme(): String?

    /** Persist the gallery display-theme toggle (enum name). */
    suspend fun saveDisplayTheme(value: String)
}

/** What a sync is currently doing, surfaced as the grid's subtle banner. */
enum class GallerySyncStatus {
    Idle,
    Syncing,

    /** The last sync could not reach the server/store (connectivity). */
    Offline,

    /** The last sync reached the server but failed (non-connectivity error). */
    Error,
}
