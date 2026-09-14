package com.runa.shared.feature.lock

import kotlinx.cinterop.ExperimentalForeignApi
import kotlinx.coroutines.suspendCancellableCoroutine
import platform.LocalAuthentication.LAContext
import platform.LocalAuthentication.LAPolicyDeviceOwnerAuthentication
import kotlin.coroutines.resume

/**
 * iOS [BiometricAuthenticator] over LocalAuthentication. `LAPolicyDeviceOwnerAuthentication` falls back to
 * the passcode by itself, so [BiometricResult.Success] covers both.
 */
@OptIn(ExperimentalForeignApi::class)
class IosBiometricAuthenticator : BiometricAuthenticator {

    override fun availability(): BiometricAvailability =
        if (LAContext().canEvaluatePolicy(LAPolicyDeviceOwnerAuthentication, null)) {
            BiometricAvailability.AVAILABLE
        } else {
            BiometricAvailability.UNAVAILABLE
        }

    override suspend fun authenticate(): BiometricResult = suspendCancellableCoroutine { cont ->
        val context = LAContext()
        if (!context.canEvaluatePolicy(LAPolicyDeviceOwnerAuthentication, null)) {
            if (cont.isActive) cont.resume(BiometricResult.Unavailable)
            return@suspendCancellableCoroutine
        }
        context.evaluatePolicy(
            LAPolicyDeviceOwnerAuthentication,
            localizedReason = PROMPT_REASON,
        ) { success, _ ->
            // Reply may arrive on an arbitrary thread; a cross-thread resume is safe under the new memory model.
            if (cont.isActive) {
                cont.resume(if (success) BiometricResult.Success else BiometricResult.Failed)
            }
        }
    }

    private companion object {
        const val PROMPT_REASON = "ロックを解除して、記録をひらきます。"
    }
}
