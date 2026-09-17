package com.runa.shared.feature.settings

/** The app-wide appearance theme.
 *  The shared module owns only the SELECTION (colors live in each client's `RunaColors`);
 *  [id] is persisted in settings and must never change. */
enum class AppTheme(val id: String) {
    /** 夜（ダーク）— the default. */
    DARK("dark"),

    /** あさ（ライト）— a bright, cream-based light theme. */
    LIGHT("light"),

    /** ピンク×ピンク — dark base with the accent pink pushed further. */
    PINK("pink");

    companion object {
        /** Maps a persisted [id] back to a theme, defaulting to [DARK] when absent or unrecognized. */
        fun fromId(id: String?): AppTheme = entries.firstOrNull { it.id == id } ?: DARK
    }
}
