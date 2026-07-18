package com.runa.shared.feature.insight

import com.runa.shared.feature.today.moon.MoonPhaseKey

/**
 * Turns an [InsightSummary] into the quiet read-back text the insight screen shows.
 *
 * This is the seam the feature is designed to swap: today it is the local,
 * offline [RuleBasedSummaryComposer]; a future server-side LLM summariser would be
 * a different implementation injected behind [InsightRepository], with the VM and
 * both UIs unchanged. `compose` is `suspend` precisely so a network-backed
 * implementation fits without touching this interface.
 */
interface SummaryComposer {
    suspend fun compose(summary: InsightSummary): InsightNarrative
}

/**
 * The two text blocks the insight screen draws: the main letter [body] and the
 * quiet [footnote] card beneath the chart. The fixed heading ("あなたへの、手紙")
 * and the period label live in the UI/VM, not here.
 */
data class InsightNarrative(
    val body: String,
    val footnote: String?,
)

/**
 * Rule-based, offline summary composition — templates with simple conditional
 * branches over the aggregated facts. The tone is a still映し返し: it states what
 * happened and never evaluates, diagnoses, advises, or encourages. No claim beyond
 * the numbers is made. (The one design-fixed case — a month with many nights whose
 * words gathered as the moon grew full — reproduces `design/16_insight.png` verbatim;
 * the other branches are drafts, tunable in this one file.)
 */
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

    /** "言葉が多かったのは、月が満ちてゆく頃。" — only when there's enough to notice a peak. */
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

    /** Phases at or growing toward full — the moon is "満ちてゆく". Full is included. */
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
