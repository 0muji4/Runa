package com.runa.shared.feature.gallery

import com.runa.shared.core.state.SyncPhase
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.StateFlow

/** The gallery boundary, local-first: [observeImages] streams the on-device DB;
 *  mutations persist locally, then reconcile with the store/server inside [refresh]. */
interface GalleryRepository {

    /** Live, newest-first grid from the local DB. */
    fun observeImages(): Flow<List<GalleryImage>>

    /** Queue a picked image: persist its bytes locally and return; the upload follows in the background. */
    suspend fun addImage(bytes: ByteArray, width: Int, height: Int, mimeType: String)

    /** Drop the image locally if never uploaded, else mark pending-delete and push. */
    suspend fun deleteImage(clientId: String)

    /** Push queued uploads/deletes, then pull the server list. Overlapping calls coalesce. */
    suspend fun refresh(): Result<Unit>

    /** Coarse phase of the last/ongoing sync, for the grid banner. */
    val syncStatus: StateFlow<SyncPhase>
}
