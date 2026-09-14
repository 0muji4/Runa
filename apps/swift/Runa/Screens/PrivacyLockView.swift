import SwiftUI
import Shared

/// プライバシー・ロック: one ON/OFF toggle for the biometric lock (device passcode as fallback).
struct PrivacyLockView: View {
    @Environment(\.runaTheme) private var runaTheme
    @Environment(\.dismiss) private var dismiss
    @StateObject private var obs = AppLockObservable()

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            RunaScreenHeader(title: L.lockSettingsTitle, onBack: { dismiss() })

            LockEmblem()
                .frame(maxWidth: .infinity)

            Text(L.lockSettingsCaption)
                .font(RunaFonts.heading(18))
                .foregroundStyle(runaTheme.body)
                .frame(maxWidth: .infinity)
                .multilineTextAlignment(.center)
                .padding(.top, RunaSpacing.lg)

            Toggle(isOn: Binding(
                get: { obs.lockEnabled },
                set: { obs.setLockEnabled($0) }
            )) {
                Text(L.lockSettingsToggle)
                    .font(RunaFonts.body(17))
                    .foregroundStyle(runaTheme.heading)
            }
            .tint(runaTheme.accent)
            .padding(.top, RunaSpacing.xl)

            if !obs.biometricAvailable() {
                Text(L.lockSettingsUnavailable)
                    .font(RunaFonts.body(13))
                    .foregroundStyle(runaTheme.subtle)
                    .padding(.top, RunaSpacing.sm)
            }

            Spacer()
        }
        .padding(.horizontal, 28)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .background(runaTheme.background)
        .toolbar(.hidden, for: .navigationBar)
        .toolbar(.hidden, for: .tabBar)
    }
}
