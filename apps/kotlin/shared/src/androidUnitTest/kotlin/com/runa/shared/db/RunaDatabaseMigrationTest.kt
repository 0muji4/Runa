package com.runa.shared.db

import app.cash.sqldelight.db.QueryResult
import app.cash.sqldelight.db.SqlDriver
import app.cash.sqldelight.driver.jdbc.sqlite.JdbcSqliteDriver
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse

/** Upgrades a schema-2 gallery (theme column + toggle table) and checks every row, queued upload included, survives. */
class RunaDatabaseMigrationTest {

    @Test
    fun schemaVersionIsThree() {
        assertEquals(3L, RunaDatabase.Schema.version)
    }

    @Test
    fun upgradeFromTwoKeepsImagesAndDropsTheme() {
        val driver = JdbcSqliteDriver(JdbcSqliteDriver.IN_MEMORY)
        createSchemaTwoGallery(driver)
        driver.execute(
            null,
            """
            INSERT INTO gallery_images(
                client_id, server_id, object_key, width, height, theme, view_url, view_url_expires_at,
                pending_bytes, content_type, created_at, updated_at, deleted_at, sync_state)
            VALUES
                ('synced-1', 'srv-1', 'gallery/u/a', 800, 600, 'pink', 'https://store.test/a', 1,
                 NULL, NULL, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z', NULL, 'synced'),
                ('queued-1', NULL, NULL, 10, 20, 'monotone', NULL, NULL,
                 X'0102', 'image/png', '2026-01-02T00:00:00Z', '2026-01-02T00:00:00Z', NULL, 'pending_upload')
            """.trimIndent(),
            0,
        )
        driver.execute(null, "INSERT INTO gallery_sync_meta(key, value) VALUES ('display_theme', 'MONOTONE')", 0)

        RunaDatabase.Schema.migrate(driver, 2, 3)

        val queries = RunaDatabase(driver).galleryQueries
        val rows = queries.selectVisible().executeAsList()
        assertEquals(listOf("queued-1", "synced-1"), rows.map { it.client_id })
        assertEquals("pending_upload", rows[0].sync_state)
        assertEquals(listOf<Byte>(1, 2), rows[0].pending_bytes?.toList())
        assertEquals("srv-1", rows[1].server_id)
        val upgraded = columnSpecs(driver, "gallery_images")
        assertFalse(upgraded.any { it.startsWith("theme ") })
        assertEquals(columnSpecs(freshDriver(), "gallery_images"), upgraded, "upgrade must land on the fresh-install schema")
        assertEquals(0L, tableCount(driver, "gallery_sync_meta"))
    }

    /** What a fresh install creates, straight from the current `.sq` schema. */
    private fun freshDriver(): SqlDriver =
        JdbcSqliteDriver(JdbcSqliteDriver.IN_MEMORY).also { RunaDatabase.Schema.create(it) }

    /** The gallery tables exactly as schema 2 shipped them (Gallery.sq before this migration). */
    private fun createSchemaTwoGallery(driver: SqlDriver) {
        driver.execute(
            null,
            """
            CREATE TABLE gallery_images (
                client_id TEXT NOT NULL PRIMARY KEY, server_id TEXT, object_key TEXT,
                width INTEGER NOT NULL, height INTEGER NOT NULL, theme TEXT NOT NULL,
                view_url TEXT, view_url_expires_at INTEGER, pending_bytes BLOB, content_type TEXT,
                created_at TEXT NOT NULL, updated_at TEXT NOT NULL, deleted_at TEXT, sync_state TEXT NOT NULL
            )
            """.trimIndent(),
            0,
        )
        driver.execute(null, "CREATE INDEX gallery_images_created_idx ON gallery_images(created_at)", 0)
        driver.execute(null, "CREATE TABLE gallery_sync_meta (key TEXT NOT NULL PRIMARY KEY, value TEXT NOT NULL)", 0)
    }

    /** "name type notnull pk" per column in declared order, so two schemas can be compared. */
    private fun columnSpecs(driver: SqlDriver, table: String): List<String> = driver.executeQuery(
        identifier = null,
        sql = "SELECT name || ' ' || type || ' ' || \"notnull\" || ' ' || pk FROM pragma_table_info(?)",
        mapper = { cursor ->
            val names = mutableListOf<String>()
            while (cursor.next().value) names += cursor.getString(0)!!
            QueryResult.Value(names)
        },
        parameters = 1,
    ) { bindString(0, table) }.value

    private fun tableCount(driver: SqlDriver, name: String): Long = driver.executeQuery(
        identifier = null,
        sql = "SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = ?",
        mapper = { cursor ->
            cursor.next()
            QueryResult.Value(cursor.getLong(0) ?: 0L)
        },
        parameters = 1,
    ) { bindString(0, name) }.value
}
