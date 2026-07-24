package com.runa.shared.core.state

/**
 * The page-level state a content screen renders, unified across features so both
 * Android and iOS drive the *same* shared state components (`RunaLoadingView` /
 * `RunaEmptyView` / `RunaOfflineView` / `RunaErrorView`) instead of each screen
 * re-implementing loading/empty/offline/error.
 *
 * Local-first offline model (the plan's DoD #2): when there is cached content to
 * show, offline/sync is carried as [Content.sync] — a quiet banner over the body —
 * rather than a body-hiding state. Only [Failure] (nothing renderable) replaces the
 * body, and it is discriminated by its [AppError] into the offline vs. error
 * screen. See the README's "状態画面" section.
 *
 * The interface is covariant (`out T`) so `Loading` / `Empty` / `Failure`
 * (declared over `Nothing`) are assignable wherever a `UiState<T>` is expected;
 * [Content] itself is an invariant data class so its generated `copy`/`equals`
 * stay well-formed.
 */
sealed interface UiState<out T> {
    /** No content yet — the screen shows the quiet full-screen loading indicator. */
    data object Loading : UiState<Nothing>

    /** Content to render, plus the current background-sync [sync] phase (the banner). */
    data class Content<T>(val data: T, val sync: SyncPhase = SyncPhase.Idle) : UiState<T>

    /** The user has authored nothing here yet — an invitation to begin, not a failure. */
    data object Empty : UiState<Nothing>

    /** Nothing renderable, with a classified [error] selecting the offline or error screen. */
    data class Failure(val error: AppError) : UiState<Nothing>
}
