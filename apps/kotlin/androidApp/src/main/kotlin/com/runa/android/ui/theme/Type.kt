package com.runa.android.ui.theme

import androidx.compose.material3.Typography
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp

/**
 * Runa typography.
 *
 * The design system calls for three families:
 *   - headings: "Shippori Mincho"
 *   - body:     "Zen Kaku Gothic New"
 *   - logo:     "Cormorant Garamond"
 *
 * The .ttf/.otf binaries can NOT be committed here, so we currently fall back to
 * [FontFamily.Default] for every family. This keeps the build green while making
 * the intended structure explicit.
 *
 * TO ENABLE THE REAL FONTS:
 *   1. Drop the font files into androidApp/src/main/res/font/ using the exact
 *      lowercase filenames listed in androidApp/FONTS.md (Android resource names
 *      must be lowercase, no spaces).
 *   2. Uncomment the FontFamily blocks below and swap the `= FontFamily.Default`
 *      assignments for them.
 *
 * // import androidx.compose.ui.text.font.Font
 * // import com.runa.android.R
 * // val ShipporiMincho = FontFamily(
 * //     Font(R.font.shippori_mincho_regular, FontWeight.Normal),
 * //     Font(R.font.shippori_mincho_bold, FontWeight.Bold),
 * // )
 * // val ZenKakuGothicNew = FontFamily(
 * //     Font(R.font.zen_kaku_gothic_new_regular, FontWeight.Normal),
 * //     Font(R.font.zen_kaku_gothic_new_medium, FontWeight.Medium),
 * // )
 * // val CormorantGaramond = FontFamily(
 * //     Font(R.font.cormorant_garamond_regular, FontWeight.Normal),
 * // )
 */
val ShipporiMincho: FontFamily = FontFamily.Default
val ZenKakuGothicNew: FontFamily = FontFamily.Default
val CormorantGaramond: FontFamily = FontFamily.Default

/**
 * Maps the families onto Material3 type roles:
 *   - display roles  -> logo (Cormorant Garamond)
 *   - headline/title -> headings (Shippori Mincho)
 *   - body/label     -> body (Zen Kaku Gothic New)
 */
val RunaTypography = Typography(
    displayLarge = TextStyle(
        fontFamily = CormorantGaramond,
        fontWeight = FontWeight.Normal,
        fontSize = 40.sp,
        lineHeight = 48.sp,
    ),
    headlineMedium = TextStyle(
        fontFamily = ShipporiMincho,
        fontWeight = FontWeight.Normal,
        fontSize = 26.sp,
        lineHeight = 34.sp,
    ),
    titleLarge = TextStyle(
        fontFamily = ShipporiMincho,
        fontWeight = FontWeight.Normal,
        fontSize = 22.sp,
        lineHeight = 30.sp,
    ),
    bodyLarge = TextStyle(
        fontFamily = ZenKakuGothicNew,
        fontWeight = FontWeight.Normal,
        fontSize = 16.sp,
        lineHeight = 24.sp,
    ),
    bodyMedium = TextStyle(
        fontFamily = ZenKakuGothicNew,
        fontWeight = FontWeight.Normal,
        fontSize = 14.sp,
        lineHeight = 22.sp,
    ),
    labelLarge = TextStyle(
        fontFamily = ZenKakuGothicNew,
        fontWeight = FontWeight.Medium,
        fontSize = 13.sp,
        lineHeight = 18.sp,
    ),
)
