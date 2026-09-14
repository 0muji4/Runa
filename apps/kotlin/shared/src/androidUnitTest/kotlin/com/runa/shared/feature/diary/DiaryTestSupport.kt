package com.runa.shared.feature.diary

import app.cash.sqldelight.driver.jdbc.sqlite.JdbcSqliteDriver
import com.runa.shared.db.RunaDatabase
import com.runa.shared.network.ApiClient
import com.runa.shared.network.ApiException
import com.runa.shared.network.NetworkMonitor
import com.runa.shared.network.dto.AppleLoginRequest
import com.runa.shared.network.dto.AuthTokens
import com.runa.shared.network.dto.CreateDiaryRequest
import com.runa.shared.network.dto.DiaryCalendarResponse
import com.runa.shared.network.dto.DiaryEntryDto
import com.runa.shared.network.dto.DiaryListResponse
import com.runa.shared.network.dto.DiarySyncResponse
import com.runa.shared.network.dto.CreateGalleryRequest
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
import com.runa.shared.network.dto.ExportDto
import com.runa.shared.network.dto.UpdateDiaryRequest
import com.runa.shared.network.dto.UpdateMeRequest
import com.runa.shared.network.dto.UserDto
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.TestCoroutineScheduler
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant

/** Fixed test [Clock]. */
class MutableClock(var instant: Instant) : Clock {
    override fun now(): Instant = instant
}

/** In-memory [NetworkMonitor] the test flips to simulate connectivity changes. */
class FakeNetworkMonitor(online: Boolean) : NetworkMonitor {
    private val _isOnline = MutableStateFlow(online)
    override val isOnline: StateFlow<Boolean> = _isOnline.asStateFlow()
    fun set(value: Boolean) { _isOnline.value = value }
}

/** In-memory stand-in for the diary backend; share one instance across harnesses to model two devices. */
class FakeDiaryServer {
    private data class Row(
        val id: String,
        val clientId: String,
        var body: String,
        var mood: String?,
        val createdAt: String,
        var updatedAt: Instant,
        var deletedAt: Instant?,
    )

    private val rows = linkedMapOf<String, Row>()
    private var idSeq = 0
    private var tickMillis = Instant.parse("2026-01-01T00:00:00Z").toEpochMilliseconds()

    var offline: Boolean = false

    fun count(): Int = rows.values.count { it.deletedAt == null }
    fun isEmpty(): Boolean = count() == 0
    fun bodyOf(clientId: String): String? = rows[clientId]?.body

    fun seed(clientId: String, body: String, mood: String? = null) {
        val t = nextTick()
        rows[clientId] = Row("srv-${++idSeq}", clientId, body, mood, t.toString(), t, null)
    }

    fun serverDelete(clientId: String) {
        rows[clientId]?.let { row ->
            val t = nextTick()
            row.deletedAt = t
            row.updatedAt = t
        }
    }

    fun upsert(req: CreateDiaryRequest): DiaryEntryDto {
        ensureOnline()
        val existing = rows[req.clientId]
        if (existing != null) {
            existing.body = req.bodyText
            existing.mood = req.mood
            existing.updatedAt = nextTick()
            return existing.toDto()
        }
        val row = Row("srv-${++idSeq}", req.clientId, req.bodyText, req.mood, req.createdAt, nextTick(), null)
        rows[req.clientId] = row
        return row.toDto()
    }

    fun update(id: String, req: UpdateDiaryRequest): DiaryEntryDto {
        ensureOnline()
        val row = rows.values.firstOrNull { it.id == id && it.deletedAt == null }
            ?: throw ApiException(404, "not_found", "no diary entry $id")
        row.body = req.bodyText
        row.mood = req.mood
        row.updatedAt = nextTick()
        return row.toDto()
    }

    fun softDelete(id: String) {
        ensureOnline()
        rows.values.firstOrNull { it.id == id }?.let { row ->
            if (row.deletedAt == null) {
                val t = nextTick()
                row.deletedAt = t
                row.updatedAt = t
            }
        }
    }

    fun delta(since: String?): DiarySyncResponse {
        ensureOnline()
        val sinceInstant = since?.let { Instant.parse(it) }
        val serverTime = nextTick()
        val changed = rows.values
            .filter { sinceInstant == null || it.updatedAt > sinceInstant }
            .sortedBy { it.updatedAt }
            .map { it.toDto() }
        return DiarySyncResponse(entries = changed, serverTime = serverTime.toString())
    }

    private fun ensureOnline() {
        if (offline) throw RuntimeException("simulated offline")
    }

    private fun Row.toDto() = DiaryEntryDto(
        id = id, clientId = clientId, bodyText = body, mood = mood,
        createdAt = createdAt, updatedAt = updatedAt.toString(), deletedAt = deletedAt?.toString(),
    )

    private fun nextTick(): Instant {
        tickMillis += 1000
        return Instant.fromEpochMilliseconds(tickMillis)
    }
}

/** [ApiClient] backed by [FakeDiaryServer]. Not Ktor MockEngine: its engine dispatcher is real IO, so
 *  requests would leave the test scheduler and `advanceUntilIdle()` would race the in-flight sync. */
class FakeDiaryApi(private val server: FakeDiaryServer) : ApiClient {

    override suspend fun createDiary(req: CreateDiaryRequest): DiaryEntryDto = server.upsert(req)
    override suspend fun updateDiary(id: String, req: UpdateDiaryRequest): DiaryEntryDto = server.update(id, req)
    override suspend fun deleteDiary(id: String) = server.softDelete(id)
    override suspend fun syncDiary(since: String?): DiarySyncResponse = server.delta(since)

    override suspend fun healthz(): HealthzResponse = unused()
    override suspend fun signup(req: SignupRequest): AuthTokens = unused()
    override suspend fun login(req: LoginRequest): AuthTokens = unused()
    override suspend fun loginApple(req: AppleLoginRequest): AuthTokens = unused()
    override suspend fun loginGoogle(req: GoogleLoginRequest): AuthTokens = unused()
    override suspend fun refresh(req: RefreshRequest): AuthTokens = unused()
    override suspend fun logout(req: LogoutRequest) = unused()
    override suspend fun getMe(): UserDto = unused()
    override suspend fun updateMe(req: UpdateMeRequest): UserDto = unused()
    override suspend fun exportData(): ExportDto = unused()
    override suspend fun deleteAccount() = unused()
    override suspend fun listDiary(limit: Int?, cursor: String?): DiaryListResponse = unused()
    override suspend fun getDiary(id: String): DiaryEntryDto = unused()
    override suspend fun getCalendar(year: Int, month: Int, tz: String?): DiaryCalendarResponse = unused()
    override suspend fun getToday(date: String?): TodayResponse = unused()
    override suspend fun getSongs(until: String, limit: Int?, cursor: String?): SongsArchiveResponse = unused()
    override suspend fun markSongPlayed(songId: String, playedAt: String?) = unused()
    override suspend fun createGalleryUploadUrl(req: GalleryUploadURLRequest): GalleryUploadURLResponse = unused()
    override suspend fun createGallery(req: CreateGalleryRequest): GalleryImageDto = unused()
    override suspend fun listGallery(limit: Int?, cursor: String?): GalleryListResponse = unused()
    override suspend fun getGallery(id: String): GalleryImageDto = unused()
    override suspend fun deleteGallery(id: String) = unused()

    private fun unused(): Nothing = error("endpoint not used by the diary sync engine")
}

/**
 * Real [DefaultDiaryRepository] over an in-memory SQLDelight database and [FakeDiaryApi], all on the
 * test scheduler.
 */
class DiaryHarness(
    scheduler: TestCoroutineScheduler,
    val server: FakeDiaryServer = FakeDiaryServer(),
    online: Boolean = false,
    val clock: MutableClock = MutableClock(Instant.parse("2026-07-05T00:00:00Z")),
) {
    val monitor = FakeNetworkMonitor(online)
    val database = RunaDatabase(JdbcSqliteDriver(JdbcSqliteDriver.IN_MEMORY).also { RunaDatabase.Schema.create(it) })
    val queries = database.diaryQueries

    private val dispatcher = StandardTestDispatcher(scheduler)

    val repo: DiaryRepository = DefaultDiaryRepository(
        database = database,
        apiClient = FakeDiaryApi(server),
        networkMonitor = monitor,
        scope = CoroutineScope(dispatcher),
        dispatcher = dispatcher,
        clock = clock,
    )

    fun rows() = queries.selectAll().executeAsList()
    fun row(clientId: String) = queries.selectByClientId(clientId).executeAsOneOrNull()
}
