import SwiftUI
import Shared

/// プライバシー・ロック (22). The confirmed design's パスコード / Face ID / すぐにロック
/// controls are simplified to one ON/OFF toggle (the agreed spec-minimal model):
/// when on, the app requires Face ID / Touch ID — with the device passcode as
/// fallback — on launch/resume. A quiet notice appears when the device has no
/// security set up (enabling the lock would then be ineffective).
struct PrivacyLockView: View {
    @Environment(\.runaTheme) private var runaTheme
    @StateObject private var obs = AppLockObservable()

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            Text("PRIVACY")
                .font(RunaFonts.body(13)).tracking(3)
                .foregroundStyle(runaTheme.subtle)
                .padding(.top, RunaSpacing.md)
            Text("ロック")
                .font(RunaFonts.heading(40))
                .foregroundStyle(runaTheme.heading)
                .padding(.top, RunaSpacing.xs)

            LockEmblem()
                .frame(maxWidth: .infinity)
                .padding(.top, RunaSpacing.xl)

            Text("あなたの夜を、あなただけに。")
                .font(RunaFonts.heading(18))
                .foregroundStyle(runaTheme.body)
                .frame(maxWidth: .infinity)
                .multilineTextAlignment(.center)
                .padding(.top, RunaSpacing.lg)

            Toggle(isOn: Binding(
                get: { obs.lockEnabled },
                set: { obs.setLockEnabled($0) }
            )) {
                Text("ロックする")
                    .font(RunaFonts.body(17))
                    .foregroundStyle(runaTheme.heading)
            }
            .tint(runaTheme.accent)
            .padding(.top, RunaSpacing.xl)

            if !obs.biometricAvailable() {
                Text("この端末では、生体認証や画面ロックが設定されていません。端末の設定をご確認ください。")
                    .font(RunaFonts.body(13))
                    .foregroundStyle(runaTheme.subtle)
                    .padding(.top, RunaSpacing.sm)
            }

            Spacer()
        }
        .padding(.horizontal, 28)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .background(runaTheme.background)
        .navigationTitle("")
        .navigationBarTitleDisplayMode(.inline)
    }
}
