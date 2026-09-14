package com.runa.android.auth

import android.content.Context
import android.net.Uri
import androidx.browser.customtabs.CustomTabsIntent
import com.runa.android.BuildConfig

/**
 * Sign in with Apple via the web (no native Android SDK). Apple `form_post`s the id_token to
 * the **https** `APPLE_REDIRECT_URI`; the return leg into the app is not wired yet.
 */
object AppleWebSignIn {

    fun isConfigured(): Boolean =
        BuildConfig.APPLE_SERVICE_ID.isNotEmpty() && BuildConfig.APPLE_REDIRECT_URI.isNotEmpty()

    /** [state]/[nonce] are opaque values echoed back for CSRF/replay protection. */
    fun launch(context: Context, state: String, nonce: String) {
        val url = Uri.parse("https://appleid.apple.com/auth/authorize").buildUpon()
            .appendQueryParameter("response_type", "code id_token")
            .appendQueryParameter("response_mode", "form_post")
            .appendQueryParameter("client_id", BuildConfig.APPLE_SERVICE_ID)
            .appendQueryParameter("redirect_uri", BuildConfig.APPLE_REDIRECT_URI)
            .appendQueryParameter("scope", "name email")
            .appendQueryParameter("state", state)
            .appendQueryParameter("nonce", nonce)
            .build()

        CustomTabsIntent.Builder().build().launchUrl(context, url)
    }
}
