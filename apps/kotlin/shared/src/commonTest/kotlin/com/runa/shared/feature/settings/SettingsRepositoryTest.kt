package com.runa.shared.feature.settings

import com.runa.shared.feature.auth.AuthState
import com.runa.shared.feature.auth.BASE_URL
import com.runa.shared.feature.auth.DefaultAuthRepository
import com.runa.shared.feature.auth.FakeSecureStore
import com.runa.shared.feature.auth.KEY_ACCESS
import com.runa.shared.feature.auth.KEY_REFRESH
import com.runa.shared.feature.auth.MockHandler
import com.runa.shared.feature.auth.jsonOk
import com.runa.shared.feature.auth.userJson
import com.runa.shared.network.HttpClientFactory
import com.runa.shared.network.KtorApiClient
import com.runa.shared.network.auth.TokenRefresher
import com.runa.shared.network.auth.TokenStore
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.http.HttpMethod
import io.ktor.http.HttpStatusCode
import kotlinx.coroutines.test.runTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertIs
import kotlin.test.assertNull
import kotlin.test.assertTrue

/** Records whether the local wipe ran, without touching a real database. */
private class RecordingCleaner : LocalDataCleaner {
    var cleared = false
    override suspend fun clearAll() {
        cleared = true
    }
}

/** Wires the real settings graph (KtorApiClient over MockEngine + DefaultAuthRepository +
 *  DefaultSettingsRepository) with a recording local cleaner. */
private class SettingsFixture(handler: MockHandler) {
    val secureStore = FakeSecureStore(mapOf(KEY_ACCESS to "access", KEY_REFRESH to "refresh"))
    val tokenStore = TokenStore(secureStore)
    private val bareClient = HttpClientFactory.createBase(MockEngine(handler))
    private val refresher = TokenRefresher(bareClient, BASE_URL, tokenStore)
    private val authClient = HttpClientFactory.createAuthenticated(MockEngine(handler), tokenStore, refresher)
    val apiClient = KtorApiClient(authClient, BASE_URL)
    val authRepository = DefaultAuthRepository(apiClient, tokenStore)
    val cleaner = RecordingCleaner()
    val repository = DefaultSettingsRepository(apiClient, authRepository, cleaner)
}

private fun userJsonNamed(name: String): String =
    """{"id":"u1","email":"a@b.com","display_name":"$name","auth_provider":"email","is_premium":false,"created_at":"2026-01-01T00:00:00Z"}"""

class SettingsRepositoryTest {

    @Test
    fun updateDisplayNameReturnsUpdatedUserAndRefreshesAuthState() = runTest {
        val handler: MockHandler = { request ->
            val path = request.url.encodedPath
            when {
                request.method == HttpMethod.Get && path.endsWith("/me") -> jsonOk(userJson())
                request.method == HttpMethod.Patch && path.endsWith("/me") -> jsonOk(userJsonNamed("新しい名前"))
                else -> respond("", HttpStatusCode.NotFound)
            }
        }
        val fixture = SettingsFixture(handler)
        // Establish an authenticated session so the cached user can be refreshed.
        fixture.authRepository.getMe()

        val result = fixture.repository.updateDisplayName("新しい名前")

        assertTrue(result.isSuccess)
        assertEquals("新しい名前", result.getOrThrow().displayName)
        val state = fixture.authRepository.authState.value
        assertIs<AuthState.Authenticated>(state)
        assertEquals("新しい名前", state.user.displayName)
    }

    @Test
    fun deleteAccountWipesLocalDataAndSignsOut() = runTest {
        val handler: MockHandler = { request ->
            val path = request.url.encodedPath
            when {
                request.method == HttpMethod.Get && path.endsWith("/me") -> jsonOk(userJson())
                request.method == HttpMethod.Delete && path.endsWith("/me") ->
                    respond("", HttpStatusCode.NoContent)
                else -> respond("", HttpStatusCode.NotFound)
            }
        }
        val fixture = SettingsFixture(handler)
        fixture.authRepository.getMe()

        val result = fixture.repository.deleteAccount()

        assertTrue(result.isSuccess)
        assertTrue(fixture.cleaner.cleared, "local data must be wiped on deletion")
        assertEquals(AuthState.Unauthenticated, fixture.authRepository.authState.value)
        assertNull(fixture.secureStore.get(KEY_ACCESS))
        assertNull(fixture.secureStore.get(KEY_REFRESH))
    }

    @Test
    fun deleteAccountFailureKeepsSession() = runTest {
        val handler: MockHandler = { request ->
            val path = request.url.encodedPath
            when {
                request.method == HttpMethod.Get && path.endsWith("/me") -> jsonOk(userJson())
                request.method == HttpMethod.Delete && path.endsWith("/me") ->
                    respond("""{"error":{"code":"internal_error","message":"boom"}}""", HttpStatusCode.InternalServerError)
                else -> respond("", HttpStatusCode.NotFound)
            }
        }
        val fixture = SettingsFixture(handler)
        fixture.authRepository.getMe()

        val result = fixture.repository.deleteAccount()

        assertTrue(result.isFailure)
        assertTrue(!fixture.cleaner.cleared, "a failed server delete must not wipe local data")
        assertIs<AuthState.Authenticated>(fixture.authRepository.authState.value)
        assertEquals("access", fixture.secureStore.get(KEY_ACCESS))
    }
}
