package com.runa.shared.feature.diary

import app.cash.sqldelight.coroutines.asFlow
import app.cash.sqldelight.coroutines.mapToList
import com.runa.shared.db.Diary_entries
import com.runa.shared.db.RunaDatabase
import com.runa.shared.network.ApiClient
import com.runa.shared.network.ApiException
import com.runa.shared.network.NetworkMonitor
import com.runa.shared.network.dto.CreateDiaryRequest
import com.runa.shared.network.dto.DiaryEntryDto
import com.runa.shared.network.dto.UpdateDiaryRequest
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.withContext
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant
import kotlin.coroutines.CoroutineContext
import kotlin.uuid.ExperimentalUuidApi
import kotlin.uuid.Uuid

/**
 * Local-first [DiaryRepository]. Writes hit the SQLDelight DB immediately (as
 * pending_*) so the UI updates without a round trip; the network is only touched
 * in [sync], which pushes pending work then pulls the server delta.
 *
 * Sync policy (see apps/go/README.md): client_id makes POST idempotent, and pull
 * conflicts resolve last-write-wins by updated_at. Auto-sync fires when
 * connectivity returns (via [NetworkMonitor]) and after every local mutation.
 */
class DefaultDiaryRepository(
    database: RunaDatabase,
    private val apiClient: ApiClient,
    private val networkMonitor: NetworkMonitor,
    private val scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Default),
    private val dispatcher: CoroutineContext = Dispatchers.Default,
    private val clock: Clock = Clock.System,
) : DiaryRepository {

    private val queries = database.diaryQueries

    private val _syncStatus = MutableStateFlow(SyncStatus.Idle)
    override val syncStatus: StateFlow<SyncStatus> = _syncStatus.asStateFlow()

    // Coalesces overlapping syncs: a caller that finds one already running simply
    // returns, since that run will push everything currently pending.
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

    override fun observeEntries(): Flow<List<DiaryEntry>> =
        queries.selectVisible().asFlow().mapToList(dispatcher).map { rows -> rows.map(::toDomain) }

    override suspend fun getEntry(clientId: String): DiaryEntry? = withContext(dispatcher) {
        queries.selectByClientId(clientId).executeAsOneOrNull()?.let(::toDomain)
    }

    @OptIn(ExperimentalUuidApi::class)
    override suspend fun createEntry(bodyText: String, mood: String?, createdAt: Instant?): DiaryEntry {
        val entry = withContext(dispatcher) {
            // created_at may be backdated (calendar "write on this day"); updated_at
            // is always "now" so the row is newer than any server delta and pushes
            // cleanly under last-write-wins.
            val now = clock.now()
            val created = (createdAt ?: now).toString()
            val updated = now.toString()
            val clientId = Uuid.random().toString()
            queries.insertEntry(clientId, null, bodyText, mood, created, updated, null, STATE_PENDING_CREATE)
            toDomain(queries.selectByClientId(clientId).executeAsOne())
        }
        scope.launch { sync() } // best-effort push; no-op offline
        return entry
    }

    override suspend fun updateEntry(clientId: String, bodyText: String, mood: String?): Result<Unit> {
        val result = withContext(dispatcher) {
            runCatching {
                val existing = queries.selectByClientId(clientId).executeAsOneOrNull()
                    ?: error("no diary entry $clientId")
                // A create still queued locally stays pending_create (there is no
                // server row to PATCH yet); an already-synced entry becomes
                // pending_update.
                val nextState =
                    if (existing.sync_state == STATE_PENDING_CREATE) STATE_PENDING_CREATE else STATE_PENDING_UPDATE
                queries.updateContent(bodyText, mood, clock.now().toString(), nextState, clientId)
            }
        }
        scope.launch { sync() }
        return result
    }

    override suspend fun deleteEntry(clientId: String): Result<Unit> {
        val result = withContext(dispatcher) {
            runCatching {
                val existing = queries.selectByClientId(clientId).executeAsOneOrNull()
                    ?: error("no diary entry $clientId")
                if (existing.sync_state == STATE_PENDING_CREATE && existing.server_id == null) {
                    // Never reached the server → just drop it; nothing to delete remotely.
                    queries.deleteByClientId(clientId)
                } else {
                    val now = clock.now().toString()
                    queries.markPendingDelete(now, now, clientId)
                }
            }
        }
        scope.launch { sync() }
        return result
    }

    override suspend fun sync(): Result<Unit> {
        if (!syncMutex.tryLock()) return Result.success(Unit)
        return try {
            _syncStatus.value = SyncStatus.Syncing
            push()
            pull()
            _syncStatus.value = SyncStatus.Idle
            Result.success(Unit)
        } catch (e: Exception) {
            // An ApiException means the server answered (we are online but it
            // errored); anything else is a transport/connectivity failure.
            _syncStatus.value = if (e is ApiException) SyncStatus.Error else SyncStatus.Offline
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
                STATE_PENDING_CREATE -> pushCreate(row)
                STATE_PENDING_UPDATE -> pushUpdate(row)
                STATE_PENDING_DELETE -> pushDelete(row)
            }
        }
    }

    private suspend fun pushCreate(row: Diary_entries) {
        // Idempotent by client_id: a retried create upserts the same server row.
        val dto = apiClient.createDiary(
            CreateDiaryRequest(
                bodyText = row.body_text,
                mood = row.mood,
                clientId = row.client_id,
                createdAt = row.created_at,
            ),
        )
        withContext(dispatcher) { queries.markSynced(dto.id, dto.updatedAt, row.client_id) }
    }

    private suspend fun pushUpdate(row: Diary_entries) {
        val serverId = row.server_id ?: return
        try {
            val dto = apiClient.updateDiary(serverId, UpdateDiaryRequest(row.body_text, row.mood))
            withContext(dispatcher) { queries.markSynced(dto.id, dto.updatedAt, row.client_id) }
        } catch (e: ApiException) {
            // 404: the entry no longer exists server-side (deleted elsewhere).
            // Server wins under last-write-wins — drop the local copy.
            if (e.statusCode == 404) withContext(dispatcher) { queries.deleteByClientId(row.client_id) } else throw e
        }
    }

    private suspend fun pushDelete(row: Diary_entries) {
        val serverId = row.server_id
        if (serverId != null) {
            try {
                apiClient.deleteDiary(serverId)
            } catch (e: ApiException) {
                if (e.statusCode != 404) throw e // 404 = already gone; treat as done
            }
        }
        withContext(dispatcher) { queries.deleteByClientId(row.client_id) }
    }

    // ---- pull ----

    private suspend fun pull() {
        val since = withContext(dispatcher) { queries.getMeta(KEY_LAST_SYNCED).executeAsOneOrNull() }
        val delta = apiClient.syncDiary(since)
        withContext(dispatcher) {
            queries.transaction {
                delta.entries.forEach(::merge)
                queries.setMeta(KEY_LAST_SYNCED, delta.serverTime)
            }
        }
    }

    private fun merge(dto: DiaryEntryDto) {
        val local = queries.selectByClientId(dto.clientId).executeAsOneOrNull()
        // A tombstone for a row we do not hold must not materialise it: this is
        // either our own pushed delete echoing back, or a delete for an entry
        // authored before this device ever synced.
        if (local == null && dto.deletedAt != null) return
        if (local != null && local.sync_state != STATE_SYNCED) {
            // A local edit hasn't been pushed yet: last-write-wins by updated_at.
            // Keep the local copy (it will push) when it is at least as new.
            if (Instant.parse(local.updated_at) >= Instant.parse(dto.updatedAt)) return
        }
        queries.applyServerRow(
            dto.clientId, dto.id, dto.bodyText, dto.mood, dto.createdAt, dto.updatedAt, dto.deletedAt,
        )
    }

    // ---- mapping ----

    private fun toDomain(row: Diary_entries): DiaryEntry = DiaryEntry(
        clientId = row.client_id,
        serverId = row.server_id,
        bodyText = row.body_text,
        mood = row.mood,
        createdAtEpochMs = Instant.parse(row.created_at).toEpochMilliseconds(),
        updatedAtEpochMs = Instant.parse(row.updated_at).toEpochMilliseconds(),
        syncState = syncStateOf(row.sync_state),
    )

    private fun syncStateOf(raw: String): SyncState = when (raw) {
        STATE_PENDING_CREATE -> SyncState.PendingCreate
        STATE_PENDING_UPDATE -> SyncState.PendingUpdate
        STATE_PENDING_DELETE -> SyncState.PendingDelete
        else -> SyncState.Synced
    }

    private companion object {
        const val KEY_LAST_SYNCED = "last_synced_at"
        const val STATE_SYNCED = "synced"
        const val STATE_PENDING_CREATE = "pending_create"
        const val STATE_PENDING_UPDATE = "pending_update"
        const val STATE_PENDING_DELETE = "pending_delete"
    }
}
