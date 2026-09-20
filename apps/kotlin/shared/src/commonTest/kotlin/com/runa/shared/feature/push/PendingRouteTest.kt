package com.runa.shared.feature.push

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull

class PendingRouteTest {

    @Test
    fun startsEmpty() {
        val route = PendingRoute()
        assertNull(route.route.value)
        assertNull(route.currentRoute())
    }

    @Test
    fun setStaysUntilConsumed() {
        val route = PendingRoute()

        route.set(PendingRouteKind.DIARY_EDITOR_NEW)
        assertEquals(PendingRouteKind.DIARY_EDITOR_NEW, route.route.value)
        assertEquals(PendingRouteKind.DIARY_EDITOR_NEW, route.currentRoute())

        route.consume()
        assertNull(route.route.value)
    }

    @Test
    fun mapsDiaryReminderKindOnly() {
        assertEquals(PendingRouteKind.DIARY_EDITOR_NEW, PendingRouteKind.fromPayloadKind("diary_reminder"))
        assertNull(PendingRouteKind.fromPayloadKind("something_else"))
        assertNull(PendingRouteKind.fromPayloadKind(null))
    }
}
