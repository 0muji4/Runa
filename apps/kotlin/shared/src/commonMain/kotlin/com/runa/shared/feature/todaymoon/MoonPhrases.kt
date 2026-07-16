package com.runa.shared.feature.todaymoon

import com.runa.shared.feature.today.moon.MoonPhaseKey

/**
 * The quiet, phase-specific line shown on 15 今日の月. Held in shared as a fixed
 * table (one line per [MoonPhaseKey]) so Android and iOS show the exact same words
 * with no network. Each is a two-line 明朝 phrase in Runa's hushed register; the
 * "\n" is the intended line break.
 *
 * The 満月 line is the one fixed by the confirmed design (design/15_todays_moon.png);
 * the others are drafts kept here so the whole table can be tuned in one place.
 */
object MoonPhrases {
    fun phraseFor(key: MoonPhaseKey): String = when (key) {
        MoonPhaseKey.NEW_MOON -> "はじまりの闇に、\nそっと願いを。"
        MoonPhaseKey.WAXING_CRESCENT -> "細い光にも、\n芽ぶくものがある。"
        MoonPhaseKey.FIRST_QUARTER -> "半分の光で、\n選んでいく。"
        MoonPhaseKey.WAXING_GIBBOUS -> "満ちてゆく夜は、\nもう少しだけ。"
        MoonPhaseKey.FULL_MOON -> "満ちた月は、\n手ばなすための夜。"
        MoonPhaseKey.WANING_GIBBOUS -> "こぼれる光を、\nゆっくり返す。"
        MoonPhaseKey.LAST_QUARTER -> "欠けてゆく半分に、\n整えるしずけさ。"
        MoonPhaseKey.WANING_CRESCENT -> "消えゆく光は、\n次の闇へのしるべ。"
    }
}
