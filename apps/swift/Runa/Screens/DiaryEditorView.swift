import SwiftUI
import Shared

/// Diary editor (10) — "書く". A whitespace-first 明朝 canvas: the day's date, a
/// quiet prompt, then the writing surface (Dynamic Type–aware). The character count
/// is never shown; autosave is durable with a whisper of an indicator. "とじる"
/// flushes and leaves. Mood is intentionally absent, matching the confirmed design.
struct DiaryEditorView: View {
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
            RunaColors.background.ignoresSafeArea()

            VStack(alignment: .leading, spacing: 0) {
                Text("\(DiaryDate.day(dayMs))　\(DiaryDate.weekday(dayMs))")
                    .font(RunaFonts.body(13))
                    .foregroundStyle(RunaColors.subtle)

                Text("きょう、心にのこったこと。")
                    .font(RunaFonts.heading(20, relativeTo: .body))
                    .foregroundStyle(RunaColors.subtle)
                    .padding(.top, RunaSpacing.md)
                    .padding(.bottom, RunaSpacing.sm)

                TextEditor(text: bodyBinding)
                    .font(RunaFonts.heading(18, relativeTo: .body))
                    .foregroundColor(RunaColors.body)
                    .lineSpacing(8)
                    .scrollContentBackground(.hidden)
                    .background(RunaColors.background)
                    .frame(maxHeight: .infinity)

                HStack(alignment: .center) {
                    Text(saveLabel(model.state?.save))
                        .font(RunaFonts.body(13))
                        .foregroundStyle(RunaColors.subtle)
                    Spacer()
                    Button {
                        model.saveNow()
                        dismiss()
                    } label: {
                        Text("とじる")
                            .font(RunaFonts.body(16))
                            .foregroundStyle(RunaColors.accent)
                            .padding(.horizontal, 28)
                            .padding(.vertical, 12)
                            .overlay(
                                RoundedRectangle(cornerRadius: 24)
                                    .stroke(RunaColors.accent.opacity(0.7), lineWidth: 1)
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
