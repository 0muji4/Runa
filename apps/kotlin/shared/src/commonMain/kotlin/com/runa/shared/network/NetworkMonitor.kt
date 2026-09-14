package com.runa.shared.network

import kotlinx.coroutines.flow.StateFlow

/** Observes device connectivity; sync engines auto-run on a false → true transition of [isOnline]. */
interface NetworkMonitor {
    /**
     * Latest connectivity, hot. Starts optimistically true so the first sync is attempted before the
     * platform reports.
     */
    val isOnline: StateFlow<Boolean>
}
