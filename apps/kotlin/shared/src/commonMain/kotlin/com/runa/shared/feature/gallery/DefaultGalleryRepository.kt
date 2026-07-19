package com.runa.shared.feature.gallery

import app.cash.sqldelight.coroutines.asFlow
import app.cash.sqldelight.coroutines.mapToList
import com.runa.shared.db.Gallery_images
import com.runa.shared.db.RunaDatabase
import com.runa.shared.network.ApiClient
import com.runa.shared.network.ApiException
import com.runa.shared.network.NetworkMonitor
import com.runa.shared.network.StorageClient
import com.runa.shared.network.dto.CreateGalleryRequest
import com.runa.shared.network.dto.GalleryImageDto
import com.runa.shared.network.dto.GalleryUploadURLRequest
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.withContext
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant
import kotlin.coroutines.CoroutineContext
import kotlin.uuid.ExperimentalUuidApi
import kotlin.uuid.Uuid

/**
 * Local-first [GalleryRepository]. A picked image is written to the DB immediately
 * as `pending_upload` (with its bytes) so the grid shows it at once; the network is
 * only touched in [sync], which pushes queued uploads/deletes then pulls the
 * server list.
 *
 * Upload is the three-step presigned-URL dance (see apps/go/README.md): ask our API
 * for a presigned PUT URL, PUT the bytes straight to the store via [StorageClient],
 * then register the metadata. Pull is a full list + reconcile (the gallery API has
 * no delta endpoint): images absent from the server were deleted on another device
 * and are dropped locally; every listed image's presigned view URL is refreshed.
 */
class DefaultGalleryRepository(
    database: RunaDatabase,
    private val apiClient: ApiClient,
    private val storageClient: StorageClient,
    private val networkMonitor: NetworkMonitor,
    private val scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Default),
    private val dispatcher: CoroutineContext = Dispatchers.Default,
    private val clock: Clock = Clock.System,
) : GalleryRepository {

    private val queries = database.galleryQueries

    private val _syncStatus = MutableStateFlow(GallerySyncStatus.Idle)
    override val syncStatus: StateFlow<GallerySyncStatus> = _syncStatus.asStateFlow()

    // Live per-image upload progress (0..1), keyed by client_id. Merged into the
    // grid stream so a cell can show its progress without a DB column.
    private val uploadProgress = MutableStateFlow<Map<String, Float>>(emptyMap())

    private val syncMutex = Mutex()

    init {
        // Auto-sync on the false → true connectivity edge (and once at startup if
        // already online, since StateFlow replays its current value on collect).
        scope.launch {
            var wasOnline = false
            networkMonitor.isOnline.collect { online ->
                if (online && !wasOnline) sync()
                wasOnline = online
            }
        }
    }

    override fun observeImages(): Flow<List<GalleryImage>> =
        combine(
            queries.selectVisible().asFlow().mapToList(dispatcher),
            uploadProgress,
        ) { rows, progress -> rows.map { toDomain(it, progress[it.client_id]) } }

    @OptIn(ExperimentalUuidApi::class)
    override suspend fun addImage(bytes: ByteArray, width: Int, height: Int, mimeType: String, theme: GalleryTheme) {
        withContext(dispatcher) {
            val now = clock.now().toString()
            queries.insertPendingUpload(
                Uuid.random().toString(), // client_id
                width.toLong(),
                height.toLong(),
                theme.wire,
                bytes, // pending_bytes
                mimeType, // content_type
                now, // created_at
                now, // updated_at
            )
        }
        scope.launch { sync() } // best-effort push; no-op offline
    }

    override suspend fun deleteImage(clientId: String) {
        withContext(dispatcher) {
            val existing = queries.selectByClientId(clientId).executeAsOneOrNull() ?: return@withContext
            if (existing.server_id == null) {
                // Never reached the server → just drop it; nothing to delete remotely.
                queries.deleteByClientId(clientId)
            } else {
                val now = clock.now().toString()
                queries.markPendingDelete(now, now, clientId)
            }
        }
        scope.launch { sync() }
    }

    override suspend fun refresh(): Result<Unit> = sync()

    override suspend fun loadDisplayTheme(): String? =
        withContext(dispatcher) { queries.getMeta(KEY_DISPLAY_THEME).executeAsOneOrNull() }

    override suspend fun saveDisplayTheme(value: String) {
        withContext(dispatcher) { queries.setMeta(KEY_DISPLAY_THEME, value) }
    }

    private suspend fun sync(): Result<Unit> {
        if (!syncMutex.tryLock()) return Result.success(Unit)
        return try {
            _syncStatus.value = GallerySyncStatus.Syncing
            push()
            pull()
            _syncStatus.value = GallerySyncStatus.Idle
            Result.success(Unit)
        } catch (e: Exception) {
            _syncStatus.value = if (e is ApiException) GallerySyncStatus.Error else GallerySyncStatus.Offline
            Result.failure(e)
        } finally {
            syncMutex.unlock()
        }
    }

    // ---- push ----

    private suspend fun push() {
        val pending = withContext(dispatcher) { queries.selectPending().executeAsList() }
        for (row in pending) {
            when (row.sync_state) {
                STATE_PENDING_UPLOAD -> pushUpload(row)
                STATE_PENDING_DELETE -> pushDelete(row)
            }
        }
    }

    private suspend fun pushUpload(row: Gallery_images) {
        val bytes = row.pending_bytes ?: run {
            // No bytes to send (shouldn't happen) — drop the orphaned row.
            withContext(dispatcher) { queries.deleteByClientId(row.client_id) }
            return
        }
        val contentType = row.content_type ?: "image/jpeg"

        // 1. Ask our API for a presigned PUT URL + object_key.
        val target = apiClient.createGalleryUploadUrl(GalleryUploadURLRequest(contentType, bytes.size.toLong()))
        setProgress(row.client_id, 0f)
        // 2. PUT the bytes straight to the store (never through our API).
        storageClient.putBytes(target.uploadUrl, bytes, contentType) { p -> setProgress(row.client_id, p) }
        // 3. Register the metadata; the response carries the presigned view URL.
        val dto = apiClient.createGallery(
            CreateGalleryRequest(target.objectKey, row.width.toInt(), row.height.toInt(), row.theme),
        )
        val expiresMs = Instant.parse(dto.urlExpiresAt).toEpochMilliseconds()
        withContext(dispatcher) {
            queries.markUploaded(dto.id, target.objectKey, dto.url, expiresMs, clock.now().toString(), row.client_id)
        }
        clearProgress(row.client_id)
    }

    private suspend fun pushDelete(row: Gallery_images) {
        val serverId = row.server_id
        if (serverId != null) {
            try {
                apiClient.deleteGallery(serverId)
            } catch (e: ApiException) {
                if (e.statusCode != 404) throw e // 404 = already gone; treat as done
            }
        }
        withContext(dispatcher) { queries.deleteByClientId(row.client_id) }
    }

    // ---- pull ----

    private suspend fun pull() {
        val fetched = mutableListOf<GalleryImageDto>()
        var cursor: String? = null
        do {
            val page = apiClient.listGallery(PAGE_LIMIT, cursor)
            fetched += page.items
            cursor = page.nextCursor
        } while (cursor != null && fetched.size < MAX_PULL)
        val complete = cursor == null // only reconcile deletes when we saw the whole list
        val serverIds = fetched.mapTo(mutableSetOf()) { it.id }

        withContext(dispatcher) {
            queries.transaction {
                fetched.forEach(::merge)
                if (complete) reconcileDeletes(serverIds)
            }
        }
    }

    @OptIn(ExperimentalUuidApi::class)
    private fun merge(dto: GalleryImageDto) {
        val existing = queries.selectByServerId(dto.id).executeAsOneOrNull()
        // A row with unpushed local work wins until it is pushed (push runs before
        // pull, so this only guards a rare interleaving).
        if (existing != null && existing.sync_state != STATE_SYNCED) return
        val clientId = existing?.client_id ?: Uuid.random().toString()
        val expiresMs = Instant.parse(dto.urlExpiresAt).toEpochMilliseconds()
        queries.applyServerRow(
            clientId,
            dto.id, // server_id
            existing?.object_key,
            dto.width.toLong(),
            dto.height.toLong(),
            dto.theme,
            dto.url, // view_url
            expiresMs, // view_url_expires_at
            dto.createdAt, // created_at
            dto.createdAt, // updated_at
        )
    }

    private fun reconcileDeletes(serverIds: Set<String>) {
        queries.selectSynced().executeAsList().forEach { row ->
            val sid = row.server_id
            if (sid != null && sid !in serverIds) queries.deleteByClientId(row.client_id)
        }
    }

    // ---- progress ----

    private fun setProgress(clientId: String, value: Float) {
        uploadProgress.value = uploadProgress.value + (clientId to value)
    }

    private fun clearProgress(clientId: String) {
        uploadProgress.value = uploadProgress.value - clientId
    }

    // ---- mapping ----

    private fun toDomain(row: Gallery_images, progress: Float?): GalleryImage = GalleryImage(
        clientId = row.client_id,
        serverId = row.server_id,
        width = row.width.toInt(),
        height = row.height.toInt(),
        theme = GalleryTheme.fromWire(row.theme),
        viewUrl = row.view_url,
        localBytes = row.pending_bytes,
        createdAtEpochMs = Instant.parse(row.created_at).toEpochMilliseconds(),
        uploadState = uploadStateOf(row.sync_state, progress),
        progress = progress ?: 0f,
    )

    private fun uploadStateOf(state: String, progress: Float?): UploadState = when (state) {
        STATE_PENDING_UPLOAD -> if (progress != null && progress > 0f) UploadState.Uploading else UploadState.Queued
        else -> UploadState.Uploaded
    }

    private companion object {
        const val KEY_DISPLAY_THEME = "display_theme"
        const val STATE_SYNCED = "synced"
        const val STATE_PENDING_UPLOAD = "pending_upload"
        const val STATE_PENDING_DELETE = "pending_delete"
        const val PAGE_LIMIT = 100
        const val MAX_PULL = 1000
    }
}
