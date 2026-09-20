package com.runa.shared.feature.push

import com.russhwolf.settings.MapSettings
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull

class PushTokenStoreTest {

    @Test
    fun startsEmpty() {
        val store = PushTokenStore(MapSettings())
        assertNull(store.token.value)
        assertNull(store.currentToken())
    }

    @Test
    fun setEmitsImmediately() {
        val store = PushTokenStore(MapSettings())
        store.set("fcm-1")
        assertEquals("fcm-1", store.token.value)
        assertEquals("fcm-1", store.currentToken())
    }

    @Test
    fun persistsAndIsRestoredByAFreshStore() {
        val settings = MapSettings()

        PushTokenStore(settings).set("fcm-1")

        // A new store over the SAME settings models a process restart.
        assertEquals("fcm-1", PushTokenStore(settings).token.value)
    }
}
