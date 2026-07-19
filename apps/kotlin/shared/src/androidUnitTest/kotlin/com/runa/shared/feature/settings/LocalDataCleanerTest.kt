package com.runa.shared.feature.settings

import app.cash.sqldelight.driver.jdbc.sqlite.JdbcSqliteDriver
import com.runa.shared.db.RunaDatabase
import kotlinx.coroutines.test.runTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull
import kotlin.test.assertTrue

/**
 * Exercises the real deleteAll SQL against the actual SQLDelight schema on a JVM
 * in-memory driver (no device needed), proving DefaultLocalDataCleaner empties
 * every user-scoped table + meta.
 */
class LocalDataCleanerTest {

    @Test
    fun clearAllEmptiesEveryUserScopedTable() = runTest {
        val database = RunaDatabase(
            JdbcSqliteDriver(JdbcSqliteDriver.IN_MEMORY).also { RunaDatabase.Schema.create(it) },
        )

        // Seed one row in each table + both meta stores.
        database.diaryQueries.insertEntry(
            "c1", null, "body", null, "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z", null, "synced",
        )
        database.diaryQueries.setMeta("last_synced_at", "2026-01-01T00:00:00Z")
        database.galleryQueries.insertPendingUpload(
            "g1", 100L, 200L, "pink", null, "image/png", "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z",
        )
        database.galleryQueries.setMeta("display_theme", "pink")
        database.todayQueries.upsertQuote("2026-01-01", "q1", "quote")
        database.todayQueries.upsertSong("2026-01-01", "s1", "title", "artist", "art", "audio")
        database.todayQueries.insertPlay("p1", "s1", "title", "artist", "art", 0L)

        // Sanity: the seed landed.
        assertTrue(database.diaryQueries.selectAll().executeAsList().isNotEmpty())
        assertTrue(database.galleryQueries.selectAll().executeAsList().isNotEmpty())

        DefaultLocalDataCleaner(database).clearAll()

        assertEquals(0, database.diaryQueries.selectAll().executeAsList().size)
        assertEquals(0, database.galleryQueries.selectAll().executeAsList().size)
        assertEquals(0, database.todayQueries.selectHistory(100L).executeAsList().size)
        assertNull(database.todayQueries.selectQuote("2026-01-01").executeAsOneOrNull())
        assertNull(database.todayQueries.selectSong("2026-01-01").executeAsOneOrNull())
        assertNull(database.diaryQueries.getMeta("last_synced_at").executeAsOneOrNull())
        assertNull(database.galleryQueries.getMeta("display_theme").executeAsOneOrNull())
    }
}
