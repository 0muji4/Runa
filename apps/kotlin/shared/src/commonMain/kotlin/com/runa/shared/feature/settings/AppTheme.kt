package com.runa.shared.feature.settings

/**
 * The app-wide appearance theme, selected by the user and applied across every
 * screen. This is distinct from the gallery's per-image / display treatment
 * ([com.runa.shared.feature.gallery.GalleryTheme] /
 * [com.runa.shared.feature.gallery.GalleryDisplayTheme]) — those style images
 * inside the gallery, whereas this recolors the whole application.
 *
 * The token VALUES for each theme live natively in each client (Compose
 * `RunaColors`, SwiftUI `RunaColors`), kept identical to the canonical table in
 * the repo README. The shared module owns only the SELECTION; the clients own the
 * colors. [id] is the stable string persisted in settings and never localized.
 */
enum class AppTheme(val id: String) {
    /** 夜（ダーク）— the default, the look the app was designed around first. */
    DARK("dark"),

    /** あさ（ライト）— a bright, cream-based light theme. */
    LIGHT("light"),

    /** ピンク×ピンク — dark base with the accent pink pushed further. */
    PINK("pink");

    companion object {
        /** Maps a persisted [id] back to a theme, defaulting to [DARK] for an
         *  absent or unrecognized value. */
        fun fromId(id: String?): AppTheme = entries.firstOrNull { it.id == id } ?: DARK
    }
}
