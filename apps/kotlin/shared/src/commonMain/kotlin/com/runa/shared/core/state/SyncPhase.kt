package com.runa.shared.core.state

/**
 * The coarse phase of a background sync, surfaced to the UI as the quiet status
 * line (`RunaSyncBanner`) over already-rendered content — never a body-hiding
 * state. This is the single shared replacement for the per-feature banner/status
 * enums that used to duplicate it (the diary `SyncStatus`, `GallerySyncStatus`,
 * and the four `*Banner` enums). It rides along on [UiState.Content.sync].
 *
 * Distinct from the per-entry `SyncState` (pending_create/update/delete), which is
 * about a single row, not a whole sync run.
 */
enum class SyncPhase {
    /** Nothing in flight; the last sync (if any) succeeded. No banner is shown. */
    Idle,

    /** A sync is running. */
    Syncing,

    /** The last sync could not reach the server (connectivity). */
    Offline,

    /** The last sync reached the server but failed (non-connectivity error). */
    Error,
}
