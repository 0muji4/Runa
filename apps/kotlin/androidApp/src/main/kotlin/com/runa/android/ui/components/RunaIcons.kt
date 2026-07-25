package com.runa.android.ui.components

import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.graphics.vector.path
import androidx.compose.ui.unit.dp

/**
 * The app's own icon set, defined as [ImageVector]s so we depend on neither the
 * material-icons artifact (which the project deliberately avoids) nor emoji/text
 * glyphs. Each icon is a single-color vector; render it through the material3
 * `Icon` composable so the current content color tints it (bottom-nav selection,
 * subtle settings rows, etc.). Paths are 24x24, matching the design's line motifs.
 *
 * Companion to [MoonArt] (the drawn moon motif); this covers the flat UI icons.
 */
object RunaIcons {

    /** Bottom-nav ホーム / テーマ row: a crescent moon. */
    val Moon: ImageVector = filled("Moon") {
        moveTo(12f, 3f)
        curveTo(7.03f, 3f, 3f, 7.03f, 3f, 12f)
        reflectiveCurveTo(7.03f, 21f, 12f, 21f)
        curveToRelative(4.97f, 0f, 9f, -4.03f, 9f, -9f)
        curveToRelative(0f, -0.46f, -0.04f, -0.92f, -0.1f, -1.36f)
        curveToRelative(-0.98f, 1.37f, -2.58f, 2.26f, -4.4f, 2.26f)
        curveToRelative(-2.98f, 0f, -5.4f, -2.42f, -5.4f, -5.4f)
        curveToRelative(0f, -1.81f, 0.89f, -3.42f, 2.26f, -4.4f)
        curveTo(12.92f, 3.04f, 12.46f, 3f, 12f, 3f)
        close()
    }

    /** Bottom-nav きょうの一曲: a music note. */
    val MusicNote: ImageVector = filled("MusicNote") {
        moveTo(12f, 3f)
        verticalLineToRelative(10.55f)
        curveToRelative(-0.59f, -0.34f, -1.27f, -0.55f, -2f, -0.55f)
        curveToRelative(-2.21f, 0f, -4f, 1.79f, -4f, 4f)
        reflectiveCurveToRelative(1.79f, 4f, 4f, 4f)
        reflectiveCurveToRelative(4f, -1.79f, 4f, -4f)
        verticalLineTo(7f)
        horizontalLineToRelative(4f)
        verticalLineTo(3f)
        horizontalLineToRelative(-6f)
        close()
    }

    /** Bottom-nav ダイアリー: a written page. */
    val Document: ImageVector = filled("Document") {
        moveTo(19f, 3f)
        horizontalLineTo(5f)
        curveTo(3.9f, 3f, 3f, 3.9f, 3f, 5f)
        verticalLineToRelative(14f)
        curveToRelative(0f, 1.1f, 0.9f, 2f, 2f, 2f)
        horizontalLineToRelative(14f)
        curveToRelative(1.1f, 0f, 2f, -0.9f, 2f, -2f)
        verticalLineTo(5f)
        curveTo(21f, 3.9f, 20.1f, 3f, 19f, 3f)
        close()
        moveTo(17f, 17f)
        horizontalLineTo(7f)
        verticalLineToRelative(-2f)
        horizontalLineToRelative(10f)
        verticalLineToRelative(2f)
        close()
        moveTo(17f, 13f)
        horizontalLineTo(7f)
        verticalLineToRelative(-2f)
        horizontalLineToRelative(10f)
        verticalLineToRelative(2f)
        close()
        moveTo(17f, 9f)
        horizontalLineTo(7f)
        verticalLineTo(7f)
        horizontalLineToRelative(10f)
        verticalLineToRelative(2f)
        close()
    }

    /** Bottom-nav ギャラリー: a framed image. */
    val Image: ImageVector = filled("Image") {
        moveTo(21f, 19f)
        verticalLineTo(5f)
        curveToRelative(0f, -1.1f, -0.9f, -2f, -2f, -2f)
        horizontalLineTo(5f)
        curveTo(3.9f, 3f, 3f, 3.9f, 3f, 5f)
        verticalLineToRelative(14f)
        curveToRelative(0f, 1.1f, 0.9f, 2f, 2f, 2f)
        horizontalLineToRelative(14f)
        curveToRelative(1.1f, 0f, 2f, -0.9f, 2f, -2f)
        close()
        moveTo(8.5f, 13.5f)
        lineToRelative(2.5f, 3.01f)
        lineTo(14.5f, 12f)
        lineToRelative(4.5f, 6f)
        horizontalLineTo(5f)
        lineToRelative(3.5f, -4.5f)
        close()
    }

    /** Settings 通知 row / notification: a bell. */
    val Bell: ImageVector = filled("Bell") {
        moveTo(12f, 22f)
        curveToRelative(1.1f, 0f, 2f, -0.9f, 2f, -2f)
        horizontalLineToRelative(-4f)
        curveToRelative(0f, 1.1f, 0.89f, 2f, 2f, 2f)
        close()
        moveTo(18f, 16f)
        verticalLineToRelative(-5f)
        curveToRelative(0f, -3.07f, -1.63f, -5.64f, -4.5f, -6.32f)
        verticalLineTo(4f)
        curveToRelative(0f, -0.83f, -0.67f, -1.5f, -1.5f, -1.5f)
        reflectiveCurveToRelative(-1.5f, 0.67f, -1.5f, 1.5f)
        verticalLineToRelative(0.68f)
        curveTo(7.64f, 5.36f, 6f, 7.92f, 6f, 11f)
        verticalLineToRelative(5f)
        lineToRelative(-2f, 2f)
        verticalLineToRelative(1f)
        horizontalLineToRelative(16f)
        verticalLineToRelative(-1f)
        lineToRelative(-2f, -2f)
        close()
    }

    /** Settings プライバシー・ロック row: a padlock. */
    val Lock: ImageVector = filled("Lock") {
        moveTo(18f, 8f)
        horizontalLineToRelative(-1f)
        verticalLineTo(6f)
        curveToRelative(0f, -2.76f, -2.24f, -5f, -5f, -5f)
        reflectiveCurveTo(7f, 3.24f, 7f, 6f)
        verticalLineToRelative(2f)
        horizontalLineTo(6f)
        curveToRelative(-1.1f, 0f, -2f, 0.9f, -2f, 2f)
        verticalLineToRelative(10f)
        curveToRelative(0f, 1.1f, 0.9f, 2f, 2f, 2f)
        horizontalLineToRelative(12f)
        curveToRelative(1.1f, 0f, 2f, -0.9f, 2f, -2f)
        verticalLineTo(10f)
        curveToRelative(0f, -1.1f, -0.9f, -2f, -2f, -2f)
        close()
        moveTo(9f, 6f)
        curveToRelative(0f, -1.66f, 1.34f, -3f, 3f, -3f)
        reflectiveCurveToRelative(3f, 1.34f, 3f, 3f)
        verticalLineToRelative(2f)
        horizontalLineTo(9f)
        verticalLineTo(6f)
        close()
        moveTo(12f, 17f)
        curveToRelative(-1.1f, 0f, -2f, -0.9f, -2f, -2f)
        reflectiveCurveToRelative(0.9f, -2f, 2f, -2f)
        reflectiveCurveToRelative(2f, 0.9f, 2f, 2f)
        reflectiveCurveToRelative(-0.9f, 2f, -2f, 2f)
        close()
    }

    /** Settings アカウント・データ row: a person. */
    val Person: ImageVector = filled("Person") {
        moveTo(12f, 12f)
        curveToRelative(2.21f, 0f, 4f, -1.79f, 4f, -4f)
        reflectiveCurveToRelative(-1.79f, -4f, -4f, -4f)
        reflectiveCurveTo(8f, 5.79f, 8f, 8f)
        reflectiveCurveToRelative(1.79f, 4f, 4f, 4f)
        close()
        moveTo(12f, 14f)
        curveToRelative(-2.67f, 0f, -8f, 1.34f, -8f, 4f)
        verticalLineToRelative(2f)
        horizontalLineToRelative(16f)
        verticalLineToRelative(-2f)
        curveToRelative(0f, -2.66f, -5.33f, -4f, -8f, -4f)
        close()
    }

    /** Home top-bar: settings gear. */
    val Settings: ImageVector = filled("Settings") {
        moveTo(19.14f, 12.94f)
        curveToRelative(0.04f, -0.3f, 0.06f, -0.61f, 0.06f, -0.94f)
        curveToRelative(0f, -0.32f, -0.02f, -0.64f, -0.07f, -0.94f)
        lineToRelative(2.03f, -1.58f)
        curveToRelative(0.18f, -0.14f, 0.23f, -0.41f, 0.12f, -0.61f)
        lineToRelative(-1.92f, -3.32f)
        curveToRelative(-0.12f, -0.22f, -0.37f, -0.29f, -0.59f, -0.22f)
        lineToRelative(-2.39f, 0.96f)
        curveToRelative(-0.5f, -0.38f, -1.03f, -0.7f, -1.62f, -0.94f)
        lineToRelative(-0.36f, -2.54f)
        curveToRelative(-0.04f, -0.24f, -0.24f, -0.41f, -0.48f, -0.41f)
        horizontalLineToRelative(-3.84f)
        curveToRelative(-0.24f, 0f, -0.43f, 0.17f, -0.47f, 0.41f)
        lineToRelative(-0.36f, 2.54f)
        curveToRelative(-0.59f, 0.24f, -1.13f, 0.57f, -1.62f, 0.94f)
        lineToRelative(-2.39f, -0.96f)
        curveToRelative(-0.22f, -0.08f, -0.47f, 0f, -0.59f, 0.22f)
        lineTo(2.74f, 8.87f)
        curveToRelative(-0.12f, 0.21f, -0.08f, 0.47f, 0.12f, 0.61f)
        lineToRelative(2.03f, 1.58f)
        curveToRelative(-0.05f, 0.3f, -0.09f, 0.63f, -0.09f, 0.94f)
        reflectiveCurveToRelative(0.02f, 0.64f, 0.07f, 0.94f)
        lineToRelative(-2.03f, 1.58f)
        curveToRelative(-0.18f, 0.14f, -0.23f, 0.41f, -0.12f, 0.61f)
        lineToRelative(1.92f, 3.32f)
        curveToRelative(0.12f, 0.22f, 0.37f, 0.29f, 0.59f, 0.22f)
        lineToRelative(2.39f, -0.96f)
        curveToRelative(0.5f, 0.38f, 1.03f, 0.7f, 1.62f, 0.94f)
        lineToRelative(0.36f, 2.54f)
        curveToRelative(0.05f, 0.24f, 0.24f, 0.41f, 0.48f, 0.41f)
        horizontalLineToRelative(3.84f)
        curveToRelative(0.24f, 0f, 0.44f, -0.17f, 0.47f, -0.41f)
        lineToRelative(0.36f, -2.54f)
        curveToRelative(0.59f, -0.24f, 1.13f, -0.56f, 1.62f, -0.94f)
        lineToRelative(2.39f, 0.96f)
        curveToRelative(0.22f, 0.08f, 0.47f, 0f, 0.59f, -0.22f)
        lineToRelative(1.92f, -3.32f)
        curveToRelative(0.12f, -0.22f, 0.07f, -0.47f, -0.12f, -0.61f)
        lineToRelative(-2.01f, -1.58f)
        close()
        moveTo(12f, 15.6f)
        curveToRelative(-1.98f, 0f, -3.6f, -1.62f, -3.6f, -3.6f)
        reflectiveCurveToRelative(1.62f, -3.6f, 3.6f, -3.6f)
        reflectiveCurveToRelative(3.6f, 1.62f, 3.6f, 3.6f)
        reflectiveCurveToRelative(-1.62f, 3.6f, -3.6f, 3.6f)
        close()
    }

    /** Account エクスポート row: an upload / export arrow. */
    val Export: ImageVector = filled("Export") {
        moveTo(9f, 16f)
        horizontalLineToRelative(6f)
        verticalLineToRelative(-6f)
        horizontalLineToRelative(4f)
        lineToRelative(-7f, -7f)
        lineToRelative(-7f, 7f)
        horizontalLineToRelative(4f)
        close()
        moveTo(5f, 18f)
        horizontalLineToRelative(14f)
        verticalLineToRelative(2f)
        horizontalLineTo(5f)
        close()
    }

    /** Account サインアウト row: a logout arrow. */
    val SignOut: ImageVector = filled("SignOut") {
        moveTo(17f, 7f)
        lineToRelative(-1.41f, 1.41f)
        lineTo(18.17f, 11f)
        horizontalLineTo(8f)
        verticalLineToRelative(2f)
        horizontalLineToRelative(10.17f)
        lineToRelative(-2.58f, 2.58f)
        lineTo(17f, 17f)
        lineToRelative(5f, -5f)
        close()
        moveTo(4f, 5f)
        horizontalLineToRelative(8f)
        verticalLineTo(3f)
        horizontalLineTo(4f)
        curveTo(2.9f, 3f, 2f, 3.9f, 2f, 5f)
        verticalLineToRelative(14f)
        curveToRelative(0f, 1.1f, 0.9f, 2f, 2f, 2f)
        horizontalLineToRelative(8f)
        verticalLineToRelative(-2f)
        horizontalLineTo(4f)
        close()
    }

    /** Trailing chevron for settings rows. */
    val ChevronRight: ImageVector = filled("ChevronRight") {
        moveTo(10f, 6f)
        lineTo(8.59f, 7.41f)
        lineTo(13.17f, 12f)
        lineToRelative(-4.58f, 4.59f)
        lineTo(10f, 18f)
        lineToRelative(6f, -6f)
        close()
    }

    private fun filled(name: String, block: androidx.compose.ui.graphics.vector.PathBuilder.() -> Unit): ImageVector =
        ImageVector.Builder(
            name = name,
            defaultWidth = 24.dp,
            defaultHeight = 24.dp,
            viewportWidth = 24f,
            viewportHeight = 24f,
        ).apply {
            path(fill = SolidColor(Color.Black), pathBuilder = block)
        }.build()
}
