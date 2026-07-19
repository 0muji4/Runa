import SwiftUI

/// テーマ (20). The three app themes as selectable cards; the active one is bordered
/// with a filled radio. Selecting one drives the shared view model, which persists it
/// and re-emits — the environment palette swaps and the whole app (this screen
/// included) recolors in place, giving the live preview the design calls for while
/// keeping the user on this screen.
struct ThemeView: View {
    @Environment(\.runaTheme) private var runaTheme
    @StateObject private var theme = ThemeObservable()

    private struct Option: Identifiable {
        let id: String
        let name: String
        let desc: String
    }

    private let options = [
        Option(id: "dark", name: "夜（ダーク）", desc: "深い夜色に、月あかり"),
        Option(id: "light", name: "あさ（ライト）", desc: "朝の光のような、白"),
        Option(id: "pink", name: "ピンク×ピンク", desc: "ルナピンクの、灯り"),
    ]

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            Text("THEME")
                .font(RunaFonts.body(13)).tracking(3)
                .foregroundStyle(runaTheme.subtle)
                .padding(.top, RunaSpacing.md)
            Text("テーマ")
                .font(RunaFonts.heading(40))
                .foregroundStyle(runaTheme.heading)
                .padding(.top, RunaSpacing.xs)
                .padding(.bottom, RunaSpacing.lg)

            ForEach(options) { option in
                card(option)
                    .padding(.bottom, RunaSpacing.sm)
            }

            Spacer()
            Text("あなたの夜に、あわせて。")
                .font(RunaFonts.body(14))
                .foregroundStyle(runaTheme.subtle)
                .frame(maxWidth: .infinity)
                .padding(.bottom, RunaSpacing.md)
        }
        .padding(.horizontal, 28)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .background(runaTheme.background)
        .navigationTitle("")
        .navigationBarTitleDisplayMode(.inline)
    }

    private func card(_ option: Option) -> some View {
        let selected = option.id == theme.themeId
        let palette = RunaTheme.forId(option.id)
        return Button {
            theme.select(option.id)
        } label: {
            HStack(spacing: RunaSpacing.sm) {
                swatch(palette)
                VStack(alignment: .leading, spacing: 4) {
                    Text(option.name).font(RunaFonts.heading(20)).foregroundStyle(runaTheme.heading)
                    Text(option.desc).font(RunaFonts.body(13)).foregroundStyle(runaTheme.subtle)
                }
                Spacer()
                radio(selected)
            }
            .padding(18)
            .background(runaTheme.surface, in: RoundedRectangle(cornerRadius: 20))
            .overlay(
                RoundedRectangle(cornerRadius: 20)
                    .stroke(selected ? runaTheme.accent : .clear, lineWidth: 1)
            )
        }
        .buttonStyle(.plain)
    }

    /// A miniature preview of a theme: its background with an accent + sub-accent dot.
    private func swatch(_ palette: RunaTheme) -> some View {
        ZStack {
            RoundedRectangle(cornerRadius: 14).fill(palette.background)
            Circle().fill(palette.subAccent).frame(width: 14, height: 14)
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading).padding(12)
            Circle().fill(palette.accent).frame(width: 10, height: 10)
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .bottomTrailing).padding(12)
        }
        .frame(width: 56, height: 56)
    }

    private func radio(_ selected: Bool) -> some View {
        ZStack {
            Circle().stroke(selected ? runaTheme.accent : runaTheme.subtle, lineWidth: 1)
            if selected {
                Circle().fill(runaTheme.accent).frame(width: 12, height: 12)
            }
        }
        .frame(width: 24, height: 24)
    }
}

#Preview {
    NavigationStack { ThemeView() }
}
