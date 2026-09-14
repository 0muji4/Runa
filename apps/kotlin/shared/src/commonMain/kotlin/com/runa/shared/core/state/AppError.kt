package com.runa.shared.core.state

import com.runa.shared.network.ApiException

/** A classified, UI-presentable failure; each case maps to one state screen. */
sealed interface AppError {
    /** Could not reach the server (connectivity / timeout). */
    data object Offline : AppError

    /** The session is no longer valid (a 401 that outlived a token refresh). */
    data class Auth(val message: String? = null) : AppError

    /** The server answered with an error (4xx/5xx); [code] is the envelope's machine-readable code. */
    data class Server(val statusCode: Int, val code: String? = null, val message: String? = null) : AppError

    /** Anything else (e.g. a decode error). */
    data class Unknown(val message: String? = null) : AppError
}

/** Classify a caught [Throwable] into an [AppError]; a non-[ApiException] means the network never answered. */
fun Throwable.toAppError(): AppError = when {
    this is ApiException && statusCode == 401 -> AppError.Auth(message)
    this is ApiException -> AppError.Server(statusCode, code, message)
    else -> AppError.Offline
}
