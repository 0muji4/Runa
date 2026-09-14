package com.runa.shared.feature.settings

import com.runa.shared.network.dto.ExportDto
import kotlinx.serialization.json.Json

/** Renders an [ExportDto] into the two formats the account screen offers: JSON and plain text. */
object SettingsExport {

    private val json = Json { prettyPrint = true }

    fun toJson(dto: ExportDto): String = json.encodeToString(ExportDto.serializer(), dto)

    fun toPlainText(dto: ExportDto): String = buildString {
        appendLine("LUNA データエクスポート")
        appendLine("エクスポート日時: ${dto.exportedAt}")
        appendLine("ユーザー: ${dto.user.displayName}")
        dto.user.email?.let { appendLine("メール: $it") }
        appendLine()
        appendLine("── 日記 (${dto.diaries.size}) ──")
        dto.diaries.sortedBy { it.createdAt }.forEach { entry ->
            appendLine()
            appendLine(entry.createdAt)
            appendLine(entry.bodyText)
            entry.mood?.let { appendLine("気分: $it") }
        }
        if (dto.images.isNotEmpty()) {
            appendLine()
            appendLine("── 画像 (${dto.images.size}) ──")
            dto.images.forEach { image ->
                appendLine("${image.createdAt}  ${image.url ?: image.objectKey}")
            }
        }
    }
}
