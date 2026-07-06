package com.runa.shared.network

import com.runa.shared.platform.httpClientEngine
import io.ktor.client.HttpClient
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.Json

/**
 * Builds the shared [HttpClient] from the platform-provided engine
 * (OkHttp on Android, Darwin on iOS) with JSON content negotiation.
 */
object HttpClientFactory {

    private val json = Json {
        ignoreUnknownKeys = true
        isLenient = true
    }

    fun create(): HttpClient =
        HttpClient(httpClientEngine()) {
            install(ContentNegotiation) {
                json(json)
            }
        }
}
