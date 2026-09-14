package com.runa.shared.feature.insight

import com.runa.shared.feature.today.moon.MoonPhaseKey

/** Turns an [InsightSummary] into the read-back text; `suspend` so a network-backed implementation fits. */
interface SummaryComposer {
    suspend fun compose(summary: InsightSummary): InsightNarrative
}

/** The letter [body] and the [footnote] card beneath the chart; heading and period label live in the UI. */
data class InsightNarrative(
    val body: String,
    val footnote: String?,
)

/** Rule-based, offline composition: states what happened, never evaluates or advises. */
object RuleBasedSummaryComposer : SummaryComposer {

    override suspend fun compose(summary: InsightSummary): InsightNarrative =
        InsightNarrative(body = body(summary), footnote = footnote(summary))

    private fun body(summary: InsightSummary): String {
        if (summary.daysJournaled == 0) {
            return "${periodPhrase(summary)}、\nしるされた夜は、まだありません。"
        }
        val nights = if (summary.daysJournaled == 1) "ひとつの夜" else "${summary.daysJournaled}の夜"
        val opening = "${periodPhrase(summary)}、あなたは\n${nights}を綴りました。"
        val moon = moonLine(summary)
        return if (moon != null) "$opening\n$moon" else opening
    }

    /** Only when there is enough to notice a peak. */
    private fun moonLine(summary: InsightSummary): String? {
        if (summary.daysJournaled < 3) return null
        val peak = summary.moonOverlap.maxByOrNull { it.count } ?: return null
        if (peak.count == 0) return null
        val whenPhrase = if (isTowardFull(peak.phaseKey)) "月が満ちてゆく頃" else "月が欠けてゆく頃"
        return "言葉が多かったのは、\n$whenPhrase。"
    }

    private fun footnote(summary: InsightSummary): String? = when {
        summary.daysJournaled >= 10 -> "いちばん静かだった夜に、\nいちばん深い言葉がありました。"
        summary.daysJournaled >= 3 -> "みじかい言葉も、\nちゃんと夜をかたどっています。"
        summary.daysJournaled >= 1 -> "ひとつの夜が、\nここに残りました。"
        else -> null
    }

    private fun periodPhrase(summary: InsightSummary): String = when (summary.periodType) {
        InsightPeriodType.Weekly -> "この一週間"
        InsightPeriodType.Monthly -> "この一か月"
    }

    /** Waxing phases; full is included. */
    private fun isTowardFull(key: MoonPhaseKey): Boolean = when (key) {
        MoonPhaseKey.WAXING_CRESCENT,
        MoonPhaseKey.FIRST_QUARTER,
        MoonPhaseKey.WAXING_GIBBOUS,
        MoonPhaseKey.FULL_MOON -> true
        MoonPhaseKey.WANING_GIBBOUS,
        MoonPhaseKey.LAST_QUARTER,
        MoonPhaseKey.WANING_CRESCENT,
        MoonPhaseKey.NEW_MOON -> false
    }
}
