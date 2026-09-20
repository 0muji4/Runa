package com.runa.shared.feature.push

import com.russhwolf.settings.Settings
import kotlin.uuid.ExperimentalUuidApi
import kotlin.uuid.Uuid

/** A UUID minted once per install and kept across token rotation; the server keys device rows by it. */
class InstallIdProvider(
    private val settings: Settings,
) {
    @OptIn(ExperimentalUuidApi::class)
    fun get(): String {
        settings.getStringOrNull(KEY_INSTALL_ID)?.let { return it }
        val id = Uuid.random().toString()
        settings.putString(KEY_INSTALL_ID, id)
        return id
    }

    // Renaming the key would mint a new id and orphan the server's device row.
    private companion object {
        const val KEY_INSTALL_ID = "notif.push.install_id"
    }
}
