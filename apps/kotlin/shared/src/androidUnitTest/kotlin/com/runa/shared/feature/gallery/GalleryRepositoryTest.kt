package com.runa.shared.feature.gallery

import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotEquals
import kotlin.test.assertNotNull
import kotlin.test.assertNull
import kotlin.test.assertTrue

/**
 * Gallery offline-first engine tests, mirroring the diary suite: they drive a real
 * [DefaultGalleryRepository] over an in-memory DB + fake server/store on the test
 * scheduler, asserting DB state directly (the mapping to [GalleryImage] is trivial).
 * These are the DoD core: upload ordering, offline queue → reconnect flush, delete
 * propagation, and presigned-URL refetch.
 */
class GalleryRepositoryTest {

    @Test
    fun uploadFollowsUrlThenPutThenRegisterAndSyncs() = runTest {
        val h = GalleryHarness(testScheduler, online = true)
        advanceUntilIdle() // initial (empty) sync from the connectivity edge

        h.repo.addImage(byteArrayOf(1, 2, 3), width = 800, height = 600, mimeType = "image/jpeg", theme = GalleryTheme.PINK)
        advanceUntilIdle()

        val row = h.onlyRow()
        assertEquals("synced", row.sync_state)
        assertNotNull(row.server_id)
        assertNotNull(row.view_url)
        assertNull(row.pending_bytes) // bytes dropped after upload
        assertEquals(800L, row.width)
        assertEquals("pink", row.theme)
        assertEquals(1, h.server.liveCount())

        // The three steps happened in the required order: URL → direct PUT → register.
        val steps = h.server.events.filter { it == "upload-url" || it == "put" || it == "register" }
        assertEquals(listOf("upload-url", "put", "register"), steps)
    }

    @Test
    fun offlineUploadQueuesThenFlushesOnReconnect() = runTest {
        val h = GalleryHarness(testScheduler, online = false)
        h.server.offline = true // the store/API is unreachable while "offline"
        advanceUntilIdle()

        h.repo.addImage(byteArrayOf(9, 9), width = 10, height = 20, mimeType = "image/png", theme = GalleryTheme.MONOTONE)
        advanceUntilIdle()

        val pending = h.onlyRow()
        assertEquals("pending_upload", pending.sync_state)
        assertNull(pending.server_id)
        assertTrue(pending.pending_bytes!!.contentEquals(byteArrayOf(9, 9))) // rendered from these
        assertEquals(0, h.server.liveCount())

        // Connectivity returns → the queued upload flushes on the false→true edge.
        h.server.offline = false
        h.monitor.set(true)
        advanceUntilIdle()

        val synced = h.row(pending.client_id)!!
        assertEquals("synced", synced.sync_state)
        assertNotNull(synced.server_id)
        assertNull(synced.pending_bytes)
        assertEquals(1, h.server.liveCount())
    }

    @Test
    fun deletePropagatesToOtherDevice() = runTest {
        val server = FakeGalleryServer()
        val a = GalleryHarness(testScheduler, server = server, online = true)
        val b = GalleryHarness(testScheduler, server = server, online = true)
        advanceUntilIdle()

        a.repo.addImage(byteArrayOf(1), width = 5, height = 5, mimeType = "image/jpeg", theme = GalleryTheme.PINK)
        advanceUntilIdle()

        // B pulls and sees the image authored on A.
        b.repo.refresh()
        advanceUntilIdle()
        assertEquals(1, b.visible().size)

        // A deletes it.
        a.repo.deleteImage(a.onlyRow().client_id)
        advanceUntilIdle()
        assertEquals(0, a.visible().size)
        assertEquals(0, server.liveCount())

        // B refreshes → reconciles the remote deletion out of its local cache.
        b.repo.refresh()
        advanceUntilIdle()
        assertEquals(0, b.visible().size)
    }

    @Test
    fun refreshRefetchesSignedViewUrl() = runTest {
        val h = GalleryHarness(testScheduler, online = true)
        advanceUntilIdle()

        h.repo.addImage(byteArrayOf(7), width = 100, height = 100, mimeType = "image/webp", theme = GalleryTheme.PINK)
        advanceUntilIdle()
        val url1 = h.onlyRow().view_url
        assertNotNull(url1)

        // A later refresh re-lists and applies freshly-signed URLs (the old one has
        // a finite lifetime); the cached view_url rotates.
        h.repo.refresh()
        advanceUntilIdle()
        val url2 = h.onlyRow().view_url
        assertNotNull(url2)
        assertNotEquals(url1, url2)
    }
}
