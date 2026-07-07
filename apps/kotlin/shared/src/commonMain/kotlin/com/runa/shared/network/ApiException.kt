package com.runa.shared.network

/**
 * Thrown by [ApiClient] when the backend returns a non-2xx response. [code] is
 * the machine-readable error code from the shared error envelope (e.g.
 * "invalid_credentials"), when the body could be parsed.
 */
class ApiException(
    val statusCode: Int,
    val code: String?,
    message: String,
) : Exception(message)
