package com.runa.android.ui.theme

import androidx.compose.ui.unit.dp

/**
 * The single top margin every bottom-tab root (Home / Today's Song / Diary /
 * Gallery) starts its header at, measured below the status-bar inset the tab shell
 * already applies. Keeping it in one place is what makes the four tabs begin at the
 * same vertical offset — previously each screen hard-coded its own value (and the
 * Scaffold-based tabs also double-applied the status-bar inset), so the top gaps
 * differed from tab to tab.
 */
val RunaTabHeaderTop = 32.dp
