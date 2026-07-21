package com.runa.shared.feature.lock

/** Whether the device can perform a biometric-or-passcode authentication. */
enum class BiometricAvailability {
    /** Face ID / fingerprint (or, as fallback, a device passcode) is usable. */
    AVAILABLE,

    /** No usable authentication (no hardware and no device passcode set). */
    UNAVAILABLE,
}

/** Outcome of an authentication attempt. */
sealed interface BiometricResult {
    /** The user authenticated (biometric or the device-passcode fallback). */
    data object Success : BiometricResult

    /** The user failed or cancelled the prompt. */
    data object Failed : BiometricResult

    /** Authentication could not be presented (no biometric and no passcode). */
    data object Unavailable : BiometricResult
}

/**
 * The platform biometric seam (Face ID / Touch ID on iOS, BiometricPrompt on
 * Android), bound in [com.runa.shared.platform.platformModule]. Replaces the old
 * `expect class BiometricAuthenticator` stub — an interface is used (like
 * AudioPlayer) because the Android implementation needs the current Activity, which
 * an `expect class` constructor can't carry.
 *
 * The device passcode is the fallback: [authenticate] presents biometric first and
 * falls back to the device credential, so [BiometricResult.Success] covers both.
 * [BiometricResult.Unavailable] means the device has neither — the caller must not
 * trap the user behind a lock that can never be satisfied.
 */
interface BiometricAuthenticator {
    /** Cheap, synchronous capability check (no prompt shown). */
    fun availability(): BiometricAvailability

    /** Present the biometric-or-passcode prompt and await the outcome. */
    suspend fun authenticate(): BiometricResult
}
