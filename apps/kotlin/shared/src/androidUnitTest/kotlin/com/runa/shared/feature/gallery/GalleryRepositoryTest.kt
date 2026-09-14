package com.runa.shared.feature.gallery

import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotEquals
import kotlin.test.assertNotNull
import kotlin.test.assertNull
import kotlin.test.assertTrue

class GalleryRepositoryTest {

    @Test
    fun uploadFollowsUrlThenPutThenRegisterAndSyncs() = runTest {
        val h = GalleryHarness(testScheduler, online = true)
        advanceUntilIdle()

        h.repo.addImage(byteArrayOf(1, 2, 3), width = 800, height = 600, mimeType = "image/jpeg", theme = GalleryTheme.PINK)
        advanceUntilIdle()

        val row = h.onlyRow()
        assertEquals("synced", row.sync_state)
        assertNotNull(row.server_id)
        assertNotNull(row.view_url)
        assertNull(row.pending_bytes)
        assertEquals(800L, row.width)
        assertEquals("pink", row.theme)
        assertEquals(1, h.server.liveCount())

        val steps = h.server.events.filter { it == "upload-url" || it == "put" || it == "register" }
        assertEquals(listOf("upload-url", "put", "register"), steps)
    }

    @Test
    fun offlineUploadQueuesThenFlushesOnReconnect() = runTest {
        val h = GalleryHarness(testScheduler, online = false)
        h.server.offline = true
        advanceUntilIdle()

        h.repo.addImage(byteArrayOf(9, 9), width = 10, height = 20, mimeType = "image/png", theme = GalleryTheme.MONOTONE)
        advanceUntilIdle()

        val pending = h.onlyRow()
        assertEquals("pending_upload", pending.sync_state)
        assertNull(pending.server_id)
        assertTrue(pending.pending_bytes!!.contentEquals(byteArrayOf(9, 9)))
        assertEquals(0, h.server.liveCount())

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

        b.repo.refresh()
        advanceUntilIdle()
        assertEquals(1, b.visible().size)

        a.repo.deleteImage(a.onlyRow().client_id)
        advanceUntilIdle()
        assertEquals(0, a.visible().size)
        assertEquals(0, server.liveCount())

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

        h.repo.refresh()
        advanceUntilIdle()
        val url2 = h.onlyRow().view_url
        assertNotNull(url2)
        assertNotEquals(url1, url2)
    }
}
