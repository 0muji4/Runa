package com.runa.shared.feature.today

import app.cash.sqldelight.driver.jdbc.sqlite.JdbcSqliteDriver
import com.runa.shared.db.RunaDatabase
import com.runa.shared.network.ApiClient
import com.runa.shared.network.HttpClientFactory
import com.runa.shared.network.KtorApiClient
import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpHeaders
import io.ktor.http.HttpStatusCode
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json

/** A fresh in-memory RunaDatabase for a single test (JVM JDBC SQLite driver). */
fun inMemoryDatabase(): RunaDatabase {
    val driver = JdbcSqliteDriver(JdbcSqliteDriver.IN_MEMORY)
    RunaDatabase.Schema.create(driver)
    return RunaDatabase(driver)
}

/** A KtorApiClient wired to the given MockEngine (same JSON config as production). */
fun mockApiClient(engine: MockEngine): ApiClient =
    KtorApiClient(
        httpClient = HttpClient(engine) {
            install(ContentNegotiation) { json(HttpClientFactory.json) }
        },
        baseUrl = "http://test",
    )

/** MockEngine that returns [body] as JSON with [status] for every request. */
fun jsonEngine(status: HttpStatusCode = HttpStatusCode.OK, body: () -> String): MockEngine =
    MockEngine {
        respond(
            content = body(),
            status = status,
            headers = headersOf(HttpHeaders.ContentType, "application/json"),
        )
    }

val todayJson = """
    {"date":"2024-12-15",
     "quote":{"id":"q1","date":"2024-12-15","body_text":"月あかり"},
     "song":{"id":"s1","date":"2024-12-15","title":"夜想曲","artist":"月詠",
             "artwork_url":"https://x/a.jpg","audio_url":"https://x/a.mp3"}}
""".trimIndent()

val songSample = com.runa.shared.network.dto.SongDto(
    id = "s1", date = "2024-12-15", title = "夜想曲", artist = "月詠",
    artworkUrl = "https://x/a.jpg", audioUrl = "https://x/a.mp3",
)
