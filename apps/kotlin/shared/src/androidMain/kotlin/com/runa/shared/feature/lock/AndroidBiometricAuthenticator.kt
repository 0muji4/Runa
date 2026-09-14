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
 * [BiometricAuthenticator] over androidx.biometric BiometricPrompt. The prompt needs a FragmentActivity,
 * pulled from [CurrentActivityHolder]; with none resumed the attempt reports [BiometricResult.Unavailable].
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
                    // Cancel / lockout / hardware error: a failed attempt, so the gate stays locked and offers a retry.
                    if (cont.isActive) cont.resume(BiometricResult.Failed)
                }

                // onAuthenticationFailed (a single non-matching read) keeps the prompt open; nothing to resume.
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
            // Pre-30 can't combine STRONG with DEVICE_CREDENTIAL; a negative button is then mandatory.
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
        const val PROMPT_SUBTITLE = "ロックを解除して、記録をひらきます。"
        const val PROMPT_CANCEL = "キャンセル"
    }
}
