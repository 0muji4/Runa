package com.runa.android.ui.theme

import androidx.compose.ui.unit.dp

/** Header metrics used only by RunaScreenHeader; shared with iOS/README (drift-guarded). */
object RunaHeader {
    /** Top of a bottom-tab root's header (below the status-bar inset). */
    val TopTab = 40.dp

    /** Top of a pushed screen's「‹ 戻る」row. */
    val TopPushed = 14.dp

    /**「‹ 戻る」to the title. */
    val BackGap = 24.dp

    /** Title to the body below it. */
    val Bottom = 24.dp
}
