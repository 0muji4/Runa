package com.runa.shared.network.dto

import kotlinx.serialization.Serializable

/**
 * Response body of GET /api/v1/healthz.
 *
 * Backend contract: HTTP 200 with JSON `{"status":"ok"}`.
 */
@Serializable
data class HealthzResponse(val status: String)
