package com.runa.android.ui.components

import androidx.compose.foundation.Image
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.height
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.runa.android.R
import com.runa.android.ui.theme.RunaColors

/**
 * Apple's official badge (asset is Apple's own, never redrawn); Apple's terms require it
 * next to any preview or artwork.
 */
@Composable
fun AppleMusicBadge(storeUrl: String, modifier: Modifier = Modifier, height: Dp = 40.dp) {
    val uriHandler = LocalUriHandler.current
    Image(
        painter = painterResource(R.drawable.apple_music_badge),
        contentDescription = stringResource(R.string.song_listen_on_apple_music),
        modifier = modifier
            .height(height)
            .clickable(role = Role.Button) { uriHandler.openUri(storeUrl) },
    )
}

/** The attribution Apple's terms require wherever a preview is offered. */
@Composable
fun ITunesCourtesyLine(modifier: Modifier = Modifier) {
    Text(
        text = stringResource(R.string.song_courtesy),
        style = MaterialTheme.typography.labelMedium.copy(fontSize = 11.sp),
        color = RunaColors.Subtle,
        modifier = modifier,
    )
}
