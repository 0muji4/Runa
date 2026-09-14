package com.runa.shared.core.state

/**
 * Coarse phase of a whole background sync run, shown as a banner over content via [UiState.Content.sync].
 * Distinct from the per-row `SyncState` (pending_create/update/delete).
 */
enum class SyncPhase {
    /** Nothing in flight; no banner is shown. */
    Idle,

    Syncing,

    /** The last sync could not reach the server (connectivity). */
    Offline,

    /** The last sync reached the server but failed (non-connectivity error). */
    Error,
}
