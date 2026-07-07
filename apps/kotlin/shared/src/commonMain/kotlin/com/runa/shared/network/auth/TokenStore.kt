package com.runa.shared.network.auth

import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.asSharedFlow

/** The persisted access + refresh token pair. */
data class StoredTokens(
    val accessToken: String,
    val refreshToken: String,
)

/**
 * Persists the token pair in the platform secure store and exposes a
 * [sessionExpired] signal that fires when a refresh attempt fails and the local
 * session is cleared. [com.runa.shared.feature.auth.AuthRepository] observes it to
 * drop the whole app back to the unauthenticated state.
 */
class TokenStore(private val store: SecureKeyValueStore) {

    private val _sessionExpired = MutableSharedFlow<Unit>(extraBufferCapacity = 1)
    val sessionExpired: SharedFlow<Unit> = _sessionExpired.asSharedFlow()

    /** Returns the stored pair, or null when either half is missing. */
    fun load(): StoredTokens? {
        val access = store.get(KEY_ACCESS) ?: return null
        val refresh = store.get(KEY_REFRESH) ?: return null
        return StoredTokens(access, refresh)
    }

    fun save(tokens: StoredTokens) {
        store.set(KEY_ACCESS, tokens.accessToken)
        store.set(KEY_REFRESH, tokens.refreshToken)
    }

    fun clear() {
        store.remove(KEY_ACCESS)
        store.remove(KEY_REFRESH)
    }

    /** Clears the tokens and notifies subscribers the session has ended. Used by
     *  the refresher when the refresh token is rejected. */
    fun clearAndNotifyExpired() {
        clear()
        _sessionExpired.tryEmit(Unit)
    }

    private companion object {
        const val KEY_ACCESS = "auth.access_token"
        const val KEY_REFRESH = "auth.refresh_token"
    }
}
