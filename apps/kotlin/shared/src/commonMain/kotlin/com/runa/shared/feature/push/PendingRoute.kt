package com.runa.shared.feature.push

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/** Where a notification tap should land once the lock and auth gates have passed. */
enum class PendingRouteKind {
    DIARY_EDITOR_NEW;

    companion object {
        /** Maps the push payload's `kind` ("diary_reminder") to a route; null for anything else. */
        fun fromPayloadKind(kind: String?): PendingRouteKind? = when (kind) {
            "diary_reminder" -> DIARY_EDITOR_NEW
            else -> null
        }
    }
}

/** Sticky notification-tap destination; the UI navigates and then calls [consume]. */
class PendingRoute {
    private val _route = MutableStateFlow<PendingRouteKind?>(null)

    val route: StateFlow<PendingRouteKind?> = _route.asStateFlow()

    fun set(kind: PendingRouteKind) {
        _route.value = kind
    }

    fun consume() {
        _route.value = null
    }

    fun currentRoute(): PendingRouteKind? = _route.value
}
