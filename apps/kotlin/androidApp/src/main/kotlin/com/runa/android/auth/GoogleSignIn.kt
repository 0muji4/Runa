package com.runa.android.auth

import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.platform.LocalContext
import androidx.credentials.CredentialManager
import androidx.credentials.CustomCredential
import androidx.credentials.GetCredentialRequest
import com.google.android.libraries.identity.googleid.GetGoogleIdOption
import com.google.android.libraries.identity.googleid.GoogleIdTokenCredential
import com.runa.android.BuildConfig
import com.runa.android.R
import kotlinx.coroutines.launch

/**
 * Remembers a launcher for Sign in with Google via the Credential Manager. On
 * success it hands the raw Google **ID token** to [onIdToken]; the shared
 * `AuthRepository.loginGoogle` posts it to the backend, which verifies it.
 *
 * Requires `GOOGLE_SERVER_CLIENT_ID` (the Google OAuth *Web* client ID) to be set
 * — see README. When absent the launcher reports a friendly "unconfigured" error.
 */
@Composable
fun rememberGoogleSignIn(
    onIdToken: (String) -> Unit,
    onError: (String) -> Unit,
): () -> Unit {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val credentialManager = remember { CredentialManager.create(context) }

    return trigger@{
        if (BuildConfig.GOOGLE_SERVER_CLIENT_ID.isEmpty()) {
            onError(context.getString(R.string.signin_provider_unconfigured))
            return@trigger
        }
        scope.launch {
            try {
                val googleIdOption = GetGoogleIdOption.Builder()
                    .setServerClientId(BuildConfig.GOOGLE_SERVER_CLIENT_ID)
                    .setFilterByAuthorizedAccounts(false)
                    .build()
                val request = GetCredentialRequest.Builder()
                    .addCredentialOption(googleIdOption)
                    .build()

                val credential = credentialManager.getCredential(context, request).credential
                if (credential is CustomCredential &&
                    credential.type == GoogleIdTokenCredential.TYPE_GOOGLE_ID_TOKEN_CREDENTIAL
                ) {
                    onIdToken(GoogleIdTokenCredential.createFrom(credential.data).idToken)
                } else {
                    onError(context.getString(R.string.signin_error_generic))
                }
            } catch (e: Exception) {
                onError(e.message ?: context.getString(R.string.signin_error_generic))
            }
        }
    }
}
