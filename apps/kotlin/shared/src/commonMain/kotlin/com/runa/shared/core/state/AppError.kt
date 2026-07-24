package com.runa.shared.core.state

import com.runa.shared.network.ApiException

/**
 * A classified, UI-presentable failure. The state screens map each case to the
 * right surface:
 *  - [Offline] → `RunaOfflineView` (retry),
 *  - [Auth]    → `RunaErrorView` with a re-authenticate CTA (drops to sign-in),
 *  - [Server] / [Unknown] → `RunaErrorView` with a retry CTA.
 *
 * This is the shared error taxonomy the plan calls for (network / auth / server /
 * unknown), replacing the ad-hoc `String` messages screens carried before.
 */
sealed interface AppError {
    /** Could not reach the server (connectivity / timeout). */
    data object Offline : AppError

    /** The session is no longer valid (a 401 that outlived a token refresh) — the
     *  user must re-authenticate. */
    data class Auth(val message: String? = null) : AppError

    /** The server was reached but answered with an error (4xx/5xx). [code] is the
     *  machine-readable code from the shared error envelope, when present. */
    data class Server(val statusCode: Int, val code: String? = null, val message: String? = null) : AppError

    /** Anything else (e.g. a decode error). */
    data class Unknown(val message: String? = null) : AppError
}

/**
 * Classify a caught [Throwable] into an [AppError]. This refines the repositories'
 * old one-line rule (`if (e is ApiException) Error else Offline`): an
 * [ApiException] carrying a 401 is an expired session ([AppError.Auth]); any other
 * [ApiException] is a [AppError.Server] error; and a non-[ApiException] is a
 * transport/connectivity failure ([AppError.Offline]) — the network never
 * answered.
 */
fun Throwable.toAppError(): AppError = when {
    this is ApiException && statusCode == 401 -> AppError.Auth(message)
    this is ApiException -> AppError.Server(statusCode, code, message)
    else -> AppError.Offline
}
