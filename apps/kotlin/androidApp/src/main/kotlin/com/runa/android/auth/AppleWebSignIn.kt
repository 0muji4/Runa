package com.runa.android.auth

import android.content.Context
import android.net.Uri
import androidx.browser.customtabs.CustomTabsIntent
import com.runa.android.BuildConfig

/**
 * Sign in with Apple on Android via the web (Apple has no native Android SDK).
 *
 * This opens Apple's authorize page in a Custom Tab. Apple then `form_post`s the
 * resulting `id_token` to the configured **https** redirect (`APPLE_REDIRECT_URI`,
 * an endpoint you control). Completing the round trip — receiving that POST and
 * handing the id_token back into the app (e.g. via an App Link) so it can call
 * the shared `AuthRepository.loginApple` — depends on that redirect endpoint and
 * the real Apple Service ID credentials, and is wired once those exist.
 *
 * See README ("Sign in with Apple on Android") for the full setup.
 */
object AppleWebSignIn {

    fun isConfigured(): Boolean =
        BuildConfig.APPLE_SERVICE_ID.isNotEmpty() && BuildConfig.APPLE_REDIRECT_URI.isNotEmpty()

    /** Launches the Apple authorize page in a Custom Tab. [state]/[nonce] are
     *  opaque values echoed back for CSRF/replay protection. */
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
