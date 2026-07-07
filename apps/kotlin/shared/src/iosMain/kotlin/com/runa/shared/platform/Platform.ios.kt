package com.runa.shared.platform

import app.cash.sqldelight.db.SqlDriver
import com.russhwolf.settings.ExperimentalSettingsImplementation
import com.russhwolf.settings.KeychainSettings
import com.runa.shared.network.auth.SecureKeyValueStore
import io.ktor.client.engine.HttpClientEngine
import io.ktor.client.engine.darwin.Darwin
import org.koin.core.module.Module
import org.koin.dsl.module

/** iOS uses the Darwin (NSURLSession) Ktor engine. */
actual fun httpClientEngine(): HttpClientEngine = Darwin.create()

/**
 * TODO: return a `NativeSqliteDriver(RunaDatabase.Schema, "runa.db")` once local
 * persistence is needed. No feature uses the DB yet, so this stays a stub.
 */
actual fun createSqlDriver(): SqlDriver =
    TODO("iOS SQLDelight driver not wired yet — no feature uses the DB")

/** Provides the Keychain-backed secure store. */
actual fun platformModule(): Module = module {
    single<SecureKeyValueStore> { KeychainSecureStore() }
}

/**
 * Keychain-backed [SecureKeyValueStore]. Reuses multiplatform-settings'
 * [KeychainSettings] (already a shared dependency) so tokens live in the iOS
 * Keychain without hand-rolled Security-framework interop. The no-arg constructor
 * uses the default keychain service scope, which is sufficient here.
 */
@OptIn(ExperimentalSettingsImplementation::class)
class KeychainSecureStore(
    private val settings: KeychainSettings = KeychainSettings(),
) : SecureKeyValueStore {

    override fun get(key: String): String? = settings.getStringOrNull(key)

    override fun set(key: String, value: String) {
        settings.putString(key, value)
    }

    override fun remove(key: String) {
        settings.remove(key)
    }
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
