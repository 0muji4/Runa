import SwiftUI
import Shared

/// Diary editor (10) — "書く". A whitespace-first 明朝 canvas: the day's date, a
/// quiet prompt, then the writing surface (Dynamic Type–aware). The character count
/// is never shown; autosave is durable with a whisper of an indicator. "とじる"
/// flushes and leaves. A quiet mood chip row sits under the prompt — one gentle word
/// for the night, feeding the insight read-back; leaving it unset is natural.
struct DiaryEditorView: View {
    @Environment(\.runaTheme) private var runaTheme
    @StateObject private var model: DiaryEditorObservable
    @Environment(\.dismiss) private var dismiss

    // The day being written; the header shows it. For a backdated entry (calendar
    // "write on this day") it is that day's local noon.
    private let dayMs: Int64

    init(clientId: String?) {
        _model = StateObject(wrappedValue: DiaryEditorObservable(clientId: clientId))
        dayMs = Int64(Date().timeIntervalSince1970 * 1000)
    }

    /// Backdated new entry for a calendar day (ISO yyyy-MM-dd).
    init(backdateIsoDate: String) {
        let epoch = DiaryEditorView.noonEpochMs(isoDate: backdateIsoDate)
        _model = StateObject(wrappedValue: DiaryEditorObservable(backdateEpochMs: epoch))
        dayMs = epoch
    }

    /// Epoch-millis of local noon on an ISO yyyy-MM-dd day — a stable mid-day instant
    /// that never slips across the date boundary.
    private static func noonEpochMs(isoDate: String) -> Int64 {
        var cal = Calendar(identifier: .gregorian)
        cal.timeZone = .current
        let parser = DateFormatter()
        parser.locale = Locale(identifier: "en_US_POSIX")
        parser.timeZone = .current
        parser.dateFormat = "yyyy-MM-dd"
        let day = parser.date(from: isoDate) ?? Date()
        var comps = cal.dateComponents([.year, .month, .day], from: day)
        comps.hour = 12
        comps.minute = 0
        let noon = cal.date(from: comps) ?? day
        return Int64(noon.timeIntervalSince1970 * 1000)
    }

    var body: some View {
        ZStack {
            runaTheme.background.ignoresSafeArea()

            VStack(alignment: .leading, spacing: 0) {
                Text("\(DiaryDate.day(dayMs))　\(DiaryDate.weekday(dayMs))")
                    .font(RunaFonts.body(13))
                    .foregroundStyle(runaTheme.subtle)

                Text("きょう、心にのこったこと。")
                    .font(RunaFonts.heading(20, relativeTo: .body))
                    .foregroundStyle(runaTheme.subtle)
                    .padding(.top, RunaSpacing.md)
                    .padding(.bottom, RunaSpacing.sm)

                moodChips(model.state?.mood)
                    .padding(.bottom, RunaSpacing.sm)

                TextEditor(text: bodyBinding)
                    .font(RunaFonts.heading(18, relativeTo: .body))
                    .foregroundColor(runaTheme.body)
                    .lineSpacing(8)
                    .scrollContentBackground(.hidden)
                    .background(runaTheme.background)
                    .frame(maxHeight: .infinity)

                HStack(alignment: .center) {
                    Text(saveLabel(model.state?.save))
                        .font(RunaFonts.body(13))
                        .foregroundStyle(runaTheme.subtle)
                    Spacer()
                    Button {
                        model.saveNow()
                        dismiss()
                    } label: {
                        Text("とじる")
                            .font(RunaFonts.body(16))
                            .foregroundStyle(runaTheme.accent)
                            .padding(.horizontal, 28)
                            .padding(.vertical, 12)
                            .overlay(
                                RoundedRectangle(cornerRadius: 24)
                                    .stroke(runaTheme.accent.opacity(0.7), lineWidth: 1)
                            )
                    }
                }
                .padding(.top, RunaSpacing.sm)
            }
            .padding(.horizontal, RunaSpacing.md)
            .padding(.top, RunaSpacing.sm)
            .padding(.bottom, RunaSpacing.sm)
        }
        .toolbar(.hidden, for: .navigationBar)
        .onDisappear { model.saveNow() }
    }

    /// The quiet mood row: one gentle word for the night. Tapping the selected chip
    /// again clears it (未選択). Options come from the shared `DiaryMood`, so what's
    /// written is exactly what the insight aggregation reads.
    private func moodChips(_ selected: String?) -> some View {
        // fixedSize(vertical:) keeps the horizontal scroll strip hugging its content
        // height, so it never competes with the greedy TextEditor below for space.
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                ForEach(diaryMoods(), id: \.self) { mood in
                    let value = diaryMoodValue(mood: mood)
                    let isSelected = value == selected
                    Text(diaryMoodLabelJa(mood: mood))
                        .font(RunaFonts.body(13))
                        .foregroundStyle(isSelected ? runaTheme.accent : runaTheme.subtle)
                        .padding(.horizontal, 14)
                        .padding(.vertical, 7)
                        .background(
                            RoundedRectangle(cornerRadius: 16)
                                .fill(isSelected ? runaTheme.accent.opacity(0.10) : Color.clear)
                        )
                        .overlay(
                            RoundedRectangle(cornerRadius: 16)
                                .stroke(isSelected ? runaTheme.accent.opacity(0.7) : runaTheme.subtle.opacity(0.3), lineWidth: 1)
                        )
                        .onTapGesture { model.onMoodChange(isSelected ? nil : value) }
                }
            }
        }
        .fixedSize(horizontal: false, vertical: true)
    }

    private var bodyBinding: Binding<String> {
        Binding(
            get: { model.state?.bodyText ?? "" },
            set: { model.onBodyChange($0) }
        )
    }

    private func saveLabel(_ status: SaveStatus?) -> String {
        switch status {
        case .saving: return "保存しています…"
        case .saved: return "保存しました"
        case .error: return "保存に、つまずきました"
        default: return "静かに、綴る" // .editing / nil (pre-load)
        }
    }
}
