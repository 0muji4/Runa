package com.runa.android.ui.screens

import android.net.Uri
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.staggeredgrid.LazyVerticalStaggeredGrid
import androidx.compose.foundation.lazy.staggeredgrid.StaggeredGridCells
import androidx.compose.foundation.lazy.staggeredgrid.itemsIndexed
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.ColorFilter
import androidx.compose.ui.graphics.ColorMatrix
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImage
import com.runa.android.R
import com.runa.android.ui.components.NewMoonEmblem
import com.runa.android.ui.screens.gallery.ImageNormalizer
import com.runa.android.ui.theme.RunaColors
import com.runa.android.ui.theme.ShipporiMincho
import com.runa.android.ui.theme.ZenKakuGothicNew
import com.runa.shared.feature.gallery.GalleryBanner
import com.runa.shared.feature.gallery.GalleryDisplayTheme
import com.runa.shared.feature.gallery.GalleryImage
import com.runa.shared.feature.gallery.GalleryUiState
import com.runa.shared.feature.gallery.GalleryViewModel
import com.runa.shared.feature.gallery.UploadState
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import org.koin.compose.koinInject
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

/**
 * 13 ギャラリー — "ひかりの記録". A whitespace-rich masonry grid of the user's images
 * with a gallery-scoped display-theme toggle (monotone ⇔ pink) that re-grades the
 * whole grid — this is NOT the app-wide theme. Tapping a cell opens the lightbox (14).
 * Everything renders from the local DB; adds queue offline and flush on reconnect.
 */
@Composable
fun GalleryScreen(viewModel: GalleryViewModel = koinInject()) {
    val state by viewModel.state.collectAsState()
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var lightboxIndex by remember { mutableStateOf<Int?>(null) }

    val picker = rememberLauncherForActivityResult(ActivityResultContracts.PickVisualMedia()) { uri: Uri? ->
        if (uri != null) {
            scope.launch {
                val picked = withContext(Dispatchers.IO) { ImageNormalizer.normalize(context, uri) }
                picked?.let { viewModel.addImage(it.bytes, it.width, it.height, it.mimeType) }
            }
        }
    }
    val launchPicker = {
        picker.launch(PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly))
    }

    Box(Modifier.fillMaxSize().background(RunaColors.Background)) {
        Column(Modifier.fillMaxSize().padding(horizontal = 20.dp)) {
            GalleryHeader(onAdd = launchPicker)
            when (val s = state) {
                is GalleryUiState.Content -> {
                    ThemeToggle(s.displayTheme, viewModel::setDisplayTheme)
                    Banner(s.banner)
                    GalleryGrid(s.images, s.displayTheme, onOpen = { index -> lightboxIndex = index })
                }
                is GalleryUiState.Empty -> {
                    ThemeToggle(s.displayTheme, viewModel::setDisplayTheme)
                    Banner(s.banner)
                    GalleryEmpty()
                }
                GalleryUiState.Loading -> Spacer(Modifier.height(240.dp))
                is GalleryUiState.Error -> GalleryEmpty()
            }
        }

        // Lightbox (14) as a full-screen overlay over the grid state.
        val current = state
        val index = lightboxIndex
        if (current is GalleryUiState.Content && index != null) {
            if (index in current.images.indices) {
                Lightbox(
                    images = current.images,
                    startIndex = index,
                    displayTheme = current.displayTheme,
                    onClose = { lightboxIndex = null },
                    onDelete = { clientId ->
                        viewModel.deleteImage(clientId)
                        lightboxIndex = null
                    },
                )
            } else {
                lightboxIndex = null
            }
        }
    }
}

@Composable
private fun GalleryHeader(onAdd: () -> Unit) {
    Box(Modifier.fillMaxWidth().padding(top = 20.dp, bottom = 4.dp)) {
        Text(
            text = stringResource(R.string.gallery_title),
            style = TextStyle(fontFamily = ShipporiMincho, fontSize = 26.sp, letterSpacing = 6.sp),
            color = RunaColors.Heading,
            modifier = Modifier.align(Alignment.Center),
        )
        Text(
            text = "＋",
            style = TextStyle(fontFamily = ZenKakuGothicNew, fontSize = 24.sp),
            color = RunaColors.Subtle,
            modifier = Modifier
                .align(Alignment.CenterEnd)
                .clip(RoundedCornerShape(20.dp))
                .clickable(onClick = onAdd)
                .padding(horizontal = 8.dp, vertical = 4.dp),
        )
    }
}

@Composable
private fun ThemeToggle(selected: GalleryDisplayTheme, onSelect: (GalleryDisplayTheme) -> Unit) {
    Row(
        Modifier.fillMaxWidth().padding(top = 8.dp),
        horizontalArrangement = Arrangement.Center,
    ) {
        Surface(color = RunaColors.Surface, shape = RoundedCornerShape(24.dp)) {
            Row(Modifier.padding(4.dp)) {
                ThemeSegment(
                    label = stringResource(R.string.gallery_theme_monotone),
                    selected = selected == GalleryDisplayTheme.MONOTONE,
                ) { onSelect(GalleryDisplayTheme.MONOTONE) }
                ThemeSegment(
                    label = stringResource(R.string.gallery_theme_pink),
                    selected = selected == GalleryDisplayTheme.PINK,
                ) { onSelect(GalleryDisplayTheme.PINK) }
            }
        }
    }
}

@Composable
private fun ThemeSegment(label: String, selected: Boolean, onClick: () -> Unit) {
    Box(
        Modifier
            .clip(RoundedCornerShape(20.dp))
            .background(if (selected) RunaColors.Accent else Color.Transparent)
            .clickable(onClick = onClick)
            .padding(horizontal = 26.dp, vertical = 8.dp),
    ) {
        Text(
            text = label,
            style = TextStyle(fontFamily = ZenKakuGothicNew, fontSize = 15.sp),
            color = if (selected) RunaColors.Background else RunaColors.Subtle,
        )
    }
}

@Composable
private fun GalleryGrid(
    images: List<GalleryImage>,
    displayTheme: GalleryDisplayTheme,
    onOpen: (Int) -> Unit,
) {
    LazyVerticalStaggeredGrid(
        columns = StaggeredGridCells.Fixed(2),
        modifier = Modifier.fillMaxSize().padding(top = 20.dp),
        verticalItemSpacing = 16.dp,
        horizontalArrangement = Arrangement.spacedBy(16.dp),
        contentPadding = PaddingValues(bottom = 32.dp),
    ) {
        itemsIndexed(images, key = { _, image -> image.clientId }) { index, image ->
            GalleryCell(image, displayTheme) { onOpen(index) }
        }
    }
}

@Composable
private fun GalleryCell(image: GalleryImage, displayTheme: GalleryDisplayTheme, onClick: () -> Unit) {
    val aspect = if (image.height > 0) image.width.toFloat() / image.height else 1f
    Box(
        Modifier
            .fillMaxWidth()
            .aspectRatio(aspect.coerceIn(0.6f, 1.6f))
            .clip(RoundedCornerShape(20.dp))
            .background(RunaColors.Surface)
            .clickable(onClick = onClick),
    ) {
        AsyncImage(
            model = image.viewUrl ?: image.localBytes,
            contentDescription = null,
            contentScale = ContentScale.Crop,
            colorFilter = displayTheme.colorFilter(),
            modifier = Modifier.fillMaxSize(),
        )
        if (image.uploadState == UploadState.Uploading || image.uploadState == UploadState.Queued) {
            Box(
                Modifier.fillMaxSize().background(RunaColors.Background.copy(alpha = 0.35f)),
                contentAlignment = Alignment.Center,
            ) {
                CircularProgressIndicator(
                    progress = { image.progress.coerceIn(0f, 1f) },
                    color = RunaColors.Accent,
                    trackColor = RunaColors.Subtle.copy(alpha = 0.3f),
                )
            }
        }
    }
}

@Composable
private fun GalleryEmpty() {
    Column(
        Modifier.fillMaxSize().padding(top = 80.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        NewMoonEmblem()
        Spacer(Modifier.height(28.dp))
        Text(
            text = stringResource(R.string.gallery_empty_line),
            style = TextStyle(fontFamily = ShipporiMincho, fontSize = 16.sp),
            color = RunaColors.Subtle,
            textAlign = TextAlign.Center,
        )
    }
}

@Composable
private fun Banner(banner: GalleryBanner) {
    val text = when (banner) {
        GalleryBanner.Offline -> stringResource(R.string.diary_banner_offline)
        GalleryBanner.Error -> stringResource(R.string.diary_banner_error)
        else -> return // None / Syncing stay silent
    }
    Text(
        text = text,
        style = TextStyle(fontFamily = ZenKakuGothicNew, fontSize = 13.sp),
        color = RunaColors.Subtle,
        textAlign = TextAlign.Center,
        modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
    )
}

@Composable
private fun Lightbox(
    images: List<GalleryImage>,
    startIndex: Int,
    displayTheme: GalleryDisplayTheme,
    onClose: () -> Unit,
    onDelete: (String) -> Unit,
) {
    val pagerState = rememberPagerState(initialPage = startIndex) { images.size }
    Box(Modifier.fillMaxSize().background(RunaColors.Background.copy(alpha = 0.98f))) {
        HorizontalPager(state = pagerState, modifier = Modifier.fillMaxSize()) { page ->
            val image = images[page]
            Column(
                Modifier.fillMaxSize().padding(24.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.Center,
            ) {
                AsyncImage(
                    model = image.viewUrl ?: image.localBytes,
                    contentDescription = null,
                    contentScale = ContentScale.Fit,
                    colorFilter = displayTheme.colorFilter(),
                    modifier = Modifier.fillMaxWidth().clip(RoundedCornerShape(24.dp)),
                )
                Spacer(Modifier.height(28.dp))
                Text(
                    text = formatDateTime(image.createdAtEpochMs),
                    style = TextStyle(fontFamily = ZenKakuGothicNew, fontSize = 14.sp, letterSpacing = 2.sp),
                    color = RunaColors.Subtle,
                )
            }
        }
        Text(
            text = "✕",
            style = TextStyle(fontFamily = ZenKakuGothicNew, fontSize = 22.sp),
            color = RunaColors.Body,
            modifier = Modifier
                .align(Alignment.TopStart)
                .padding(8.dp)
                .clip(RoundedCornerShape(20.dp))
                .clickable(onClick = onClose)
                .padding(horizontal = 10.dp, vertical = 6.dp),
        )
        Text(
            text = stringResource(R.string.gallery_delete),
            style = TextStyle(fontFamily = ZenKakuGothicNew, fontSize = 14.sp),
            color = RunaColors.Subtle,
            modifier = Modifier
                .align(Alignment.TopEnd)
                .padding(8.dp)
                .clip(RoundedCornerShape(20.dp))
                .clickable {
                    images.getOrNull(pagerState.currentPage)?.let { onDelete(it.clientId) }
                }
                .padding(horizontal = 12.dp, vertical = 6.dp),
        )
    }
}

/** Monotone = full desaturation; pink = a black→#F4A9C0 duotone. Applied to grid + lightbox. */
private fun GalleryDisplayTheme.colorFilter(): ColorFilter = when (this) {
    GalleryDisplayTheme.MONOTONE -> ColorFilter.colorMatrix(ColorMatrix().apply { setToSaturation(0f) })
    GalleryDisplayTheme.PINK -> ColorFilter.colorMatrix(PinkDuotone)
}

// Black→pink duotone: map luminance onto #F4A9C0 (0.957, 0.663, 0.753).
private val PinkDuotone = ColorMatrix(
    floatArrayOf(
        0.286f, 0.562f, 0.109f, 0f, 0f,
        0.198f, 0.389f, 0.076f, 0f, 0f,
        0.225f, 0.442f, 0.086f, 0f, 0f,
        0f, 0f, 0f, 1f, 0f,
    ),
)

private fun formatDateTime(epochMs: Long): String =
    SimpleDateFormat("M月d日  HH:mm", Locale.JAPAN).format(Date(epochMs))
