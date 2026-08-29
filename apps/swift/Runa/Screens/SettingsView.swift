import SwiftUI

/// 設定 トップ (19). A quiet list of entry points: theme, notification and
/// privacy-lock (the latter two are 導線 only for a later feature), then
/// account・データ, the LUNA+ card and the app version. Sign-out lives on the
/// account screen (23) per the confirmed design. Pushed onto Home's NavigationStack.
struct SettingsView: View {
    let onSignOut: () -> Void

    @Environment(\.runaTheme) private var runaTheme
    @StateObject private var theme = ThemeObservable()
    @StateObject private var notification = NotificationSettingsObservable()
    @StateObject private var lock = AppLockObservable()

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                Text(L.settingsTitle)
                    .font(RunaFonts.heading(40))
                    .foregroundStyle(runaTheme.heading)
                    .padding(.top, RunaSpacing.md)
                    .padding(.bottom, RunaSpacing.lg)

                NavigationLink { ThemeView() } label: {
                    SettingRow(icon: "moon", label: L.settingsRowTheme, value: themeName(theme.themeId))
                }
                divider
                NavigationLink { NotificationSettingsView() } label: {
                    SettingRow(
                        icon: "bell",
                        label: L.settingsRowNotifications,
                        value: notification.enabled ? notification.time.label : L.notifValueOff
                    )
                }
                divider
                NavigationLink { PrivacyLockView() } label: {
                    SettingRow(
                        icon: "lock",
                        label: L.settingsRowPrivacyLock,
                        value: lock.lockEnabled ? L.lockValueOn : L.lockValueOff
                    )
                }
                divider
                NavigationLink { AccountView(onSignOut: onSignOut) } label: {
                    SettingRow(icon: "person", label: L.settingsRowAccount)
                }

                premiumCard
                    .padding(.top, RunaSpacing.lg)

                Text(L.settingsVersion(appVersion))
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
        .toolbar(.hidden, for: .tabBar)
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
                Text(L.settingsPremiumTitle).font(RunaFonts.heading(22)).foregroundStyle(runaTheme.heading)
                Text(L.settingsPremiumSubtitle).font(RunaFonts.body(13)).foregroundStyle(runaTheme.subtle)
            }
            Spacer()
            Image(systemName: "chevron.right").foregroundStyle(runaTheme.accent)
        }
        .padding(20)
        .background(runaTheme.surface, in: RoundedRectangle(cornerRadius: 20))
    }

    private var appVersion: String {
        (Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String) ?? "1.0"
    }

    private func themeName(_ id: String) -> String {
        switch id {
        case "light": return L.themeLightName
        case "pink": return L.themePinkName
        default: return L.themeDarkName
        }
    }
}

/// One quiet settings row: leading SF Symbol, label, optional trailing value, chevron.
private struct SettingRow: View {
    @Environment(\.runaTheme) private var runaTheme
    let icon: String
    let label: String
    var value: String? = nil
    var enabled: Bool = true

    var body: some View {
        let labelColor = enabled ? runaTheme.heading : runaTheme.subtle
        HStack {
            Image(systemName: icon).font(.system(size: 18)).foregroundStyle(labelColor).frame(width: 36, alignment: .leading)
            Text(label).font(RunaFonts.body(17)).foregroundStyle(labelColor)
            Spacer()
            if let value {
                Text(value).font(RunaFonts.body(14)).foregroundStyle(runaTheme.subtle)
            }
            Image(systemName: "chevron.right").font(.system(size: 14)).foregroundStyle(runaTheme.subtle)
        }
        .padding(.vertical, 20)
        .contentShape(Rectangle())
    }
}

#Preview {
    NavigationStack { SettingsView(onSignOut: {}) }
}
