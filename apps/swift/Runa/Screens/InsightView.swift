import SwiftUI
import Shared

/// 16 インサイト — "あなたへの、手紙". A quiet retrospective letter: the period label, a
/// 明朝 heading, the rule-based summary, then the moon-phase overlap histogram (the
/// hero, a lone pink peak) and a soft mood-dot line, closed by a still footnote card.
/// A minimal 週/月 toggle and ‹ › period nav sit above. Everything renders from the
/// local diary — no network. Matches the Android InsightScreen so both OSes agree.
struct InsightView: View {
    @StateObject private var model = InsightObservable()
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        ZStack(alignment: .top) {
            RunaColors.background.ignoresSafeArea()
            ScrollView {
                VStack(alignment: .leading, spacing: 0) {
                    Button { dismiss() } label: {
                        Text("‹ 戻る").font(RunaFonts.body(13)).foregroundStyle(RunaColors.subtle)
                    }
                    .padding(.top, 14)
                    .padding(.vertical, 6)

                    content
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.horizontal, 28)
            }
            .scrollIndicators(.hidden)
        }
        .toolbar(.hidden, for: .navigationBar)
    }

    @ViewBuilder private var content: some View {
        if let state = model.state {
            switch onEnum(of: state) {
            case .content(let c):
                periodBar(label: c.periodLabel, type: c.periodType)
                letter(c.insight)
                banner(c.banner)
                Spacer().frame(height: 40)
            case .empty(let e):
                periodBar(label: e.periodLabel, type: e.periodType)
                emptyLetter()
                banner(e.banner)
            case .loading:
                Color.clear.frame(height: 240)
            case .error:
                errorLetter()
            }
        } else {
            Color.clear.frame(height: 240)
        }
    }

    // MARK: period controls

    private func periodBar(label: String, type: InsightPeriodType) -> some View {
        VStack(spacing: 0) {
            HStack(spacing: 10) {
                toggleChip("週", selected: isWeekly(type)) { model.showWeekly() }
                toggleChip("月", selected: !isWeekly(type)) { model.showMonthly() }
            }
            .frame(maxWidth: .infinity)
            .padding(.top, 12)

            HStack {
                Button { model.showPrevious() } label: {
                    Text("‹").font(RunaFonts.logo(34)).foregroundStyle(RunaColors.subtle)
                }
                Spacer()
                Button { model.showCurrent() } label: {
                    Text(label).font(RunaFonts.heading(15)).foregroundStyle(RunaColors.subtle)
                }
                .buttonStyle(.plain)
                Spacer()
                Button { model.showNext() } label: {
                    Text("›").font(RunaFonts.logo(34)).foregroundStyle(RunaColors.subtle)
                }
            }
            .padding(.top, 20)
        }
    }

    private func toggleChip(_ label: String, selected: Bool, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Text(label)
                .font(RunaFonts.heading(15))
                .foregroundStyle(selected ? RunaColors.accent : RunaColors.subtle)
                .padding(.horizontal, 20)
                .padding(.vertical, 7)
                .overlay {
                    if selected {
                        RoundedRectangle(cornerRadius: 16)
                            .stroke(RunaColors.accent.opacity(0.7), lineWidth: 1)
                    }
                }
        }
        .buttonStyle(.plain)
    }

    private func isWeekly(_ type: InsightPeriodType) -> Bool {
        switch type {
        case .weekly: return true
        default: return false
        }
    }

    // MARK: the letter

    private func letter(_ insight: Insight) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            Text("あなたへの、手紙")
                .font(RunaFonts.heading(32))
                .foregroundStyle(RunaColors.heading)
                .padding(.top, 16)

            Text(insight.narrative.body)
                .font(RunaFonts.heading(18))
                .foregroundStyle(RunaColors.body)
                .lineSpacing(10)
                .fixedSize(horizontal: false, vertical: true)
                .padding(.top, 28)

            moonOverlapChart(insight.summary.moonOverlap)
                .padding(.top, 40)

            moodDots(insight.summary.moodDistribution, unmooded: Int(insight.summary.unmoodedCount))
                .padding(.top, 36)

            if let footnote = insight.narrative.footnote {
                Text(footnote)
                    .font(RunaFonts.heading(17))
                    .foregroundStyle(RunaColors.body)
                    .lineSpacing(9)
                    .fixedSize(horizontal: false, vertical: true)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(26)
                    .background(RunaColors.surface)
                    .clipShape(RoundedRectangle(cornerRadius: 20))
                    .padding(.top, 40)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    /// The hero histogram: entries bucketed across the lunar cycle (新月 → 満月 → 新月).
    /// The busiest phase glows moonlight-pink; the rest stay muted.
    private func moonOverlapChart(_ buckets: [MoonPhaseBucket]) -> some View {
        let counts = buckets.map { Int($0.count) }
        let maxCount = counts.max() ?? 0
        let peakIndex = maxCount > 0 ? counts.firstIndex(of: maxCount) : nil
        return VStack(spacing: 10) {
            HStack(alignment: .bottom, spacing: 8) {
                ForEach(Array(buckets.enumerated()), id: \.offset) { index, bucket in
                    let fraction = maxCount > 0 ? Double(bucket.count) / Double(maxCount) : 0
                    RoundedRectangle(cornerRadius: 6)
                        .fill(peakIndex == index ? RunaColors.accent : RunaColors.subtle.opacity(0.28))
                        .frame(maxWidth: .infinity)
                        .frame(height: max(120 * fraction, 8))
                }
            }
            .frame(height: 120, alignment: .bottom)

            HStack {
                Text("新月").font(RunaFonts.body(12)).foregroundStyle(RunaColors.subtle)
                Spacer()
                Text("満月").font(RunaFonts.body(12)).foregroundStyle(RunaColors.subtle)
                Spacer()
                Text("新月").font(RunaFonts.body(12)).foregroundStyle(RunaColors.subtle)
            }
        }
    }

    /// The soft mood line: a few dots per recorded mood, and a quiet note for the unmarked nights.
    private func moodDots(_ distribution: [MoodCount], unmooded: Int) -> some View {
        let present = distribution.filter { $0.count > 0 }
        return VStack(alignment: .leading, spacing: 12) {
            ForEach(Array(present.enumerated()), id: \.offset) { _, moodCount in
                HStack(spacing: 6) {
                    Text(diaryMoodLabelJa(mood: moodCount.mood))
                        .font(RunaFonts.heading(14))
                        .foregroundStyle(RunaColors.body)
                        .frame(width: 64, alignment: .leading)
                    ForEach(0 ..< min(Int(moodCount.count), 12), id: \.self) { _ in
                        Circle().fill(RunaColors.subAccent).frame(width: 7, height: 7)
                    }
                }
            }
            if unmooded > 0 {
                Text("しるしのない夜も、\(unmooded)。")
                    .font(RunaFonts.heading(13))
                    .foregroundStyle(RunaColors.subtle)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    // MARK: empty / error / banner

    private func emptyLetter() -> some View {
        VStack(spacing: 0) {
            NewMoonEmblem(diameter: 116)
            Text("まだ、しるした夜がありません。")
                .font(RunaFonts.heading(22))
                .foregroundStyle(RunaColors.heading)
                .multilineTextAlignment(.center)
                .padding(.top, 28)
            Text("この期間は、静かなままです。")
                .font(RunaFonts.body(14))
                .foregroundStyle(RunaColors.subtle)
                .multilineTextAlignment(.center)
                .padding(.top, 14)
        }
        .frame(maxWidth: .infinity)
        .padding(.top, 64)
    }

    private func errorLetter() -> some View {
        VStack {
            Text("同期に、少しつまずいています。")
                .font(RunaFonts.body(14))
                .foregroundStyle(RunaColors.subtle)
                .multilineTextAlignment(.center)
        }
        .frame(maxWidth: .infinity)
        .padding(.top, 80)
    }

    @ViewBuilder private func banner(_ banner: InsightBanner) -> some View {
        if let text = bannerText(banner) {
            Text(text)
                .font(RunaFonts.body(13))
                .foregroundStyle(RunaColors.subtle)
                .frame(maxWidth: .infinity)
                .padding(.top, 24)
        }
    }

    private func bannerText(_ banner: InsightBanner) -> String? {
        switch banner {
        case .offline: return "オフライン。綴った言葉は、端末に守られています。"
        case .error: return "同期に、少しつまずいています。"
        default: return nil // .none / .syncing stay silent
        }
    }
}
