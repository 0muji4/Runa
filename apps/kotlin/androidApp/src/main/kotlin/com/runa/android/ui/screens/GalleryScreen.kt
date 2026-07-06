package com.runa.android.ui.screens

import androidx.compose.runtime.Composable
import androidx.compose.ui.res.stringResource
import com.runa.android.R

/** ギャラリー tab — empty shell. TODO: gallery feature slice. */
@Composable
fun GalleryScreen() {
    PlaceholderScreen(
        title = stringResource(R.string.tab_gallery),
        placeholder = stringResource(R.string.placeholder_coming_soon),
    )
}
