package com.runa.shared.feature.settings

import app.cash.sqldelight.driver.jdbc.sqlite.JdbcSqliteDriver
import com.runa.shared.db.RunaDatabase
import kotlinx.coroutines.test.runTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull
import kotlin.test.assertTrue

class LocalDataCleanerTest {

    @Test
    fun clearAllEmptiesEveryUserScopedTable() = runTest {
        val database = RunaDatabase(
            JdbcSqliteDriver(JdbcSqliteDriver.IN_MEMORY).also { RunaDatabase.Schema.create(it) },
        )

        database.diaryQueries.insertEntry(
            "c1", null, "body", null, "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z", null, "synced",
        )
        database.diaryQueries.setMeta("last_synced_at", "2026-01-01T00:00:00Z")
        database.galleryQueries.insertPendingUpload(
            "g1", 100L, 200L, null, "image/png", "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z",
        )
        database.todayQueries.upsertQuote("2026-01-01", "q1", "quote")
        database.todayQueries.upsertSong("2026-01-01", "s1", "title", "artist", "art", "preview", "store")
        database.todayQueries.insertPlay("p1", "s1", "title", "artist", 0L)

        assertTrue(database.diaryQueries.selectAll().executeAsList().isNotEmpty())
        assertTrue(database.galleryQueries.selectAll().executeAsList().isNotEmpty())

        DefaultLocalDataCleaner(database).clearAll()

        assertEquals(0, database.diaryQueries.selectAll().executeAsList().size)
        assertEquals(0, database.galleryQueries.selectAll().executeAsList().size)
        assertEquals(0, database.todayQueries.selectHistory(100L).executeAsList().size)
        assertNull(database.todayQueries.selectQuote("2026-01-01").executeAsOneOrNull())
        assertNull(database.todayQueries.selectSong("2026-01-01").executeAsOneOrNull())
        assertNull(database.diaryQueries.getMeta("last_synced_at").executeAsOneOrNull())
    }
}
