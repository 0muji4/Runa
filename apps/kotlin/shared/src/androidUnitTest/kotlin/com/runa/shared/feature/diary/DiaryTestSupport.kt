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
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.TestCoroutineScheduler
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant

/** Fixed test [Clock]; local write timestamps are irrelevant once pushed (the
 *  server assigns authoritative, monotonic updated_at values). */
class MutableClock(var instant: Instant) : Clock {
    override fun now(): Instant = instant
}

/** In-memory [NetworkMonitor] the test flips to simulate connectivity changes. */
class FakeNetworkMonitor(online: Boolean) : NetworkMonitor {
    private val _isOnline = MutableStateFlow(online)
    override val isOnline: StateFlow<Boolean> = _isOnline.asStateFlow()
    fun set(value: Boolean) { _isOnline.value = value }
}

/**
 * A minimal in-memory stand-in for the Go diary backend. It mirrors the server's
 * contract closely enough to test the client sync engine: idempotent upsert by
 * client_id, soft delete, and a since-filtered delta with tombstones. A single
 * instance can be shared by two harnesses to model two devices talking to one
 * server.
 */
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

    private val rows = linkedMapOf<String, Row>() // keyed by client_id (single test user)
    private var idSeq = 0
    private var tickMillis = Instant.parse("2026-01-01T00:00:00Z").toEpochMilliseconds()

    /** When true every endpoint throws, simulating a transport/connectivity failure. */
    var offline: Boolean = false

    fun count(): Int = rows.values.count { it.deletedAt == null }
    fun isEmpty(): Boolean = count() == 0
    fun bodyOf(clientId: String): String? = rows[clientId]?.body

    /** Seed a row as if authored on another device. */
    fun seed(clientId: String, body: String, mood: String? = null) {
        val t = nextTick()
        rows[clientId] = Row("srv-${++idSeq}", clientId, body, mood, t.toString(), t, null)
    }

    /** Soft-delete a row server-side (as another device would), so the next pull
     *  carries the tombstone. */
    fun serverDelete(clientId: String) {
        rows[clientId]?.let { row ->
            val t = nextTick()
            row.deletedAt = t
            row.updatedAt = t
        }
    }

    // ---- the four endpoints the sync engine calls ----

    /** POST /api/v1/diary — idempotent by client_id. */
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

    /** PATCH /api/v1/diary/{id} — 404 once the row is gone (deleted elsewhere). */
    fun update(id: String, req: UpdateDiaryRequest): DiaryEntryDto {
        ensureOnline()
        val row = rows.values.firstOrNull { it.id == id && it.deletedAt == null }
            ?: throw ApiException(404, "not_found", "no diary entry $id")
        row.body = req.bodyText
        row.mood = req.mood
        row.updatedAt = nextTick()
        return row.toDto()
    }

    /** DELETE /api/v1/diary/{id} — soft delete, idempotent. */
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

    /** GET /api/v1/diary/sync?since= — every change after the watermark, tombstones included. */
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

/**
 * [ApiClient] backed directly by [FakeDiaryServer].
 *
 * This deliberately bypasses Ktor's MockEngine. `HttpClientEngineBase.dispatcher`
 * is a real IO dispatcher, so a request issued from a sync coroutine leaves the
 * TestCoroutineScheduler — `advanceUntilIdle()` then returns while the call is
 * still in flight and the assertions race the repository's best-effort
 * `scope.launch { sync() }`. Faking the [ApiClient] seam instead keeps every
 * coroutine on the test scheduler, which is what makes these tests deterministic.
 */
class FakeDiaryApi(private val server: FakeDiaryServer) : ApiClient {

    override suspend fun createDiary(req: CreateDiaryRequest): DiaryEntryDto = server.upsert(req)
    override suspend fun updateDiary(id: String, req: UpdateDiaryRequest): DiaryEntryDto = server.update(id, req)
    override suspend fun deleteDiary(id: String) = server.softDelete(id)
    override suspend fun syncDiary(since: String?): DiarySyncResponse = server.delta(since)

    // The sync engine never reaches these; they only satisfy the interface.
    override suspend fun healthz(): HealthzResponse = unused()
    override suspend fun signup(req: SignupRequest): AuthTokens = unused()
    override suspend fun login(req: LoginRequest): AuthTokens = unused()
    override suspend fun loginApple(req: AppleLoginRequest): AuthTokens = unused()
    override suspend fun loginGoogle(req: GoogleLoginRequest): AuthTokens = unused()
    override suspend fun refresh(req: RefreshRequest): AuthTokens = unused()
    override suspend fun logout(req: LogoutRequest) = unused()
    override suspend fun getMe(): UserDto = unused()
    override suspend fun listDiary(limit: Int?, cursor: String?): DiaryListResponse = unused()
    override suspend fun getDiary(id: String): DiaryEntryDto = unused()
    override suspend fun getCalendar(year: Int, month: Int, tz: String?): DiaryCalendarResponse = unused()
    override suspend fun getToday(date: String?): TodayResponse = unused()
    override suspend fun getSongs(limit: Int?, cursor: String?): SongsArchiveResponse = unused()
    override suspend fun markSongPlayed(songId: String, playedAt: String?) = unused()

    private fun unused(): Nothing = error("endpoint not used by the diary sync engine")
}

/**
 * Wires a real [DefaultDiaryRepository] over a JVM in-memory SQLDelight database
 * and a [FakeDiaryApi]. All coroutines run on the test scheduler so the test
 * drives sync timing deterministically.
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
