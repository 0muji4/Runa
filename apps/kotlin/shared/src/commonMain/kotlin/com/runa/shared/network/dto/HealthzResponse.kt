package com.runa.shared.network.dto

import kotlinx.serialization.Serializable

/** Response body of GET /api/v1/healthz: `{"status":"ok"}`. */
@Serializable
data class HealthzResponse(val status: String)
