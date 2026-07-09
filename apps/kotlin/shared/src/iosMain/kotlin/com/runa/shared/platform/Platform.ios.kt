package com.runa.shared.platform

import app.cash.sqldelight.db.SqlDriver
import app.cash.sqldelight.driver.native.NativeSqliteDriver
import com.russhwolf.settings.ExperimentalSettingsImplementation
import com.russhwolf.settings.KeychainSettings
import com.runa.shared.db.RunaDatabase
import com.runa.shared.network.NetworkMonitor
import com.runa.shared.network.auth.SecureKeyValueStore
import io.ktor.client.engine.HttpClientEngine
import io.ktor.client.engine.darwin.Darwin
import kotlinx.cinterop.ExperimentalForeignApi
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import org.koin.core.module.Module
import org.koin.dsl.module
import platform.Network.nw_path_get_status
import platform.Network.nw_path_monitor_create
import platform.Network.nw_path_monitor_set_queue
import platform.Network.nw_path_monitor_set_update_handler
import platform.Network.nw_path_monitor_start
import platform.Network.nw_path_status_satisfied
import platform.darwin.dispatch_queue_create

/** iOS uses the Darwin (NSURLSession) Ktor engine. */
actual fun httpClientEngine(): HttpClientEngine = Darwin.create()

/**
 * iOS Koin bindings: the Keychain-backed secure store, the SQLDelight native
 * driver (persisted to `runa.db`) and the connectivity monitor.
 */
actual fun platformModule(): Module = module {
    single<SecureKeyValueStore> { KeychainSecureStore() }
    single<SqlDriver> { NativeSqliteDriver(RunaDatabase.Schema, "runa.db") }
    single<NetworkMonitor> { IosNetworkMonitor() }
}

/**
 * [NetworkMonitor] over `NWPathMonitor`. The update handler fires on a private
 * dispatch queue whenever the path changes; we publish "satisfied" as online.
 */
@OptIn(ExperimentalForeignApi::class)
class IosNetworkMonitor : NetworkMonitor {
    private val _isOnline = MutableStateFlow(true)
    override val isOnline: StateFlow<Boolean> = _isOnline.asStateFlow()

    init {
        val monitor = nw_path_monitor_create()
        nw_path_monitor_set_update_handler(monitor) { path ->
            _isOnline.value = nw_path_get_status(path) == nw_path_status_satisfied
        }
        nw_path_monitor_set_queue(monitor, dispatch_queue_create("com.runa.network.monitor", null))
        nw_path_monitor_start(monitor)
    }
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
