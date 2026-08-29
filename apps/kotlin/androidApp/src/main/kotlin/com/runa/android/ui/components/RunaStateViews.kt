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

/**
 * The four shared state screens — the single home for LUNA's 空 / オフライン /
 * ローディング / エラー surfaces (confirmed designs 24–27). Every feature routes its
 * page-level [UiState] through [RunaStateView] instead of re-implementing loading /
 * empty / offline / error, so the world-view (the moon motif, the quiet voice) stays
 * consistent and identical to iOS's `RunaStateViews`.
 *
 * The emblems (`GlowingMoon` / `NewMoonEmblem` / `CloudedMoon` / `StumbleEmblem`) are
 * the fixed cross-theme motif from [com.runa.android.ui.components] (they do not
 * recolor with the theme); the surrounding text and CTAs read the theme tokens.
 * The loading indicator is a quiet moon + three dots (never a spinner) and honors
 * reduced motion ([rememberReducedMotion]).
 */

/**
 * The app-wide "re-authenticate" action, provided once by `RunaApp` (it clears the
 * session so the shared auth state drops to sign-in — the same door sign-out uses).
 * Defaulted here so the [RunaErrorView] auth CTA works without every screen threading
 * a callback; the global `TokenStore.sessionExpired` signal remains the primary
 * re-auth path (DoD #3).
 */
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
 * Dispatches a page-level [UiState] to the right shared surface. [empty] is a slot
 * so each feature supplies its own [RunaEmptyView] copy (the empty motif + layout is
 * shared, the words are the feature's). [content] renders the loaded body and is
 * handed the [SyncPhase] so the screen can place a [RunaSyncBanner] over it.
 *
 * [onRetry] backs the offline/error retry; [onReauthenticate] backs the
 * session-expired CTA (which drops the app to sign-in via the shared auth state).
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
            // Server / unknown share the same quiet 「読み込めませんでした。」 error surface.
            is AppError.Server, is AppError.Unknown -> RunaErrorView(onCta = onRetry, modifier = modifier)
        }
        is UiState.Content -> content(state.data, state.sync)
    }
}

/** Loading (26): a quiet glowing moon + three dots. Never a spinner; reduced-motion safe. */
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

/**
 * Empty (24): the new-moon emblem over a quiet invitation. Copy is per-feature —
 * an empty page is an invitation to begin, so callers pass their own [title]/[body]
 * and optional [ctaLabel]/[onCta].
 */
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

/** Offline (25): the clouded moon; what's cached is still shown behind this — this
 *  surface is only for when there is nothing to show. Quiet retry. */
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

/**
 * Error (27): the stumble emblem. Does not apologize — says what happened and how to
 * go on, in the world's voice. Defaults to the generic 「読み込めませんでした。」 copy; the auth
 * variant overrides the copy + CTA.
 */
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

/**
 * The quiet status line shown over cached content (DoD #2): offline/error only —
 * a running sync is signalled by the screen's own pull indicator, so Idle/Syncing
 * render nothing.
 */
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

/**
 * Centered column the state surfaces share. Fills the width; the caller controls the
 * height via [modifier] (`fillMaxSize()` for a full screen, a fixed height inside a
 * scroll). Vertical centering applies when the height is bounded.
 */
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

/** A bordered pill CTA. [accent] uses the moonlight-pink accent, else a quiet subtle outline. */
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

/** Three quiet dots. Animated (staggered fade) unless [animate] is false (reduced motion). */
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
