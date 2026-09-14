package com.runa.shared.network.auth

/** Secure key-value seam backed by the OS secure store (Android: EncryptedSharedPreferences; iOS: Keychain). */
interface SecureKeyValueStore {
    fun get(key: String): String?
    fun set(key: String, value: String)
    fun remove(key: String)
}
