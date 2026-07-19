package com.runa.shared.feature.calendar

import com.runa.shared.feature.diary.DiaryEntry
import com.runa.shared.feature.diary.DiaryRepository
import com.runa.shared.feature.diary.SyncState
import com.runa.shared.feature.diary.SyncStatus
import com.runa.shared.network.ApiClient
import com.runa.shared.network.dto.AppleLoginRequest
import com.runa.shared.network.dto.CreateDiaryRequest
import com.runa.shared.network.dto.CreateGalleryRequest
import com.runa.shared.network.dto.DiaryCalendarResponse
import com.runa.shared.network.dto.GalleryUploadURLRequest
import com.runa.shared.network.dto.GoogleLoginRequest
import com.runa.shared.network.dto.LoginRequest
import com.runa.shared.network.dto.LogoutRequest
import com.runa.shared.network.dto.RefreshRequest
import com.runa.shared.network.dto.SignupRequest
import com.runa.shared.network.dto.UpdateDiaryRequest
import com.runa.shared.network.dto.UpdateMeRequest
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.runTest
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant
import kotlinx.datetime.TimeZone
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

/**
 * Calendar composition tests: entry-count reflection and time-zone-boundary
 * grouping, over a fake diary stream — fully in commonMain, no DB or network.
 */
class CalendarRepositoryTest {

    private val utc = TimeZone.UTC
    private val tokyo = TimeZone.of("Asia/Tokyo") // UTC+9
    private val fixedClock = FixedClock(Instant.parse("2026-07-04T00:00:00Z"))

    private fun repo(diary: FakeDiaryRepository, api: FakeCalendarApi = FakeCalendarApi()) =
        DefaultCalendarRepository(diaryRepository = diary, apiClient = api, clock = fixedClock)

    @Test
    fun observeMonthReflectsDiaryCountsPerDay() = runTest {
        val diary = FakeDiaryRepository()
        diary.setEntries(
            listOf(
                entryOn("2026-07-01T09:00:00Z"),
                entryOn("2026-07-01T20:00:00Z"),
                entryOn("2026-07-04T02:00:00Z"),
            ),
        )
        val days = repo(diary).observeMonth(2026, 7, utc).first()

        assertEquals(2, days.first { it.day == 1 }.entryCount)
        assertEquals(1, days.first { it.day == 4 }.entryCount)
        assertEquals(0, days.first { it.day == 2 }.entryCount)
        assertTrue(days.first { it.day == 4 }.isToday, "the 4th is today per the fixed clock")
    }

    @Test
    fun groupingUsesTheLocalDateOfTheZone() = runTest {
        // 2026-07-03T22:00Z is still the 3rd in UTC but already the 4th in Tokyo.
        val diary = FakeDiaryRepository()
        diary.setEntries(listOf(entryOn("2026-07-03T22:00:00Z")))

        val utcDays = repo(diary).observeMonth(2026, 7, utc).first()
        assertEquals(1, utcDays.first { it.day == 3 }.entryCount, "UTC groups it on the 3rd")
        assertEquals(0, utcDays.first { it.day == 4 }.entryCount)

        val tokyoDays = repo(diary).observeMonth(2026, 7, tokyo).first()
        assertEquals(0, tokyoDays.first { it.day == 3 }.entryCount)
        assertEquals(1, tokyoDays.first { it.day == 4 }.entryCount, "Tokyo groups it on the 4th")
    }

    @Test
    fun lateNightEntryStaysOnItsLocalDay() = runTest {
        // 23:30 local Tokyo on the 10th (14:30Z) must stay on the 10th.
        val diary = FakeDiaryRepository()
        diary.setEntries(listOf(entryOn("2026-07-10T14:30:00Z")))
        val days = repo(diary).observeMonth(2026, 7, tokyo).first()
        assertEquals(1, days.first { it.day == 10 }.entryCount)
    }

    @Test
    fun observeEntriesOnReturnsOnlyThatDay() = runTest {
        val diary = FakeDiaryRepository()
        val d10 = entryOn("2026-07-10T09:00:00Z", clientId = "a")
        val d11 = entryOn("2026-07-11T09:00:00Z", clientId = "b")
        diary.setEntries(listOf(d10, d11))

        val onTenth = repo(diary).observeEntriesOn(2026, 7, 10, utc).first()
        assertEquals(listOf("a"), onTenth.map { it.clientId })
    }

    @Test
    fun refreshDelegatesToDiarySyncAndProbesServer() = runTest {
        val diary = FakeDiaryRepository()
        val api = FakeCalendarApi()
        val result = repo(diary, api).refresh(2026, 7, utc)

        assertTrue(result.isSuccess)
        assertEquals(1, diary.syncCalls, "refresh must run the diary sync")
        assertEquals(1, api.calendarCalls, "refresh confirms the server counts on success")
    }

    // ---- helpers ----

    private fun entryOn(instant: String, clientId: String = instant): DiaryEntry {
        val ms = Instant.parse(instant).toEpochMilliseconds()
        return DiaryEntry(
            clientId = clientId, serverId = null, bodyText = "x", mood = null,
            createdAtEpochMs = ms, updatedAtEpochMs = ms, syncState = SyncState.Synced,
        )
    }
}

private class FixedClock(private val instant: Instant) : Clock {
    override fun now(): Instant = instant
}

/** Minimal in-memory [DiaryRepository]: a controllable entry stream + sync counter. */
private class FakeDiaryRepository : DiaryRepository {
    private val entries = MutableStateFlow<List<DiaryEntry>>(emptyList())
    private val _syncStatus = MutableStateFlow(SyncStatus.Idle)
    var syncCalls = 0
        private set

    fun setEntries(list: List<DiaryEntry>) { entries.value = list }

    override fun observeEntries(): Flow<List<DiaryEntry>> = entries
    override val syncStatus: StateFlow<SyncStatus> = _syncStatus
    override suspend fun getEntry(clientId: String): DiaryEntry? = entries.value.firstOrNull { it.clientId == clientId }
    override suspend fun createEntry(bodyText: String, mood: String?, createdAt: Instant?): DiaryEntry =
        error("unused")
    override suspend fun updateEntry(clientId: String, bodyText: String, mood: String?): Result<Unit> = Result.success(Unit)
    override suspend fun deleteEntry(clientId: String): Result<Unit> = Result.success(Unit)
    override suspend fun sync(): Result<Unit> { syncCalls++; return Result.success(Unit) }
}

/** ApiClient stub: only getCalendar is live (records the call); the rest are unused. */
private class FakeCalendarApi : ApiClient {
    var calendarCalls = 0
        private set

    override suspend fun getCalendar(year: Int, month: Int, tz: String?): DiaryCalendarResponse {
        calendarCalls++
        return DiaryCalendarResponse(year = year, month = month, days = emptyList())
    }

    override suspend fun healthz() = error("unused")
    override suspend fun signup(req: SignupRequest) = error("unused")
    override suspend fun login(req: LoginRequest) = error("unused")
    override suspend fun loginApple(req: AppleLoginRequest) = error("unused")
    override suspend fun loginGoogle(req: GoogleLoginRequest) = error("unused")
    override suspend fun refresh(req: RefreshRequest) = error("unused")
    override suspend fun logout(req: LogoutRequest) = error("unused")
    override suspend fun getMe() = error("unused")
    override suspend fun updateMe(req: UpdateMeRequest) = error("unused")
    override suspend fun exportData() = error("unused")
    override suspend fun deleteAccount() = error("unused")
    override suspend fun listDiary(limit: Int?, cursor: String?) = error("unused")
    override suspend fun createDiary(req: CreateDiaryRequest) = error("unused")
    override suspend fun getDiary(id: String) = error("unused")
    override suspend fun updateDiary(id: String, req: UpdateDiaryRequest) = error("unused")
    override suspend fun deleteDiary(id: String) = error("unused")
    override suspend fun syncDiary(since: String?) = error("unused")
    override suspend fun getToday(date: String?) = error("unused")
    override suspend fun getSongs(limit: Int?, cursor: String?) = error("unused")
    override suspend fun markSongPlayed(songId: String, playedAt: String?) = error("unused")
    override suspend fun createGalleryUploadUrl(req: GalleryUploadURLRequest) = error("unused")
    override suspend fun createGallery(req: CreateGalleryRequest) = error("unused")
    override suspend fun listGallery(limit: Int?, cursor: String?) = error("unused")
    override suspend fun getGallery(id: String) = error("unused")
    override suspend fun deleteGallery(id: String) = error("unused")
}
