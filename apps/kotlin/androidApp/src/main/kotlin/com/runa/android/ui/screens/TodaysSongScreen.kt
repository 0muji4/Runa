package com.runa.android.ui.screens

import androidx.compose.runtime.Composable
import androidx.compose.ui.res.stringResource
import com.runa.android.R

/** きょうの一曲 tab — empty shell. TODO: today's-song feature slice. */
@Composable
fun TodaysSongScreen() {
    PlaceholderScreen(
        title = stringResource(R.string.tab_todays_song),
        placeholder = stringResource(R.string.placeholder_coming_soon),
    )
}
