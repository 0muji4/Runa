package com.runa.shared.feature.lock

import android.content.Context
import android.os.Build
import androidx.biometric.BiometricManager
import androidx.biometric.BiometricManager.Authenticators.BIOMETRIC_STRONG
import androidx.biometric.BiometricManager.Authenticators.DEVICE_CREDENTIAL
import androidx.biometric.BiometricPrompt
import androidx.core.content.ContextCompat
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlin.coroutines.resume

/**
 * Android [BiometricAuthenticator] over androidx.biometric BiometricPrompt. Face /
 * fingerprint is primary; the device passcode is the fallback (via
 * `DEVICE_CREDENTIAL` on API 30+, so [BiometricResult.Success] covers both). On
 * older APIs the STRONG+credential combination is unsupported, so the prompt shows
 * biometric-only with a cancel button (degraded fallback, documented).
 *
 * The prompt needs a [androidx.fragment.app.FragmentActivity]; it is pulled from
 * [CurrentActivityHolder], which the single Activity keeps current via its
 * lifecycle. If no Activity is resumed the attempt reports [BiometricResult.Unavailable].
 */
class AndroidBiometricAuthenticator(
    private val context: Context,
) : BiometricAuthenticator {

    override fun availability(): BiometricAvailability {
        val status = BiometricManager.from(context).canAuthenticate(allowedAuthenticators())
        return if (status == BiometricManager.BIOMETRIC_SUCCESS) {
            BiometricAvailability.AVAILABLE
        } else {
            BiometricAvailability.UNAVAILABLE
        }
    }

    override suspend fun authenticate(): BiometricResult {
        val activity = CurrentActivityHolder.current() ?: return BiometricResult.Unavailable
        val promptInfo = buildPromptInfo()

        return suspendCancellableCoroutine { cont ->
            val executor = ContextCompat.getMainExecutor(activity)
            val callback = object : BiometricPrompt.AuthenticationCallback() {
                override fun onAuthenticationSucceeded(result: BiometricPrompt.AuthenticationResult) {
                    if (cont.isActive) cont.resume(BiometricResult.Success)
                }

                override fun onAuthenticationError(errorCode: Int, errString: CharSequence) {
                    // Cancel / lockout / hardware error: treat as a failed attempt so
                    // the gate stays locked and offers a retry. availability() already
                    // screens out the "no credential at all" case.
                    if (cont.isActive) cont.resume(BiometricResult.Failed)
                }

                // onAuthenticationFailed (a single non-matching read) keeps the prompt
                // open; the system lets the user try again, so nothing is resumed here.
            }

            // BiometricPrompt must be built and started on the main thread.
            executor.execute {
                runCatching { BiometricPrompt(activity, executor, callback).authenticate(promptInfo) }
                    .onFailure { if (cont.isActive) cont.resume(BiometricResult.Failed) }
            }
        }
    }

    private fun buildPromptInfo(): BiometricPrompt.PromptInfo {
        val builder = BiometricPrompt.PromptInfo.Builder()
            .setTitle(PROMPT_TITLE)
            .setSubtitle(PROMPT_SUBTITLE)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            builder.setAllowedAuthenticators(BIOMETRIC_STRONG or DEVICE_CREDENTIAL)
        } else {
            // Pre-30 can't combine STRONG with DEVICE_CREDENTIAL; require a negative
            // button (mandatory when device credential isn't an allowed authenticator).
            builder.setAllowedAuthenticators(BIOMETRIC_STRONG)
            builder.setNegativeButtonText(PROMPT_CANCEL)
        }
        return builder.build()
    }

    private fun allowedAuthenticators(): Int =
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            BIOMETRIC_STRONG or DEVICE_CREDENTIAL
        } else {
            BIOMETRIC_STRONG
        }

    private companion object {
        const val PROMPT_TITLE = "ロックを解除"
        const val PROMPT_SUBTITLE = "あなたの夜を、あなただけに。"
        const val PROMPT_CANCEL = "キャンセル"
    }
}
