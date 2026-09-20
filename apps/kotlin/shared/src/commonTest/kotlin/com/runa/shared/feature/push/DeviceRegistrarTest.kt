package com.runa.shared.feature.push

import com.runa.shared.feature.auth.AuthRepository
import com.runa.shared.feature.auth.AuthState
import com.runa.shared.feature.notification.DefaultNotificationSettingsRepository
import com.runa.shared.feature.notification.ReminderTime
import com.runa.shared.network.ApiClient
import com.runa.shared.network.NetworkMonitor
import com.runa.shared.network.dto.AppleLoginRequest
import com.runa.shared.network.dto.CreateDiaryRequest
import com.runa.shared.network.dto.CreateGalleryRequest
import com.runa.shared.network.dto.DeviceDto
import com.runa.shared.network.dto.GalleryUploadURLRequest
import com.runa.shared.network.dto.GoogleLoginRequest
import com.runa.shared.network.dto.LoginRequest
import com.runa.shared.network.dto.LogoutRequest
import com.runa.shared.network.dto.RefreshRequest
import com.runa.shared.network.dto.RegisterDeviceRequest
import com.runa.shared.network.dto.SignupRequest
import com.runa.shared.network.dto.UpdateDiaryRequest
import com.runa.shared.network.dto.UpdateMeRequest
import com.runa.shared.network.dto.UserDto
import com.russhwolf.settings.MapSettings
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.test.TestScope
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import kotlinx.datetime.Instant
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

private class FakeAuthRepository : AuthRepository {
    override val authState = MutableStateFlow<AuthState>(AuthState.Restoring)

    override suspend fun signupEmail(email: String, password: String, displayName: String?) = error("unused")
    override suspend fun loginEmail(email: String, password: String) = error("unused")
    override suspend fun loginApple(idToken: String, displayName: String?) = error("unused")
    override suspend fun loginGoogle(idToken: String) = error("unused")
    override suspend fun refresh() = error("unused")
    override suspend fun logout() = error("unused")
    override suspend fun getMe() = error("unused")
    override suspend fun restoreSession() = error("unused")
    override fun clearError() = error("unused")
    override fun endSession() = error("unused")
    override fun updateCachedUser(user: UserDto) = error("unused")
}

private class FakeNetworkMonitor(online: Boolean = true) : NetworkMonitor {
    override val isOnline = MutableStateFlow(online)
}

private class RecordingApi : ApiClient {
    val requests = mutableListOf<RegisterDeviceRequest>()
    var fail = false

    override suspend fun registerDevice(req: RegisterDeviceRequest): DeviceDto {
        requests += req
        if (fail) error("boom")
        return DeviceDto(
            id = "d1",
            installId = req.installId,
            pushToken = req.pushToken,
            platform = req.platform,
            notifyTime = req.notifyTime,
            timeZone = req.timeZone,
            enabled = req.enabled,
            createdAt = "2026-01-01T00:00:00Z",
            updatedAt = "2026-01-01T00:00:00Z",
        )
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
    override suspend fun getCalendar(year: Int, month: Int, tz: String?) = error("unused")
    override suspend fun getToday(date: String?) = error("unused")
    override suspend fun getSongs(until: String, limit: Int?, cursor: String?) = error("unused")
    override suspend fun markSongPlayed(songId: String, playedAt: String?) = error("unused")
    override suspend fun createGalleryUploadUrl(req: GalleryUploadURLRequest) = error("unused")
    override suspend fun createGallery(req: CreateGalleryRequest) = error("unused")
    override suspend fun listGallery(limit: Int?, cursor: String?) = error("unused")
    override suspend fun getGallery(id: String) = error("unused")
    override suspend fun deleteGallery(id: String) = error("unused")
}

private fun user(id: String) = UserDto(id = id, displayName = "Runa", authProvider = "email")

private class Fixture(scope: CoroutineScope, online: Boolean = true) {
    val auth = FakeAuthRepository()
    val tokens = PushTokenStore(MapSettings())
    val notifications = DefaultNotificationSettingsRepository(MapSettings())
    val installIds = InstallIdProvider(MapSettings())
    val network = FakeNetworkMonitor(online)
    val api = RecordingApi()
    val settings = MapSettings()
    var now = Instant.parse("2026-09-20T12:00:00Z")
    var zone = "Asia/Tokyo"

    val registrar = DeviceRegistrar(
        authRepository = auth,
        pushTokenStore = tokens,
        notificationSettings = notifications,
        installIdProvider = installIds,
        networkMonitor = network,
        apiClient = api,
        settings = settings,
        platform = "android",
        clock = { now },
        timeZoneId = { zone },
        scope = scope,
    ).also { it.start() }

    fun expected(
        token: String = "fcm-1",
        enabled: Boolean = false,
        time: String = "22:00",
        zone: String = "Asia/Tokyo",
    ) = RegisterDeviceRequest(installIds.get(), token, "android", time, zone, enabled)

    val hasFingerprint: Boolean get() = settings.hasKey("notif.push.registered_fingerprint")
}

@OptIn(ExperimentalCoroutinesApi::class)
class DeviceRegistrarTest {

    // Unconfined so the registrar's collector runs eagerly on the test scheduler.
    private fun TestScope.fixture(online: Boolean = true) =
        Fixture(CoroutineScope(UnconfinedTestDispatcher(testScheduler)), online)

    @Test
    fun tokenThenAuthRegistersOnce() = runTest {
        val fx = fixture()
        advanceUntilIdle()

        fx.tokens.set("fcm-1")
        advanceUntilIdle()
        assertTrue(fx.api.requests.isEmpty())

        fx.auth.authState.value = AuthState.Authenticated(user("u1"))
        advanceUntilIdle()

        assertEquals(listOf(fx.expected()), fx.api.requests)
        assertTrue(fx.hasFingerprint)
    }

    @Test
    fun authThenTokenRegistersOnce() = runTest {
        val fx = fixture()

        fx.auth.authState.value = AuthState.Authenticated(user("u1"))
        advanceUntilIdle()
        assertTrue(fx.api.requests.isEmpty())

        fx.tokens.set("fcm-1")
        advanceUntilIdle()

        assertEquals(listOf(fx.expected()), fx.api.requests)
    }

    @Test
    fun unchangedFingerprintDoesNotRegisterAgain() = runTest {
        val fx = fixture()
        fx.tokens.set("fcm-1")
        fx.auth.authState.value = AuthState.Authenticated(user("u1"))
        advanceUntilIdle()

        fx.registrar.refresh()
        fx.network.isOnline.value = false
        fx.network.isOnline.value = true
        advanceUntilIdle()

        assertEquals(1, fx.api.requests.size)
    }

    @Test
    fun settingsChangeRegistersAgain() = runTest {
        val fx = fixture()
        fx.tokens.set("fcm-1")
        fx.auth.authState.value = AuthState.Authenticated(user("u1"))
        advanceUntilIdle()

        fx.notifications.setReminderEnabled(true)
        advanceUntilIdle()
        assertEquals(fx.expected(enabled = true), fx.api.requests.last())

        fx.notifications.setReminderTime(ReminderTime(23, 0))
        advanceUntilIdle()

        assertEquals(3, fx.api.requests.size)
        assertEquals(fx.expected(enabled = true, time = "23:00"), fx.api.requests.last())
    }

    @Test
    fun tokenRotationRegistersAgain() = runTest {
        val fx = fixture()
        fx.tokens.set("fcm-1")
        fx.auth.authState.value = AuthState.Authenticated(user("u1"))
        advanceUntilIdle()

        fx.tokens.set("fcm-2")
        advanceUntilIdle()

        assertEquals(listOf(fx.expected(), fx.expected(token = "fcm-2")), fx.api.requests)
    }

    @Test
    fun offlineDefersUntilOnline() = runTest {
        val fx = fixture(online = false)
        fx.tokens.set("fcm-1")
        fx.auth.authState.value = AuthState.Authenticated(user("u1"))
        advanceUntilIdle()
        assertTrue(fx.api.requests.isEmpty())

        fx.network.isOnline.value = true
        advanceUntilIdle()

        assertEquals(listOf(fx.expected()), fx.api.requests)
    }

    @Test
    fun signOutClearsTheFingerprintAndReLoginRegistersAgain() = runTest {
        val fx = fixture()
        fx.tokens.set("fcm-1")
        fx.auth.authState.value = AuthState.Authenticated(user("u1"))
        advanceUntilIdle()
        assertTrue(fx.hasFingerprint)

        fx.auth.authState.value = AuthState.Unauthenticated
        advanceUntilIdle()
        assertFalse(fx.hasFingerprint)
        assertEquals(1, fx.api.requests.size)

        fx.auth.authState.value = AuthState.Authenticated(user("u1"))
        advanceUntilIdle()

        assertEquals(2, fx.api.requests.size)
    }

    @Test
    fun userSwitchRegistersAgain() = runTest {
        val fx = fixture()
        fx.tokens.set("fcm-1")
        fx.auth.authState.value = AuthState.Authenticated(user("u1"))
        advanceUntilIdle()

        fx.auth.authState.value = AuthState.Authenticated(user("u2"))
        advanceUntilIdle()

        assertEquals(2, fx.api.requests.size)
    }

    @Test
    fun refreshAfterZoneChangeRegistersAgain() = runTest {
        val fx = fixture()
        fx.tokens.set("fcm-1")
        fx.auth.authState.value = AuthState.Authenticated(user("u1"))
        advanceUntilIdle()

        fx.zone = "Europe/London"
        fx.registrar.refresh()
        advanceUntilIdle()

        assertEquals(fx.expected(zone = "Europe/London"), fx.api.requests.last())
        assertEquals(2, fx.api.requests.size)
    }

    @Test
    fun failureIsNotPersistedAndTheNextTriggerRetries() = runTest {
        val fx = fixture()
        fx.api.fail = true
        fx.tokens.set("fcm-1")
        fx.auth.authState.value = AuthState.Authenticated(user("u1"))
        advanceUntilIdle()
        assertEquals(1, fx.api.requests.size)
        assertFalse(fx.hasFingerprint)

        fx.api.fail = false
        fx.registrar.refresh()
        advanceUntilIdle()

        assertEquals(2, fx.api.requests.size)
        assertTrue(fx.hasFingerprint)
    }

    @Test
    fun registrationOlderThanADayIsRenewed() = runTest {
        val fx = fixture()
        fx.tokens.set("fcm-1")
        fx.auth.authState.value = AuthState.Authenticated(user("u1"))
        advanceUntilIdle()

        fx.now = Instant.parse("2026-09-21T11:00:00Z")
        fx.registrar.refresh()
        advanceUntilIdle()
        assertEquals(1, fx.api.requests.size)

        fx.now = Instant.parse("2026-09-21T13:00:00Z")
        fx.registrar.refresh()
        advanceUntilIdle()

        assertEquals(2, fx.api.requests.size)
    }
}
