package com.runa.android.ui.theme

import androidx.compose.ui.unit.dp

/**
 * Header metrics. Every screen header is built from these four values by
 * [com.runa.android.ui.components.RunaScreenHeader] — screens do not set their own.
 *
 * The values are pinned by the canon table in README「画面ヘッダー（全画面共通の型）」
 * and must match iOS `RunaHeaderMetrics`; `hack/check-header-tokens.sh` verifies it.
 * Keeping them in one place is what makes every screen start at the same vertical
 * offset — previously each screen hard-coded its own (5 different top offsets and
 * 5 different gaps below the title), so the headers differed from screen to screen.
 */
object RunaHeader {
    /** Top of a bottom-tab root's header, below the status-bar inset the shell applies. */
    val TopTab = 40.dp

    /** Top of a pushed screen's「‹ 戻る」row. */
    val TopPushed = 14.dp

    /**「‹ 戻る」to the title. */
    val BackGap = 24.dp

    /** Title to the body below it. */
    val Bottom = 24.dp
}
