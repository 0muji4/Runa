import SwiftUI

/// 設定 — reached from the Home tab's top bar. Still mostly a shell, but it now
/// hosts sign-out, which flips the shared auth state back to unauthenticated and
/// returns the whole app to the sign-in flow.
/// TODO: real settings content (account, notifications, biometric lock, ...).
struct SettingsView: View {
    let onSignOut: () -> Void

    var body: some View {
        ScreenScaffold(
            title: "設定",
            placeholder: "設定項目がここに表示されます。"
        ) {
            Button(action: onSignOut) {
                Text("サインアウト")
                    .font(RunaFonts.body(16))
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, RunaSpacing.sm)
                    .background(RunaColors.surface)
                    .foregroundStyle(RunaColors.accent)
                    .clipShape(RoundedRectangle(cornerRadius: 12))
            }
            .padding(.top, RunaSpacing.md)
        }
        .navigationTitle("設定")
        .navigationBarTitleDisplayMode(.inline)
    }
}

#Preview {
    NavigationStack {
        SettingsView(onSignOut: {})
    }
    .preferredColorScheme(.dark)
}
