package com.runa.shared.network

import kotlinx.coroutines.flow.StateFlow

/**
 * Observes device connectivity. The diary sync engine subscribes to [isOnline]
 * to (a) drive the offline banner and (b) auto-run a sync the moment
 * connectivity returns (a false → true transition).
 *
 * Actuals: Android wraps `ConnectivityManager.registerDefaultNetworkCallback`,
 * iOS wraps `NWPathMonitor`. Both are bound in `platformModule()`.
 */
interface NetworkMonitor {
    /** Latest connectivity, hot. Starts optimistically true so the first sync is
     *  attempted even before the platform reports a path. */
    val isOnline: StateFlow<Boolean>
}
