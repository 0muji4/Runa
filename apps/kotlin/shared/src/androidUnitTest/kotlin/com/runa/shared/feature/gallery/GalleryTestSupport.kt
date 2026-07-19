package com.runa.shared.feature.gallery

import app.cash.sqldelight.driver.jdbc.sqlite.JdbcSqliteDriver
import com.runa.shared.db.RunaDatabase
import com.runa.shared.feature.diary.FakeNetworkMonitor
import com.runa.shared.feature.diary.MutableClock
import com.runa.shared.network.ApiClient
import com.runa.shared.network.ApiException
import com.runa.shared.network.StorageClient
import com.runa.shared.network.dto.AppleLoginRequest
import com.runa.shared.network.dto.AuthTokens
import com.runa.shared.network.dto.CreateDiaryRequest
import com.runa.shared.network.dto.CreateGalleryRequest
import com.runa.shared.network.dto.DiaryCalendarResponse
import com.runa.shared.network.dto.DiaryEntryDto
import com.runa.shared.network.dto.DiaryListResponse
import com.runa.shared.network.dto.DiarySyncResponse
import com.runa.shared.network.dto.GalleryImageDto
import com.runa.shared.network.dto.GalleryListResponse
import com.runa.shared.network.dto.GalleryUploadURLRequest
import com.runa.shared.network.dto.GalleryUploadURLResponse
import com.runa.shared.network.dto.GoogleLoginRequest
import com.runa.shared.network.dto.HealthzResponse
import com.runa.shared.network.dto.LoginRequest
import com.runa.shared.network.dto.LogoutRequest
import com.runa.shared.network.dto.RefreshRequest
import com.runa.shared.network.dto.SignupRequest
import com.runa.shared.network.dto.SongsArchiveResponse
import com.runa.shared.network.dto.TodayResponse
import com.runa.shared.network.dto.UpdateDiaryRequest
import com.runa.shared.network.dto.UserDto
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.TestCoroutineScheduler
import kotlinx.datetime.Instant

/**
 * A minimal in-memory stand-in for the Go gallery backend + object store. It mirrors
 * the server's contract closely enough to test the client's upload/sync engine:
 * upload-url issuance, "the object must be PUT before it can be registered",
 * idempotent registration by object_key, a full list with FRESH presigned URLs each
 * call (so URL refresh is observable), and soft delete. [events] records the call
 * order across the API and the storage PUT so the three-step dance can be asserted.
 * One instance can be shared by two harnesses to model two devices.
 */
class FakeGalleryServer {
    private data class Img(
        val id: String,
        val objectKey: String,
        val width: Int,
        val height: Int,
        val theme: String,
        val createdAt: String,
        var deleted: Boolean = false,
    )

    private val images = linkedMapOf<String, Img>() // by server id
    private val pendingUrls = mutableMapOf<String, String>() // upload_url -> object_key
    private val uploaded = mutableSetOf<String>() // object_keys that were actually PUT
    private var idSeq = 0
    private var keySeq = 0
    private var urlSeq = 0
    private var tickMs = Instant.parse("2026-02-01T00:00:00Z").toEpochMilliseconds()

    /** Ordered log of API + storage calls (upload-url, put, register, list, get, delete). */
    val events = mutableListOf<String>()

    /** When true every endpoint throws, simulating a transport/connectivity failure. */
    var offline = false

    fun liveCount(): Int = images.values.count { !it.deleted }

    // ---- API ----

    fun uploadUrl(req: GalleryUploadURLRequest): GalleryUploadURLResponse {
        ensureOnline(); events += "upload-url"
        val key = "gallery/test-user/obj-${++keySeq}"
        val url = "https://store.test/put/${++urlSeq}"
        pendingUrls[url] = key
        return GalleryUploadURLResponse(
            objectKey = key,
            uploadUrl = url,
            method = "PUT",
            headers = mapOf("Content-Type" to req.contentType),
            expiresAt = future().toString(),
            maxSize = 10_000_000,
        )
    }

    fun register(req: CreateGalleryRequest): GalleryImageDto {
        ensureOnline(); events += "register"
        if (req.objectKey !in uploaded) {
            throw ApiException(400, "validation_error", "object not uploaded")
        }
        val existing = images.values.firstOrNull { it.objectKey == req.objectKey && !it.deleted }
        val img = existing ?: Img("srv-${++idSeq}", req.objectKey, req.width, req.height, req.theme, nextIso())
            .also { images[it.id] = it }
        return img.toDto()
    }

    fun list(): GalleryListResponse {
        ensureOnline(); events += "list"
        val items = images.values
            .filter { !it.deleted }
            .sortedByDescending { it.createdAt }
            .map { it.toDto() }
        return GalleryListResponse(items = items, nextCursor = null)
    }

    fun get(id: String): GalleryImageDto {
        ensureOnline(); events += "get"
        val img = images[id]?.takeIf { !it.deleted } ?: throw ApiException(404, "not_found", "no image $id")
        return img.toDto()
    }

    fun delete(id: String) {
        ensureOnline(); events += "delete"
        val img = images[id] ?: throw ApiException(404, "not_found", "no image $id")
        img.deleted = true
    }

    // ---- storage side (called by FakeStorageClient) ----

    fun completeUpload(url: String) {
        val key = pendingUrls.remove(url) ?: error("PUT to unknown upload url $url")
        uploaded += key
        events += "put"
    }

    private fun ensureOnline() {
        if (offline) throw RuntimeException("simulated offline")
    }

    // Each read issues a FRESH presigned URL (rotating), so a refresh is observable.
    private fun Img.toDto() = GalleryImageDto(
        id = id,
        url = "https://store.test/get/$id?v=${++urlSeq}",
        urlExpiresAt = future().toString(),
        width = width,
        height = height,
        theme = theme,
        createdAt = createdAt,
    )

    private fun future(): Instant = Instant.fromEpochMilliseconds(tickMs + 3_600_000)

    private fun nextIso(): String {
        tickMs += 1000
        return Instant.fromEpochMilliseconds(tickMs).toString()
    }
}

/** [StorageClient] that records the direct PUT into [FakeGalleryServer]. */
class FakeStorageClient(private val server: FakeGalleryServer) : StorageClient {
    override suspend fun putBytes(url: String, bytes: ByteArray, contentType: String, onProgress: (Float) -> Unit) {
        onProgress(0.5f)
        server.completeUpload(url)
        onProgress(1f)
    }
}

/** [ApiClient] backed by [FakeGalleryServer]; only the gallery methods are live. */
class FakeGalleryApi(private val server: FakeGalleryServer) : ApiClient {
    override suspend fun createGalleryUploadUrl(req: GalleryUploadURLRequest): GalleryUploadURLResponse = server.uploadUrl(req)
    override suspend fun createGallery(req: CreateGalleryRequest): GalleryImageDto = server.register(req)
    override suspend fun listGallery(limit: Int?, cursor: String?): GalleryListResponse = server.list()
    override suspend fun getGallery(id: String): GalleryImageDto = server.get(id)
    override suspend fun deleteGallery(id: String) = server.delete(id)

    // Unused by the gallery engine; present only to satisfy the interface.
    override suspend fun healthz(): HealthzResponse = unused()
    override suspend fun signup(req: SignupRequest): AuthTokens = unused()
    override suspend fun login(req: LoginRequest): AuthTokens = unused()
    override suspend fun loginApple(req: AppleLoginRequest): AuthTokens = unused()
    override suspend fun loginGoogle(req: GoogleLoginRequest): AuthTokens = unused()
    override suspend fun refresh(req: RefreshRequest): AuthTokens = unused()
    override suspend fun logout(req: LogoutRequest) = unused()
    override suspend fun getMe(): UserDto = unused()
    override suspend fun listDiary(limit: Int?, cursor: String?): DiaryListResponse = unused()
    override suspend fun createDiary(req: CreateDiaryRequest): DiaryEntryDto = unused()
    override suspend fun getDiary(id: String): DiaryEntryDto = unused()
    override suspend fun updateDiary(id: String, req: UpdateDiaryRequest): DiaryEntryDto = unused()
    override suspend fun deleteDiary(id: String) = unused()
    override suspend fun syncDiary(since: String?): DiarySyncResponse = unused()
    override suspend fun getCalendar(year: Int, month: Int, tz: String?): DiaryCalendarResponse = unused()
    override suspend fun getToday(date: String?): TodayResponse = unused()
    override suspend fun getSongs(limit: Int?, cursor: String?): SongsArchiveResponse = unused()
    override suspend fun markSongPlayed(songId: String, playedAt: String?) = unused()

    private fun unused(): Nothing = error("endpoint not used by the gallery engine")
}

/**
 * Wires a real [DefaultGalleryRepository] over a JVM in-memory SQLDelight database,
 * a [FakeGalleryApi] and a [FakeStorageClient], all on the test scheduler so sync
 * timing is deterministic.
 */
class GalleryHarness(
    scheduler: TestCoroutineScheduler,
    val server: FakeGalleryServer = FakeGalleryServer(),
    online: Boolean = false,
    val clock: MutableClock = MutableClock(Instant.parse("2026-07-19T00:00:00Z")),
) {
    val monitor = FakeNetworkMonitor(online)
    val database = RunaDatabase(JdbcSqliteDriver(JdbcSqliteDriver.IN_MEMORY).also { RunaDatabase.Schema.create(it) })
    val queries = database.galleryQueries

    private val dispatcher = StandardTestDispatcher(scheduler)

    val repo: GalleryRepository = DefaultGalleryRepository(
        database = database,
        apiClient = FakeGalleryApi(server),
        storageClient = FakeStorageClient(server),
        networkMonitor = monitor,
        scope = CoroutineScope(dispatcher),
        dispatcher = dispatcher,
        clock = clock,
    )

    fun rows() = queries.selectAll().executeAsList()
    fun visible() = queries.selectVisible().executeAsList()
    fun row(clientId: String) = queries.selectByClientId(clientId).executeAsOneOrNull()
    fun onlyRow() = rows().single()
}
