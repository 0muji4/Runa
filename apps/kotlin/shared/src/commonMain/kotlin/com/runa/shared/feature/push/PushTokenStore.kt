package com.runa.shared.feature.push

import com.russhwolf.settings.Settings
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/** Holds the latest FCM / APNs token; fed by the Android messaging service and the Swift AppDelegate. */
class PushTokenStore(
    private val settings: Settings,
) {
    private val _token = MutableStateFlow(settings.getStringOrNull(KEY_TOKEN))

    val token: StateFlow<String?> = _token.asStateFlow()

    fun set(token: String) {
        settings.putString(KEY_TOKEN, token)
        _token.value = token
    }

    /** Synchronous current value, so the iOS observable can seed without a flash. */
    fun currentToken(): String? = _token.value

    private companion object {
        const val KEY_TOKEN = "notif.push.token"
    }
}
