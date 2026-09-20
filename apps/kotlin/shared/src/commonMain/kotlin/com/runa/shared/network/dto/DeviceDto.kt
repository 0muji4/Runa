package com.runa.shared.network.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** Body for PUT /api/v1/devices; [notifyTime] is "HH:MM" in the IANA [timeZone]. */
@Serializable
data class RegisterDeviceRequest(
    @SerialName("install_id") val installId: String,
    @SerialName("push_token") val pushToken: String,
    val platform: String,
    @SerialName("notify_time") val notifyTime: String,
    @SerialName("time_zone") val timeZone: String,
    val enabled: Boolean,
)

@Serializable
data class DeviceDto(
    val id: String,
    @SerialName("install_id") val installId: String,
    @SerialName("push_token") val pushToken: String,
    val platform: String,
    @SerialName("notify_time") val notifyTime: String,
    @SerialName("time_zone") val timeZone: String,
    val enabled: Boolean,
    @SerialName("created_at") val createdAt: String,
    @SerialName("updated_at") val updatedAt: String,
)
