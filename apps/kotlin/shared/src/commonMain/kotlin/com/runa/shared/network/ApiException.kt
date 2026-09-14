package com.runa.shared.network

/** Thrown by [ApiClient] on a non-2xx response; [code] is the envelope's machine-readable code when parsed. */
class ApiException(
    val statusCode: Int,
    val code: String?,
    message: String,
) : Exception(message)
