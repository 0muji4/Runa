package com.runa.shared.core.state

import com.runa.shared.network.ApiException
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertIs

/**
 * [toAppError] classification — the shared error taxonomy the state screens depend
 * on: a 401 becomes an [AppError.Auth] (re-auth CTA), any other server response is
 * an [AppError.Server], and a non-[ApiException] (the network never answered) is
 * [AppError.Offline].
 */
class AppErrorTest {

    @Test
    fun unauthorizedBecomesAuth() {
        val error = ApiException(statusCode = 401, code = "token_expired", message = "session expired").toAppError()
        val auth = assertIs<AppError.Auth>(error)
        assertEquals("session expired", auth.message)
    }

    @Test
    fun serverErrorBecomesServer() {
        val error = ApiException(statusCode = 503, code = "unavailable", message = "down").toAppError()
        val server = assertIs<AppError.Server>(error)
        assertEquals(503, server.statusCode)
        assertEquals("unavailable", server.code)
    }

    @Test
    fun otherClientErrorBecomesServer() {
        val error = ApiException(statusCode = 404, code = null, message = "not found").toAppError()
        val server = assertIs<AppError.Server>(error)
        assertEquals(404, server.statusCode)
    }

    @Test
    fun nonApiExceptionBecomesOffline() {
        val error = RuntimeException("could not reach host").toAppError()
        assertEquals(AppError.Offline, error)
    }
}
