package com.runa.shared.feature.auth

import io.ktor.http.HttpHeaders
import io.ktor.http.HttpStatusCode
import kotlinx.coroutines.test.runTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertIs
import kotlin.test.assertNull
import kotlin.test.assertTrue

class AuthRepositoryTest {

    @Test
    fun loginEmailPersistsTokensAndAuthenticates() = runTest {
        val harness = AuthHarness { request ->
            when (request.url.encodedPath) {
                "/api/v1/auth/login" -> jsonOk(authTokensJson("access1", "refresh1"))
                else -> jsonOk(errorJson("internal_error"), HttpStatusCode.InternalServerError)
            }
        }

        val result = harness.repository.loginEmail("a@b.com", "password123")

        assertTrue(result.isSuccess)
        val state = harness.repository.authState.value
        assertIs<AuthState.Authenticated>(state)
        assertEquals("Runa", state.user.displayName)
        assertEquals("access1", harness.secureStore.get(KEY_ACCESS))
        assertEquals("refresh1", harness.secureStore.get(KEY_REFRESH))
    }

    @Test
    fun signupEmailAuthenticates() = runTest {
        val harness = AuthHarness { request ->
            when (request.url.encodedPath) {
                "/api/v1/auth/signup" -> jsonOk(authTokensJson("a", "r"), HttpStatusCode.Created)
                else -> jsonOk(errorJson("internal_error"), HttpStatusCode.InternalServerError)
            }
        }

        val result = harness.repository.signupEmail("a@b.com", "password123", "Runa")

        assertTrue(result.isSuccess)
        assertIs<AuthState.Authenticated>(harness.repository.authState.value)
    }

    @Test
    fun appleAndGoogleSignInAuthenticate() = runTest {
        val harness = AuthHarness { request ->
            when (request.url.encodedPath) {
                "/api/v1/auth/apple", "/api/v1/auth/google" -> jsonOk(authTokensJson("a", "r"))
                else -> jsonOk(errorJson("internal_error"), HttpStatusCode.InternalServerError)
            }
        }

        assertTrue(harness.repository.loginApple("apple-id-token", "Runa").isSuccess)
        assertIs<AuthState.Authenticated>(harness.repository.authState.value)

        assertTrue(harness.repository.loginGoogle("google-id-token").isSuccess)
        assertIs<AuthState.Authenticated>(harness.repository.authState.value)
    }

    @Test
    fun invalidLoginMovesToErrorState() = runTest {
        val harness = AuthHarness { request ->
            when (request.url.encodedPath) {
                "/api/v1/auth/login" -> jsonOk(errorJson("invalid_credentials"), HttpStatusCode.Unauthorized)
                else -> jsonOk(errorJson("internal_error"), HttpStatusCode.InternalServerError)
            }
        }

        val result = harness.repository.loginEmail("a@b.com", "wrong")

        assertTrue(result.isFailure)
        assertIs<AuthState.Error>(harness.repository.authState.value)
    }

    @Test
    fun unauthorizedTriggersRefreshAndRetriesOriginalRequest() = runTest {
        // Start with an expired-ish access token in the store.
        val harness = AuthHarness(
            initialTokens = mapOf(KEY_ACCESS to "old", KEY_REFRESH to "r1"),
        ) { request ->
            when (request.url.encodedPath) {
                "/api/v1/me" -> {
                    val bearer = request.headers[HttpHeaders.Authorization]
                    if (bearer == "Bearer new") {
                        jsonOk(userJson())
                    } else {
                        jsonOk(errorJson("token_expired"), HttpStatusCode.Unauthorized)
                    }
                }
                "/api/v1/auth/refresh" -> jsonOk(authTokensJson("new", "r2", withUser = false))
                else -> jsonOk(errorJson("internal_error"), HttpStatusCode.InternalServerError)
            }
        }

        val result = harness.repository.getMe()

        assertTrue(result.isSuccess, "getMe should succeed after an automatic refresh")
        // The token pair was rotated and persisted.
        assertEquals("new", harness.secureStore.get(KEY_ACCESS))
        assertEquals("r2", harness.secureStore.get(KEY_REFRESH))
        assertIs<AuthState.Authenticated>(harness.repository.authState.value)
    }

    @Test
    fun failedRefreshClearsSessionAndBecomesUnauthenticated() = runTest {
        val harness = AuthHarness(
            initialTokens = mapOf(KEY_ACCESS to "old", KEY_REFRESH to "r1"),
        ) { request ->
            when (request.url.encodedPath) {
                "/api/v1/me" -> jsonOk(errorJson("token_expired"), HttpStatusCode.Unauthorized)
                "/api/v1/auth/refresh" -> jsonOk(errorJson("token_invalid"), HttpStatusCode.Unauthorized)
                else -> jsonOk(errorJson("internal_error"), HttpStatusCode.InternalServerError)
            }
        }

        harness.repository.restoreSession()

        assertEquals(AuthState.Unauthenticated, harness.repository.authState.value)
        assertNull(harness.secureStore.get(KEY_ACCESS))
        assertNull(harness.secureStore.get(KEY_REFRESH))
    }

    @Test
    fun restoreSessionWithNoTokensIsUnauthenticated() = runTest {
        val harness = AuthHarness { jsonOk(userJson()) }

        harness.repository.restoreSession()

        assertEquals(AuthState.Unauthenticated, harness.repository.authState.value)
    }

    @Test
    fun restoreSessionWithValidTokenIsAuthenticated() = runTest {
        val harness = AuthHarness(
            initialTokens = mapOf(KEY_ACCESS to "good", KEY_REFRESH to "r1"),
        ) { request ->
            when (request.url.encodedPath) {
                "/api/v1/me" -> {
                    if (request.headers[HttpHeaders.Authorization] == "Bearer good") jsonOk(userJson())
                    else jsonOk(errorJson("token_invalid"), HttpStatusCode.Unauthorized)
                }
                else -> jsonOk(errorJson("internal_error"), HttpStatusCode.InternalServerError)
            }
        }

        harness.repository.restoreSession()

        val state = harness.repository.authState.value
        assertIs<AuthState.Authenticated>(state)
        assertEquals("a@b.com", state.user.email)
    }

    @Test
    fun logoutClearsTokensAndBecomesUnauthenticated() = runTest {
        val harness = AuthHarness(
            initialTokens = mapOf(KEY_ACCESS to "a", KEY_REFRESH to "r1"),
        ) { request ->
            when (request.url.encodedPath) {
                "/api/v1/auth/logout" -> jsonOk("", HttpStatusCode.NoContent)
                else -> jsonOk(errorJson("internal_error"), HttpStatusCode.InternalServerError)
            }
        }

        val result = harness.repository.logout()

        assertTrue(result.isSuccess)
        assertEquals(AuthState.Unauthenticated, harness.repository.authState.value)
        assertNull(harness.secureStore.get(KEY_ACCESS))
        assertNull(harness.secureStore.get(KEY_REFRESH))
    }
}
