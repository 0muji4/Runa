package com.runa.shared.platform

import app.cash.sqldelight.db.SqlDriver
import io.ktor.client.engine.HttpClientEngine
import io.ktor.client.engine.darwin.Darwin

/** iOS uses the Darwin (NSURLSession) Ktor engine. */
actual fun httpClientEngine(): HttpClientEngine = Darwin.create()

/**
 * TODO: return a `NativeSqliteDriver(RunaDatabase.Schema, "runa.db")` once local
 * persistence is needed. No feature uses the DB yet, so this stays a stub.
 */
actual fun createSqlDriver(): SqlDriver =
    TODO("iOS SQLDelight driver not wired yet — no feature uses the DB")

/** TODO: back with the iOS Keychain. */
actual class SecureStorage {
    actual fun get(key: String): String? =
        TODO("SecureStorage.get not implemented")

    actual fun set(key: String, value: String): Unit =
        TODO("SecureStorage.set not implemented")

    actual fun remove(key: String): Unit =
        TODO("SecureStorage.remove not implemented")
}

/** TODO: back with APNs device-token retrieval. */
actual class PushTokenProvider {
    actual suspend fun currentToken(): String? =
        TODO("PushTokenProvider.currentToken not implemented")
}

/** TODO: back with StoreKit. Placeholder for now. */
actual class BillingClient

/** TODO: back with LocalAuthentication (Face ID / Touch ID). */
actual class BiometricAuthenticator {
    actual suspend fun authenticate(): Boolean =
        TODO("BiometricAuthenticator.authenticate not implemented")
}
