package com.runa.shared.feature.insight

import com.runa.shared.feature.diary.DiaryMood
import com.runa.shared.feature.today.moon.MoonPhaseKey
import kotlinx.coroutines.test.runTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertNull
import kotlin.test.assertTrue

/**
 * Rule-based summary: the design-fixed case reproduced verbatim, the branch
 * variations, and a tone guardrail asserting the text never advises or diagnoses.
 */
class SummaryComposerTest {

    @Test
    fun reproducesTheConfirmedMonthlyLetter() = runTest {
        // design/16_insight.png: a month of 18 nights whose words gathered toward full moon.
        val summary = summary(
            type = InsightPeriodType.Monthly,
            days = 18,
            entryCount = 18,
            moon = peakAt(MoonPhaseKey.FULL_MOON, 6),
        )
        val n = RuleBasedSummaryComposer.compose(summary)

        assertEquals(
            "この一か月、あなたは\n18の夜を綴りました。\n言葉が多かったのは、\n月が満ちてゆく頃。",
            n.body,
        )
        assertEquals("いちばん静かだった夜に、\nいちばん深い言葉がありました。", n.footnote)
    }

    @Test
    fun waningPeakReadsAsTheMoonEbbing() = runTest {
        val n = RuleBasedSummaryComposer.compose(
            summary(type = InsightPeriodType.Monthly, days = 12, entryCount = 12, moon = peakAt(MoonPhaseKey.LAST_QUARTER, 5)),
        )
        assertTrue(n.body.contains("月が欠けてゆく頃"), n.body)
    }

    @Test
    fun weeklyUsesTheWeekPhrasing() = runTest {
        val n = RuleBasedSummaryComposer.compose(
            summary(type = InsightPeriodType.Weekly, days = 4, entryCount = 4, moon = peakAt(MoonPhaseKey.WAXING_GIBBOUS, 2)),
        )
        assertTrue(n.body.startsWith("この一週間、あなたは"), n.body)
    }

    @Test
    fun aSingleNightIsNamedGentlyWithoutAMoonClaim() = runTest {
        val n = RuleBasedSummaryComposer.compose(
            summary(type = InsightPeriodType.Monthly, days = 1, entryCount = 1, moon = peakAt(MoonPhaseKey.FULL_MOON, 1)),
        )
        assertEquals("この一か月、あなたは\nひとつの夜を綴りました。", n.body)
        assertFalse(n.body.contains("言葉が多かった"), "too few nights to claim a peak")
        assertEquals("ひとつの夜が、\nここに残りました。", n.footnote)
    }

    @Test
    fun anEmptyPeriodStatesItPlainly() = runTest {
        val n = RuleBasedSummaryComposer.compose(summary(days = 0, entryCount = 0))
        assertEquals("この一か月、\nしるされた夜は、まだありません。", n.body)
        assertNull(n.footnote)
    }

    @Test
    fun neverAdvisesDiagnosesOrEncourages() = runTest {
        // A still映し返し only — none of these evaluative / prescriptive tokens may appear.
        val forbidden = listOf(
            "がんばって", "頑張", "大丈夫", "べき", "しましょう", "しよう",
            "診断", "アドバイス", "おすすめ", "改善", "治", "病", "気をつけ",
        )
        val matrix = buildList {
            for (type in InsightPeriodType.entries) {
                for (days in listOf(0, 1, 3, 7, 18, 31)) {
                    for (most in listOf(null, DiaryMood.Calm, DiaryMood.Heavy, DiaryMood.Tired)) {
                        for (peak in MoonPhaseKey.entries) {
                            add(summary(type = type, days = days, entryCount = days, most = most, moon = peakAt(peak, days)))
                        }
                    }
                }
            }
        }
        for (s in matrix) {
            val n = RuleBasedSummaryComposer.compose(s)
            val text = n.body + "\n" + (n.footnote ?: "")
            for (word in forbidden) {
                assertFalse(text.contains(word), "summary must not advise/diagnose but contains '$word':\n$text")
            }
        }
    }

    // ---- helpers ----

    private fun peakAt(peak: MoonPhaseKey, count: Int): List<MoonPhaseBucket> =
        MoonPhaseKey.entries.map { MoonPhaseBucket(it, if (it == peak) count else 0) }

    private fun summary(
        type: InsightPeriodType = InsightPeriodType.Monthly,
        days: Int = 0,
        entryCount: Int = days,
        unmooded: Int = 0,
        most: DiaryMood? = null,
        streak: Int = days,
        moon: List<MoonPhaseBucket> = MoonPhaseKey.entries.map { MoonPhaseBucket(it, 0) },
    ) = InsightSummary(
        periodType = type,
        daysJournaled = days,
        entryCount = entryCount,
        unmoodedCount = unmooded,
        moodDistribution = DiaryMood.entries.map { MoodCount(it, 0) },
        mostFrequentMood = most,
        longestStreak = streak,
        moonOverlap = moon,
    )
}
