package com.runa.shared.feature.lock

import kotlinx.cinterop.ExperimentalForeignApi
import kotlinx.coroutines.suspendCancellableCoroutine
import platform.LocalAuthentication.LAContext
import platform.LocalAuthentication.LAPolicy
import kotlin.coroutines.resume

/**
 * iOS [BiometricAuthenticator] over LocalAuthentication. Uses
 * `LAPolicy.LAPolicyDeviceOwnerAuthentication`, which presents Face ID / Touch ID first and
 * automatically falls back to the device passcode — so [BiometricResult.Success]
 * covers both and there is no separate fallback to wire. [availability] reflects
 * whether that policy can be evaluated at all (no biometrics AND no passcode →
 * UNAVAILABLE, so the gate won't trap the user).
 */
@OptIn(ExperimentalForeignApi::class)
class IosBiometricAuthenticator : BiometricAuthenticator {

    override fun availability(): BiometricAvailability =
        if (LAContext().canEvaluatePolicy(LAPolicy.LAPolicyDeviceOwnerAuthentication, null)) {
            BiometricAvailability.AVAILABLE
        } else {
            BiometricAvailability.UNAVAILABLE
        }

    override suspend fun authenticate(): BiometricResult = suspendCancellableCoroutine { cont ->
        val context = LAContext()
        if (!context.canEvaluatePolicy(LAPolicy.LAPolicyDeviceOwnerAuthentication, null)) {
            if (cont.isActive) cont.resume(BiometricResult.Unavailable)
            return@suspendCancellableCoroutine
        }
        context.evaluatePolicy(
            LAPolicy.LAPolicyDeviceOwnerAuthentication,
            localizedReason = PROMPT_REASON,
        ) { success, _ ->
            // Reply may arrive on an arbitrary thread; the new memory model makes a
            // cross-thread continuation resume safe.
            if (cont.isActive) {
                cont.resume(if (success) BiometricResult.Success else BiometricResult.Failed)
            }
        }
    }

    private companion object {
        const val PROMPT_REASON = "ロックを解除して、あなたの夜をひらきます。"
    }
}
