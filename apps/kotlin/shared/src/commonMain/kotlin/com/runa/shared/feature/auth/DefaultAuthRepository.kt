package com.runa.shared.feature.auth

import com.runa.shared.network.ApiClient
import com.runa.shared.network.auth.StoredTokens
import com.runa.shared.network.auth.TokenStore
import com.runa.shared.network.dto.AppleLoginRequest
import com.runa.shared.network.dto.AuthTokens
import com.runa.shared.network.dto.GoogleLoginRequest
import com.runa.shared.network.dto.LoginRequest
import com.runa.shared.network.dto.LogoutRequest
import com.runa.shared.network.dto.RefreshRequest
import com.runa.shared.network.dto.SignupRequest
import com.runa.shared.network.dto.UserDto
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

/**
 * Default [AuthRepository]: orchestrates [ApiClient] + [TokenStore] and drives
 * [authState]. It also listens for [TokenStore.sessionExpired] (fired when an
 * automatic refresh fails mid-session) and drops back to unauthenticated.
 */
class DefaultAuthRepository(
    private val apiClient: ApiClient,
    private val tokenStore: TokenStore,
    scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Default),
) : AuthRepository {

    private val _authState = MutableStateFlow<AuthState>(AuthState.Restoring)
    override val authState: StateFlow<AuthState> = _authState.asStateFlow()

    init {
        scope.launch {
            tokenStore.sessionExpired.collect {
                tokenStore.clear()
                _authState.value = AuthState.Unauthenticated
            }
        }
    }

    override suspend fun signupEmail(email: String, password: String, displayName: String?): Result<Unit> =
        authenticate { apiClient.signup(SignupRequest(email, password, displayName)) }

    override suspend fun loginEmail(email: String, password: String): Result<Unit> =
        authenticate { apiClient.login(LoginRequest(email, password)) }

    override suspend fun loginApple(idToken: String, displayName: String?): Result<Unit> =
        authenticate { apiClient.loginApple(AppleLoginRequest(idToken, displayName)) }

    override suspend fun loginGoogle(idToken: String): Result<Unit> =
        authenticate { apiClient.loginGoogle(GoogleLoginRequest(idToken)) }

    override suspend fun refresh(): Result<Unit> {
        val current = tokenStore.load()
        if (current == null) {
            _authState.value = AuthState.Unauthenticated
            return Result.failure(IllegalStateException("no session to refresh"))
        }
        return try {
            val tokens = apiClient.refresh(RefreshRequest(current.refreshToken))
            tokenStore.save(StoredTokens(tokens.accessToken, tokens.refreshToken))
            Result.success(Unit)
        } catch (e: Exception) {
            tokenStore.clear()
            _authState.value = AuthState.Unauthenticated
            Result.failure(e)
        }
    }

    override suspend fun logout(): Result<Unit> {
        val current = tokenStore.load()
        val result = try {
            if (current != null) apiClient.logout(LogoutRequest(current.refreshToken))
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
        // Always end the local session, even if the server call failed.
        tokenStore.clear()
        _authState.value = AuthState.Unauthenticated
        return result
    }

    override suspend fun getMe(): Result<UserDto> = try {
        val user = apiClient.getMe()
        _authState.value = AuthState.Authenticated(user)
        Result.success(user)
    } catch (e: Exception) {
        Result.failure(e)
    }

    override suspend fun restoreSession() {
        if (tokenStore.load() == null) {
            _authState.value = AuthState.Unauthenticated
            return
        }
        try {
            // The HTTP layer refreshes automatically if the access token expired.
            val user = apiClient.getMe()
            _authState.value = AuthState.Authenticated(user)
        } catch (_: Exception) {
            tokenStore.clear()
            _authState.value = AuthState.Unauthenticated
        }
    }

    override fun clearError() {
        if (_authState.value is AuthState.Error) {
            _authState.value = AuthState.Unauthenticated
        }
    }

    /** Runs a sign-in call, persisting tokens and moving to Authenticated on
     *  success or Error on failure. */
    private suspend fun authenticate(call: suspend () -> AuthTokens): Result<Unit> {
        _authState.value = AuthState.Authenticating
        return try {
            val tokens = call()
            tokenStore.save(StoredTokens(tokens.accessToken, tokens.refreshToken))
            val user = tokens.user ?: apiClient.getMe()
            _authState.value = AuthState.Authenticated(user)
            Result.success(Unit)
        } catch (e: Exception) {
            _authState.value = AuthState.Error(e.message ?: "authentication failed")
            Result.failure(e)
        }
    }
}
