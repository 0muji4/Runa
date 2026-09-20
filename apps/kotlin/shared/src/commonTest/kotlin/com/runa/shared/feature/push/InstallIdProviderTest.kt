package com.runa.shared.feature.push

import com.russhwolf.settings.MapSettings
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotEquals
import kotlin.test.assertTrue

class InstallIdProviderTest {

    @Test
    fun mintsAUuidOnceAndReturnsItAgain() {
        val provider = InstallIdProvider(MapSettings())
        val first = provider.get()
        assertTrue(Regex("[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}").matches(first), first)
        assertEquals(first, provider.get())
    }

    @Test
    fun isStableAcrossInstancesOverTheSameSettings() {
        val settings = MapSettings()
        val first = InstallIdProvider(settings).get()
        assertEquals(first, InstallIdProvider(settings).get())
    }

    @Test
    fun differsBetweenInstalls() {
        assertNotEquals(InstallIdProvider(MapSettings()).get(), InstallIdProvider(MapSettings()).get())
    }
}
