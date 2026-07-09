package com.runa.android.ui.screens.diary

import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.Locale

/** Formats an epoch-millis timestamp as a quiet Japanese date line, e.g. 7月9日 22:14. */
private val dateFormatter: DateTimeFormatter =
    DateTimeFormatter.ofPattern("M月d日 HH:mm", Locale.JAPAN)

fun formatDiaryDate(epochMs: Long): String =
    Instant.ofEpochMilli(epochMs).atZone(ZoneId.systemDefault()).format(dateFormatter)
