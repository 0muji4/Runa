package com.runa.shared.feature.lock

/** Whether the device can perform a biometric-or-passcode authentication. */
enum class BiometricAvailability {
    /** Face ID / fingerprint (or, as fallback, a device passcode) is usable. */
    AVAILABLE,

    UNAVAILABLE,
}

sealed interface BiometricResult {
    /** The user authenticated (biometric or the device-passcode fallback). */
    data object Success : BiometricResult

    /** The user failed or cancelled the prompt. */
    data object Failed : BiometricResult

    data object Unavailable : BiometricResult
}

/** The platform biometric seam, bound in [com.runa.shared.platform.platformModule]. [authenticate] falls back
 *  to the device passcode; on [BiometricResult.Unavailable] the caller must not trap the user behind a lock. */
interface BiometricAuthenticator {
    /** Cheap, synchronous capability check (no prompt shown). */
    fun availability(): BiometricAvailability

    suspend fun authenticate(): BiometricResult
}
