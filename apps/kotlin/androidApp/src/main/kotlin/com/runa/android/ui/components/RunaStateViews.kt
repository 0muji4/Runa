package com.runa.android.ui.components

import android.provider.Settings
import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.runa.android.R
import com.runa.android.ui.theme.RunaColors
import com.runa.shared.core.state.AppError
import com.runa.shared.core.state.SyncPhase
import com.runa.shared.core.state.UiState

/** App-wide re-authenticate action, provided once by `RunaApp`; clears the session. */
val LocalReauthenticate = staticCompositionLocalOf<() -> Unit> { {} }

/** True when the OS animation scale is 0 (reduced motion). Read once per composition. */
@Composable
fun rememberReducedMotion(): Boolean {
    val context = LocalContext.current
    return remember(context) {
        Settings.Global.getFloat(
            context.contentResolver,
            Settings.Global.ANIMATOR_DURATION_SCALE,
            1f,
        ) == 0f
    }
}

/**
 * Dispatches a page-level [UiState] to the shared surface. [empty] is a per-feature slot;
 * [content] receives the [SyncPhase] so the screen can place a [RunaSyncBanner] over it.
 */
@Composable
fun <T> RunaStateView(
    state: UiState<T>,
    onRetry: () -> Unit,
    empty: @Composable () -> Unit,
    modifier: Modifier = Modifier,
    onReauthenticate: () -> Unit = LocalReauthenticate.current,
    content: @Composable (data: T, sync: SyncPhase) -> Unit,
) {
    when (state) {
        UiState.Loading -> RunaLoadingView(modifier = modifier)
        UiState.Empty -> empty()
        is UiState.Failure -> when (state.error) {
            AppError.Offline -> RunaOfflineView(onRetry = onRetry, modifier = modifier)
            is AppError.Auth -> RunaErrorView(
                title = stringResource(R.string.state_auth_title),
                body = stringResource(R.string.state_auth_body),
                ctaLabel = stringResource(R.string.state_auth_cta),
                onCta = onReauthenticate,
                modifier = modifier,
            )
            is AppError.Server, is AppError.Unknown -> RunaErrorView(onCta = onRetry, modifier = modifier)
        }
        is UiState.Content -> content(state.data, state.sync)
    }
}

/** Loading: a glowing moon + three dots. Never a spinner; reduced-motion safe. */
@Composable
fun RunaLoadingView(
    modifier: Modifier = Modifier,
    caption: String = stringResource(R.string.state_loading_caption),
) {
    StateScaffold(modifier) {
        GlowingMoon(diameter = 132.dp)
        Spacer(Modifier.height(28.dp))
        Text(
            text = caption,
            style = MaterialTheme.typography.titleLarge,
            color = RunaColors.Heading,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(24.dp))
        ThreeDotProgress(animate = !rememberReducedMotion())
    }
}

/** Empty: the new-moon emblem over per-feature copy. */
@Composable
fun RunaEmptyView(
    title: String,
    body: String,
    modifier: Modifier = Modifier,
    ctaLabel: String? = null,
    onCta: (() -> Unit)? = null,
) {
    StateScaffold(modifier) {
        NewMoonEmblem(diameter = 116.dp)
        Spacer(Modifier.height(28.dp))
        Text(title, style = MaterialTheme.typography.headlineMedium, color = RunaColors.Heading, textAlign = TextAlign.Center)
        Spacer(Modifier.height(14.dp))
        Text(body, style = MaterialTheme.typography.bodyMedium, color = RunaColors.Subtle, textAlign = TextAlign.Center)
        if (ctaLabel != null && onCta != null) {
            Spacer(Modifier.height(36.dp))
            RunaPillButton(label = ctaLabel, onClick = onCta, accent = true)
        }
    }
}

/** Offline, full-page: only when there is nothing cached to show (else [RunaSyncBanner]). */
@Composable
fun RunaOfflineView(
    onRetry: () -> Unit,
    modifier: Modifier = Modifier,
) {
    StateScaffold(modifier) {
        CloudedMoon(diameter = 116.dp)
        Spacer(Modifier.height(28.dp))
        Text(
            text = stringResource(R.string.state_offline_title),
            style = MaterialTheme.typography.headlineMedium,
            color = RunaColors.Heading,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(14.dp))
        Text(
            text = stringResource(R.string.state_offline_body),
            style = MaterialTheme.typography.bodyMedium,
            color = RunaColors.Subtle,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(36.dp))
        RunaPillButton(label = stringResource(R.string.state_offline_cta), onClick = onRetry, accent = false)
    }
}

/** Error: the stumble emblem. Defaults to the generic copy; the auth variant overrides copy + CTA. */
@Composable
fun RunaErrorView(
    onCta: () -> Unit,
    modifier: Modifier = Modifier,
    title: String = stringResource(R.string.state_error_title),
    body: String = stringResource(R.string.state_error_body),
    ctaLabel: String = stringResource(R.string.state_error_cta),
) {
    StateScaffold(modifier) {
        StumbleEmblem(diameter = 116.dp)
        Spacer(Modifier.height(28.dp))
        Text(title, style = MaterialTheme.typography.headlineMedium, color = RunaColors.Heading, textAlign = TextAlign.Center)
        Spacer(Modifier.height(14.dp))
        Text(body, style = MaterialTheme.typography.bodyMedium, color = RunaColors.Subtle, textAlign = TextAlign.Center)
        Spacer(Modifier.height(36.dp))
        RunaPillButton(label = ctaLabel, onClick = onCta, accent = true)
    }
}

/** Status line over cached content: offline/error only; Idle/Syncing render nothing. */
@Composable
fun RunaSyncBanner(phase: SyncPhase, modifier: Modifier = Modifier) {
    val text = when (phase) {
        SyncPhase.Idle, SyncPhase.Syncing -> null
        SyncPhase.Offline -> stringResource(R.string.state_banner_offline)
        SyncPhase.Error -> stringResource(R.string.state_banner_error)
    } ?: return

    Text(
        text = text,
        style = MaterialTheme.typography.labelLarge,
        color = RunaColors.Subtle,
        textAlign = TextAlign.Center,
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 24.dp, vertical = 8.dp),
    )
}

/** Shared centered column; the caller bounds the height via [modifier] for vertical centering. */
@Composable
private fun StateScaffold(modifier: Modifier, content: @Composable () -> Unit) {
    Column(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 40.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
        content = { content() },
    )
}

/** Bordered pill CTA. [accent] uses the accent color, else the subtle outline. */
@Composable
private fun RunaPillButton(label: String, onClick: () -> Unit, accent: Boolean) {
    val tint = if (accent) RunaColors.Accent else RunaColors.Subtle
    Box(
        modifier = Modifier
            .clickable(onClick = onClick)
            .border(1.dp, tint.copy(alpha = 0.7f), RoundedCornerShape(28.dp))
            .padding(horizontal = 32.dp, vertical = 14.dp),
    ) {
        Text(text = label, style = MaterialTheme.typography.bodyLarge, color = tint)
    }
}

/** Three dots, staggered fade unless [animate] is false (reduced motion). */
@Composable
private fun ThreeDotProgress(animate: Boolean) {
    val accent = RunaColors.Accent
    val idle = RunaColors.Subtle.copy(alpha = 0.45f)

    if (!animate) {
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            Dot(accent)
            Dot(idle)
            Dot(idle)
        }
        return
    }

    val transition = rememberInfiniteTransition(label = "loading-dots")
    Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
        repeat(3) { index ->
            val alpha by transition.animateFloat(
                initialValue = 0.3f,
                targetValue = 1f,
                animationSpec = infiniteRepeatable(
                    animation = tween(durationMillis = 900, delayMillis = index * 200, easing = LinearEasing),
                    repeatMode = RepeatMode.Reverse,
                ),
                label = "loading-dot-$index",
            )
            Dot(accent.copy(alpha = alpha))
        }
    }
}

@Composable
private fun Dot(color: Color) {
    Box(Modifier.size(8.dp).background(color, CircleShape))
}
