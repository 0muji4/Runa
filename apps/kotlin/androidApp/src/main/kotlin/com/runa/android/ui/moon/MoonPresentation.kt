package com.runa.android.ui.moon

import com.runa.shared.feature.today.moon.MoonPhaseKey
import com.runa.shared.feature.today.moon.moonPhaseGlyph
import com.runa.shared.feature.today.moon.moonPhaseNameJa

/** Thin adapter over the shared moon glyph/name functions. */
object MoonPresentation {
    fun glyph(key: MoonPhaseKey): String = moonPhaseGlyph(key)
    fun name(key: MoonPhaseKey): String = moonPhaseNameJa(key)
}
