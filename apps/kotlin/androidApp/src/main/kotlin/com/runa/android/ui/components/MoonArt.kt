package com.runa.android.ui.components

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.size
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.graphics.drawscope.Fill
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.drawscope.clipPath
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp

/**
 * Runa moon motif, drawn with the Canvas — NO emoji, NO text glyphs. Every screen
 * that needs a moon (sign-in, onboarding, notification, diary cards/detail, and
 * the empty/offline/error states) shares these primitives so the world-view stays
 * consistent across the app and identical in spirit to the iOS MoonArt.
 */

// The moon is a FIXED cross-theme motif (design decision): it never recolors with
// the app theme, so it holds its own dark-motif constants rather than reading the
// theme tokens. Values match the original dark palette.
private val MoonLit = Color(0xFFF7F2E4)         // bright limb
private val MoonCream = Color(0xFFE8E2D0)        // sub-accent cream
private val MoonDark = Color(0xFF34343E)         // unlit disc, a touch above the surface
private val MoonRing = Color(0xFF3E3E48)
private val MoonMuted = Color(0xFF9A9AA5)        // subtle
private val MoonAccent = Color(0xFFF4A9C0)       // moonlight accent
private val MoonBackground = Color(0xFF0E0E12)   // background

/** Soft radial glow used behind the bright moons. */
private fun DrawScope.drawGlow(center: Offset, radius: Float, tint: Color, alpha: Float) {
    drawCircle(
        brush = Brush.radialGradient(
            colors = listOf(tint.copy(alpha = alpha), Color.Transparent),
            center = center,
            radius = radius,
        ),
        radius = radius,
        center = center,
    )
}

/**
 * A full, softly glowing moon — the hero mark for sign-in / onboarding / splash.
 * The whole mark (glow included) fits within [diameter].
 */
@Composable
fun GlowingMoon(
    modifier: Modifier = Modifier,
    diameter: Dp = 132.dp,
    haloTint: Color = MoonCream,
) {
    Canvas(modifier.size(diameter)) {
        val c = center
        val moonR = size.minDimension * 0.30f
        // Outer halo, then a warmer inner glow.
        drawGlow(c, size.minDimension * 0.5f, haloTint, 0.22f)
        drawGlow(c, moonR * 1.7f, MoonLit, 0.30f)
        // Moon body with an off-centre highlight for a little dimension.
        drawCircle(
            brush = Brush.radialGradient(
                colors = listOf(MoonLit, MoonCream),
                center = Offset(c.x - moonR * 0.28f, c.y - moonR * 0.28f),
                radius = moonR * 1.5f,
            ),
            radius = moonR,
            center = c,
        )
    }
}

/**
 * A small moon-phase disc for diary cards / detail. The lit fraction comes from the
 * shared moon calculator ([illumination] 0..1); [waxing] puts the lit limb on the
 * right. Rendered with the offset-shadow method: a lit disc with a same-size shadow
 * disc slid toward the dark limb, clipped to the moon.
 */
@Composable
fun MoonPhaseDisc(
    illumination: Float,
    waxing: Boolean,
    modifier: Modifier = Modifier,
    diameter: Dp = 22.dp,
) {
    Canvas(modifier.size(diameter)) {
        drawMoonPhase(illumination.coerceIn(0f, 1f), waxing)
    }
}

private fun DrawScope.drawMoonPhase(illumination: Float, waxing: Boolean) {
    val c = center
    val r = size.minDimension / 2f * 0.92f
    val moon = Path().apply { addOvalCompat(c, r) }
    // Faint halo for the bright phases so a full moon glows a little.
    if (illumination > 0.55f) drawGlow(c, r * 1.9f, MoonCream, 0.18f * illumination)
    clipPath(moon) {
        // Fully lit disc, then darken the unlit limb.
        drawCircle(MoonCream, r, c)
        val dir = if (waxing) -1f else 1f // shadow slides toward the dark side
        val shadowX = c.x + dir * 2f * r * illumination
        drawCircle(MoonDark, r, Offset(shadowX, c.y))
    }
    // Thin rim keeps the disc legible on the near-black background.
    drawCircle(MoonRing, r, c, style = Stroke(width = 1.2f))
}

/** New-moon emblem for the empty state (24): a dark disc inside a faint ring. */
@Composable
fun NewMoonEmblem(modifier: Modifier = Modifier, diameter: Dp = 108.dp) {
    Canvas(modifier.size(diameter)) {
        val c = center
        val r = size.minDimension * 0.32f
        drawGlow(c, size.minDimension * 0.46f, MoonCream, 0.10f)
        drawCircle(MoonRing, r * 1.32f, c, style = Stroke(width = 1.4f))
        drawCircle(
            brush = Brush.radialGradient(
                colors = listOf(Color(0xFF26262E), Color(0xFF14141A)),
                center = Offset(c.x - r * 0.2f, c.y - r * 0.2f),
                radius = r * 1.3f,
            ),
            radius = r,
            center = c,
        )
    }
}

/** Clouded moon for the offline state (25): a dim disc crossed by a soft stroke. */
@Composable
fun CloudedMoon(modifier: Modifier = Modifier, diameter: Dp = 108.dp) {
    Canvas(modifier.size(diameter)) {
        val c = center
        val r = size.minDimension * 0.30f
        drawCircle(
            brush = Brush.radialGradient(
                colors = listOf(Color(0xFF3A3A44), Color(0xFF24242C)),
                center = Offset(c.x - r * 0.25f, c.y - r * 0.25f),
                radius = r * 1.4f,
            ),
            radius = r,
            center = c,
        )
        drawLine(
            color = MoonMuted,
            start = Offset(c.x - r * 1.15f, c.y + r * 0.8f),
            end = Offset(c.x + r * 1.15f, c.y - r * 0.8f),
            strokeWidth = 2f,
            cap = StrokeCap.Round,
        )
    }
}

/** Gentle "stumble" emblem for the error state (27): a soft disc holding an ! mark. */
@Composable
fun StumbleEmblem(modifier: Modifier = Modifier, diameter: Dp = 108.dp) {
    Canvas(modifier.size(diameter)) {
        val c = center
        val r = size.minDimension * 0.30f
        drawGlow(c, size.minDimension * 0.44f, MoonMuted, 0.12f)
        drawCircle(
            brush = Brush.radialGradient(
                colors = listOf(Color(0xFFB9B4AC), Color(0xFF8F8B84)),
                center = Offset(c.x - r * 0.25f, c.y - r * 0.25f),
                radius = r * 1.4f,
            ),
            radius = r,
            center = c,
        )
        // ! mark, drawn (no glyph): a stem and a dot in the background colour.
        val stemTop = Offset(c.x, c.y - r * 0.42f)
        val stemBottom = Offset(c.x, c.y + r * 0.10f)
        drawLine(MoonBackground, stemTop, stemBottom, strokeWidth = r * 0.16f, cap = StrokeCap.Round)
        drawCircle(MoonBackground, r * 0.09f, Offset(c.x, c.y + r * 0.40f))
    }
}

/**
 * The notification mark (04): the glowing moon with a small moonlight-pink badge
 * carrying a minimal bell — drawn, not an emoji.
 */
@Composable
fun NotificationMoon(modifier: Modifier = Modifier, diameter: Dp = 132.dp) {
    Canvas(modifier.size(diameter)) {
        val c = center
        val moonR = size.minDimension * 0.28f
        drawGlow(c, size.minDimension * 0.5f, MoonCream, 0.20f)
        drawGlow(c, moonR * 1.7f, MoonLit, 0.28f)
        drawCircle(
            brush = Brush.radialGradient(
                colors = listOf(MoonLit, MoonCream),
                center = Offset(c.x - moonR * 0.28f, c.y - moonR * 0.28f),
                radius = moonR * 1.5f,
            ),
            radius = moonR,
            center = c,
        )
        // Pink badge up and to the right of the moon.
        val badgeC = Offset(c.x + moonR * 0.95f, c.y - moonR * 0.95f)
        val badgeR = moonR * 0.6f
        drawGlow(badgeC, badgeR * 2.1f, MoonAccent, 0.35f)
        drawCircle(MoonAccent, badgeR, badgeC)
        drawBell(badgeC, badgeR * 0.92f, MoonBackground)
    }
}

/** A minimal bell: a rounded dome body, a base line and a clapper dot. */
private fun DrawScope.drawBell(c: Offset, s: Float, color: Color) {
    val w = s * 0.9f
    val h = s * 1.0f
    val body = Path().apply {
        moveTo(c.x - w * 0.5f, c.y + h * 0.28f)
        // sides sweep up into a dome
        cubicTo(
            c.x - w * 0.5f, c.y - h * 0.15f,
            c.x - w * 0.32f, c.y - h * 0.5f,
            c.x, c.y - h * 0.5f,
        )
        cubicTo(
            c.x + w * 0.32f, c.y - h * 0.5f,
            c.x + w * 0.5f, c.y - h * 0.15f,
            c.x + w * 0.5f, c.y + h * 0.28f,
        )
        close()
    }
    drawPath(body, color, style = Fill)
    // base line + clapper
    drawLine(
        color,
        Offset(c.x - w * 0.6f, c.y + h * 0.30f),
        Offset(c.x + w * 0.6f, c.y + h * 0.30f),
        strokeWidth = s * 0.14f,
        cap = StrokeCap.Round,
    )
    drawCircle(color, s * 0.12f, Offset(c.x, c.y + h * 0.5f))
}

/**
 * The privacy-lock emblem (22): a padlock inside a soft rounded square, stroked in
 * cream — the quiet motif the lock screen and the settings screen share. Drawn, no
 * glyph.
 */
@Composable
fun LockEmblem(modifier: Modifier = Modifier, diameter: Dp = 120.dp) {
    Canvas(modifier.size(diameter)) {
        val c = center
        val boxHalf = size.minDimension * 0.30f
        val corner = boxHalf * 0.5f
        drawGlow(c, size.minDimension * 0.5f, MoonCream, 0.10f)
        // Rounded-square container.
        drawRoundRect(
            color = MoonRing,
            topLeft = Offset(c.x - boxHalf, c.y - boxHalf),
            size = Size(boxHalf * 2, boxHalf * 2),
            cornerRadius = androidx.compose.ui.geometry.CornerRadius(corner, corner),
            style = Stroke(width = 2f),
        )
        // Padlock body.
        val bodyW = boxHalf * 0.9f
        val bodyH = boxHalf * 0.72f
        val bodyTop = c.y - bodyH * 0.1f
        drawRoundRect(
            color = MoonCream,
            topLeft = Offset(c.x - bodyW / 2f, bodyTop),
            size = Size(bodyW, bodyH),
            cornerRadius = androidx.compose.ui.geometry.CornerRadius(bodyW * 0.16f, bodyW * 0.16f),
            style = Stroke(width = 2.4f),
        )
        // Shackle arc above the body.
        val shackleR = bodyW * 0.32f
        val shacklePath = Path().apply {
            moveTo(c.x - shackleR, bodyTop)
            cubicTo(
                c.x - shackleR, bodyTop - shackleR * 1.6f,
                c.x + shackleR, bodyTop - shackleR * 1.6f,
                c.x + shackleR, bodyTop,
            )
        }
        drawPath(shacklePath, MoonCream, style = Stroke(width = 2.4f))
        // Keyhole dot.
        drawCircle(MoonCream, bodyW * 0.09f, Offset(c.x, bodyTop + bodyH * 0.5f))
    }
}

private fun Path.addOvalCompat(center: Offset, radius: Float) {
    addOval(
        androidx.compose.ui.geometry.Rect(
            offset = Offset(center.x - radius, center.y - radius),
            size = Size(radius * 2, radius * 2),
        ),
    )
}
