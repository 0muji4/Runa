package com.runa.shared.network.auth

/**
 * A small secure key-value seam. The platform actuals back it with the OS secure
 * store (Android: EncryptedSharedPreferences; iOS: Keychain). Common code — and
 * commonTest with an in-memory fake — depend only on this interface, never on the
 * platform class, so token persistence is testable without a device.
 */
interface SecureKeyValueStore {
    fun get(key: String): String?
    fun set(key: String, value: String)
    fun remove(key: String)
}
