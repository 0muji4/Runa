package com.runa.android.ui.screens

import androidx.compose.runtime.Composable
import androidx.compose.ui.res.stringResource
import com.runa.android.R

/** ダイアリー tab — empty shell. TODO: diary feature slice. */
@Composable
fun DiaryScreen() {
    PlaceholderScreen(
        title = stringResource(R.string.tab_diary),
        placeholder = stringResource(R.string.placeholder_coming_soon),
    )
}
