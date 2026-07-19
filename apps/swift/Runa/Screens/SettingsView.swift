import SwiftUI

/// 設定 トップ (19). A quiet list of entry points: theme, notification and
/// privacy-lock (the latter two are 導線 only for a later feature), then
/// account・データ, the LUNA+ card and the app version. Sign-out lives on the
/// account screen (23) per the confirmed design. Pushed onto Home's NavigationStack.
struct SettingsView: View {
    let onSignOut: () -> Void

    @Environment(\.runaTheme) private var runaTheme
    @StateObject private var theme = ThemeObservable()

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                Text("設定")
                    .font(RunaFonts.heading(40))
                    .foregroundStyle(runaTheme.heading)
                    .padding(.top, RunaSpacing.md)
                    .padding(.bottom, RunaSpacing.lg)

                NavigationLink { ThemeView() } label: {
                    SettingRow(glyph: "☾", label: "テーマ", value: themeName(theme.themeId))
                }
                divider
                SettingRow(glyph: "◷", label: "通知", value: "準備中", enabled: false)
                divider
                SettingRow(glyph: "⚿", label: "プライバシー・ロック", value: "準備中", enabled: false)
                divider
                NavigationLink { AccountView(onSignOut: onSignOut) } label: {
                    SettingRow(glyph: "◍", label: "アカウント・データ")
                }

                premiumCard
                    .padding(.top, RunaSpacing.lg)

                Text("LUNA version \(appVersion)")
                    .font(RunaFonts.body(12))
                    .foregroundStyle(runaTheme.subtle)
                    .frame(maxWidth: .infinity)
                    .padding(.top, RunaSpacing.lg)
                    .padding(.bottom, RunaSpacing.md)
            }
            .padding(.horizontal, 28)
        }
        .background(runaTheme.background)
        .navigationTitle("")
        .navigationBarTitleDisplayMode(.inline)
    }

    private var divider: some View {
        Rectangle()
            .fill(runaTheme.subtle.opacity(0.15))
            .frame(height: 1)
    }

    private var premiumCard: some View {
        // LUNA+ 導線. The paywall is a separate, not-yet-built feature — a quiet card.
        HStack(spacing: RunaSpacing.sm) {
            Circle().fill(runaTheme.subAccent).frame(width: 56, height: 56)
            VStack(alignment: .leading, spacing: 4) {
                Text("LUNA +").font(RunaFonts.heading(22)).foregroundStyle(runaTheme.heading)
                Text("プレミアムで、もっと深く").font(RunaFonts.body(13)).foregroundStyle(runaTheme.subtle)
            }
            Spacer()
            Text("›").foregroundStyle(runaTheme.accent)
        }
        .padding(20)
        .background(runaTheme.surface, in: RoundedRectangle(cornerRadius: 20))
    }

    private var appVersion: String {
        (Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String) ?? "1.0"
    }

    private func themeName(_ id: String) -> String {
        switch id {
        case "light": return "あさ（ライト）"
        case "pink": return "ピンク×ピンク"
        default: return "夜（ダーク）"
        }
    }
}

/// One quiet settings row: leading glyph, label, optional trailing value, chevron.
private struct SettingRow: View {
    @Environment(\.runaTheme) private var runaTheme
    let glyph: String
    let label: String
    var value: String? = nil
    var enabled: Bool = true

    var body: some View {
        let labelColor = enabled ? runaTheme.heading : runaTheme.subtle
        HStack {
            Text(glyph).font(RunaFonts.body(18)).foregroundStyle(labelColor).frame(width: 36, alignment: .leading)
            Text(label).font(RunaFonts.body(17)).foregroundStyle(labelColor)
            Spacer()
            if let value {
                Text(value).font(RunaFonts.body(14)).foregroundStyle(runaTheme.subtle)
            }
            Text("›").font(RunaFonts.body(20)).foregroundStyle(runaTheme.subtle)
        }
        .padding(.vertical, 20)
        .contentShape(Rectangle())
    }
}

#Preview {
    NavigationStack { SettingsView(onSignOut: {}) }
}
