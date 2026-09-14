package com.runa.shared.feature.todaymoon

import com.runa.shared.feature.today.moon.MoonPhaseKey

/** The phase-specific line shown on 今日の月; the "\n" is the intended line break. */
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
