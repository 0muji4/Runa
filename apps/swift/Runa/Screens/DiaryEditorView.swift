import SwiftUI
import Shared

/// Diary editor (screen 10) — "書く". A whitespace-first, Shippori Mincho canvas
/// (Dynamic Type–aware) where the character count is never shown. Autosave is
/// durable — the entry persists from the first line — with a deliberately quiet
/// indicator. Mood is an optional, still choice.
struct DiaryEditorView: View {
    @StateObject private var model: DiaryEditorObservable

    init(clientId: String?) {
        _model = StateObject(wrappedValue: DiaryEditorObservable(clientId: clientId))
    }

    var body: some View {
        ZStack {
            RunaColors.background.ignoresSafeArea()

            VStack(alignment: .leading, spacing: RunaSpacing.md) {
                ZStack(alignment: .topLeading) {
                    if (model.state?.bodyText ?? "").isEmpty {
                        Text("今日のことを、そっと。")
                            .font(RunaFonts.heading(18, relativeTo: .body))
                            .foregroundStyle(RunaColors.subtle)
                            .padding(.top, 8)
                            .padding(.leading, 5)
                    }
                    TextEditor(text: bodyBinding)
                        .font(RunaFonts.heading(18, relativeTo: .body))
                        .foregroundColor(RunaColors.body)
                        .scrollContentBackground(.hidden)
                        .background(RunaColors.background)
                }
                .frame(maxHeight: .infinity)

                moodRow
            }
            .padding(.horizontal, RunaSpacing.md)
            .padding(.top, RunaSpacing.sm)
        }
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Text(saveLabel(model.state?.save))
                    .font(RunaFonts.body(13))
                    .foregroundStyle(RunaColors.subtle)
            }
        }
        .onDisappear { model.saveNow() }
    }

    private var bodyBinding: Binding<String> {
        Binding(
            get: { model.state?.bodyText ?? "" },
            set: { model.onBodyChange($0) }
        )
    }

    private var moodRow: some View {
        VStack(alignment: .leading, spacing: RunaSpacing.xs) {
            Text("きょうの心もち（任意）")
                .font(RunaFonts.body(13))
                .foregroundStyle(RunaColors.subtle)
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 8) {
                    ForEach(DiaryMood.allCases, id: \.self) { mood in
                        let selected = model.state?.mood == mood.rawValue
                        Button {
                            model.onMoodChange(selected ? nil : mood.rawValue)
                        } label: {
                            Text(mood.label)
                                .font(RunaFonts.body(13))
                                .padding(.horizontal, 14)
                                .padding(.vertical, 8)
                                .background(selected ? RunaColors.accent.opacity(0.18) : RunaColors.surface)
                                .foregroundStyle(selected ? RunaColors.accent : RunaColors.subtle)
                                .clipShape(Capsule())
                        }
                    }
                }
            }
        }
        .padding(.bottom, RunaSpacing.sm)
    }

    private func saveLabel(_ status: SaveStatus?) -> String {
        switch status {
        case .saving: return "保存中…"
        case .saved: return "保存済み"
        case .error: return "保存に失敗"
        default: return "下書き" // .editing / nil (pre-load)
        }
    }
}
