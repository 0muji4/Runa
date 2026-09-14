package com.runa.shared.feature.diary

import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotNull
import kotlin.test.assertNull
import kotlin.test.assertTrue

class DiaryRepositoryTest {

    @Test
    fun offlineCreateRendersLocallyThenSyncsOnReconnect() = runTest {
        val h = DiaryHarness(testScheduler)
        h.server.offline = true

        val entry = h.repo.createEntry("夜の記録", "calm")
        advanceUntilIdle()

        val local = h.rows()
        assertEquals(1, local.size)
        assertEquals("pending_create", local.single().sync_state)
        assertNull(local.single().server_id)
        assertTrue(h.server.isEmpty(), "nothing should have reached the server while offline")

        h.server.offline = false
        h.repo.sync()
        advanceUntilIdle()

        val synced = h.row(entry.clientId)!!
        assertEquals("synced", synced.sync_state)
        assertNotNull(synced.server_id)
        assertEquals(1, h.server.count())
        assertEquals("夜の記録", h.server.bodyOf(entry.clientId))
    }

    @Test
    fun reSendingAPendingCreateDoesNotDuplicate() = runTest {
        val h = DiaryHarness(testScheduler)
        val entry = h.repo.createEntry("一日目", null)
        advanceUntilIdle()
        assertEquals(1, h.server.count())

        // Simulate a lost ack: the server has the row but the client still believes it is pending.
        h.queries.updateContent("一日目（再送）", null, h.clock.now().toString(), "pending_create", entry.clientId)
        h.repo.sync()
        advanceUntilIdle()

        assertEquals(1, h.server.count(), "the repeated client_id must upsert, not duplicate")
        assertEquals("synced", h.row(entry.clientId)!!.sync_state)
        assertEquals("一日目（再送）", h.server.bodyOf(entry.clientId))
    }

    @Test
    fun syncPullsAnEntryAuthoredOnAnotherDevice() = runTest {
        val h = DiaryHarness(testScheduler)
        h.server.seed("cid-remote", "別端末の記録")

        h.repo.sync()
        advanceUntilIdle()

        val local = h.row("cid-remote")
        assertNotNull(local)
        assertEquals("別端末の記録", local!!.body_text)
        assertEquals("synced", local.sync_state)
    }

    @Test
    fun deletionPropagatesBothDirections() = runTest {
        val h = DiaryHarness(testScheduler)

        val entry = h.repo.createEntry("消す", null)
        advanceUntilIdle()
        h.repo.deleteEntry(entry.clientId)
        advanceUntilIdle()
        assertNull(h.row(entry.clientId), "a pushed delete drops the local row")
        assertEquals(0, h.server.count())

        h.server.seed("cid-remote", "遠隔で作成")
        h.repo.sync()
        advanceUntilIdle()
        assertNotNull(h.row("cid-remote"))

        h.server.serverDelete("cid-remote")
        h.repo.sync()
        advanceUntilIdle()
        val tombstoned = h.row("cid-remote")
        assertNotNull(tombstoned)
        assertNotNull(tombstoned!!.deleted_at, "the pulled tombstone marks the row deleted")
    }

    @Test
    fun lastWriteWinsAcrossTwoDevices() = runTest {
        val server = FakeDiaryServer()
        val a = DiaryHarness(testScheduler, server = server)
        val b = DiaryHarness(testScheduler, server = server)

        val entry = a.repo.createEntry("原本", null)
        advanceUntilIdle()
        b.repo.sync()
        advanceUntilIdle()
        assertNotNull(b.row(entry.clientId))

        a.repo.updateEntry(entry.clientId, "Aの推敲", null)
        advanceUntilIdle()
        b.repo.updateEntry(entry.clientId, "Bの推敲", null)
        advanceUntilIdle()

        a.repo.sync()
        advanceUntilIdle()

        assertEquals("Bの推敲", server.bodyOf(entry.clientId))
        assertEquals("Bの推敲", a.row(entry.clientId)!!.body_text)
        assertEquals("Bの推敲", b.row(entry.clientId)!!.body_text)
    }
}
