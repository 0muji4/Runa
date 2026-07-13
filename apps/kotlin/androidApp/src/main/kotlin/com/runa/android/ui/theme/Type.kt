package com.runa.android.ui.theme

import androidx.compose.material3.Typography
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.Font
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp
import com.runa.android.R

/**
 * Runa typography. The design system calls for three families:
 *   - headings: "Shippori Mincho"       (明朝 — long-form and titles)
 *   - body:     "Zen Kaku Gothic New"   (本文)
 *   - logo:     "Cormorant Garamond"    (ロゴ「LUNA」)
 *
 * The OFL binaries live in androidApp/src/main/res/font/ (see androidApp/FONTS.md).
 * Only Regular + Medium are bundled per family; emphasis is carried by size and
 * colour, not heavy weights — in keeping with the quiet, spare design tone.
 */
val ShipporiMincho: FontFamily = FontFamily(
    Font(R.font.shippori_mincho_regular, FontWeight.Normal),
    Font(R.font.shippori_mincho_medium, FontWeight.Medium),
)
val ZenKakuGothicNew: FontFamily = FontFamily(
    Font(R.font.zen_kaku_gothic_new_regular, FontWeight.Normal),
    Font(R.font.zen_kaku_gothic_new_medium, FontWeight.Medium),
)
val CormorantGaramond: FontFamily = FontFamily(
    Font(R.font.cormorant_garamond_regular, FontWeight.Normal),
    Font(R.font.cormorant_garamond_medium, FontWeight.Medium),
)

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
