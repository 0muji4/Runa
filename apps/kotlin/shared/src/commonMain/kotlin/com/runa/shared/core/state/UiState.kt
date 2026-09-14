package com.runa.shared.core.state

/**
 * Page-level state a content screen renders. With cached content, offline/sync rides on
 * [Content.sync] as a banner; only [Failure] (nothing renderable) replaces the body.
 */
sealed interface UiState<out T> {
    data object Loading : UiState<Nothing>

    /** Content to render, plus the background-sync [sync] phase. */
    data class Content<T>(val data: T, val sync: SyncPhase = SyncPhase.Idle) : UiState<T>

    /** The user has authored nothing here yet; not a failure. */
    data object Empty : UiState<Nothing>

    /** Nothing renderable, with a classified [error] selecting the offline or error screen. */
    data class Failure(val error: AppError) : UiState<Nothing>
}
