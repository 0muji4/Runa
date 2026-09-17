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
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import coil.compose.AsyncImage
import com.runa.android.R
import com.runa.android.ui.components.RunaEmptyView
import com.runa.android.ui.components.RunaScreenHeader
import com.runa.android.ui.components.RunaStateView
import com.runa.android.ui.components.RunaSyncBanner
import com.runa.android.ui.screens.gallery.ImageNormalizer
import com.runa.android.ui.theme.RunaColors
import com.runa.android.ui.theme.ZenKakuGothicNew
import com.runa.shared.core.state.SyncPhase
import com.runa.shared.core.state.UiState
import com.runa.shared.feature.gallery.GalleryImage
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
 * ギャラリー: masonry grid of the user's photos as they are. Renders from the local DB;
 * adds queue offline and flush on reconnect.
 */
@Composable
fun GalleryScreen(viewModel: GalleryViewModel = koinInject()) {
    val state by viewModel.state.collectAsStateWithLifecycle()
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var lightboxIndex by rememberSaveable { mutableStateOf<Int?>(null) }

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
            val sync = (state as? UiState.Content<List<GalleryImage>>)?.sync ?: SyncPhase.Idle
            RunaSyncBanner(sync)
            Box(Modifier.weight(1f).fillMaxWidth()) {
                RunaStateView(
                    state = state,
                    onRetry = viewModel::refresh,
                    empty = {
                        RunaEmptyView(
                            title = stringResource(R.string.gallery_empty_line),
                            body = stringResource(R.string.gallery_empty_body),
                            modifier = Modifier.fillMaxSize(),
                        )
                    },
                    modifier = Modifier.fillMaxSize(),
                ) { images, _ ->
                    GalleryGrid(images, onOpen = { index -> lightboxIndex = index })
                }
            }
        }

        val current = state
        val index = lightboxIndex
        if (current is UiState.Content<List<GalleryImage>> && index != null) {
            val images = current.data
            if (index in images.indices) {
                Lightbox(
                    images = images,
                    startIndex = index,
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
    RunaScreenHeader(title = stringResource(R.string.gallery_title)) {
        Text(
            text = "＋",
            style = TextStyle(fontFamily = ZenKakuGothicNew, fontSize = 24.sp),
            color = RunaColors.Subtle,
            modifier = Modifier
                .clip(RoundedCornerShape(20.dp))
                .clickable(onClick = onAdd)
                .padding(horizontal = 8.dp, vertical = 4.dp),
        )
    }
}

@Composable
private fun GalleryGrid(
    images: List<GalleryImage>,
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
            GalleryCell(image) { onOpen(index) }
        }
    }
}

@Composable
private fun GalleryCell(image: GalleryImage, onClick: () -> Unit) {
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
private fun Lightbox(
    images: List<GalleryImage>,
    startIndex: Int,
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

private fun formatDateTime(epochMs: Long): String =
    SimpleDateFormat("M月d日  HH:mm", Locale.JAPAN).format(Date(epochMs))
