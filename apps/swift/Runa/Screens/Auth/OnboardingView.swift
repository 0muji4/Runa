import SwiftUI
import UserNotifications

/// Onboarding (①②). Whitespace-first and spare, exactly as the design intends: a
/// softly glowing moon, one large left-aligned 明朝 line, and a quiet "すすむ" to
/// advance — no filled buttons, no body paragraph.
struct OnboardingView: View {
    @Environment(\.runaTheme) private var runaTheme
    let title: String
    let onNext: () -> Void

    var body: some View {
        ZStack {
            runaTheme.background.ignoresSafeArea()

            VStack(alignment: .leading, spacing: 0) {
                GlowingMoon(diameter: 116)
                    .padding(.top, RunaSpacing.md)
                Spacer()
                Text(title)
                    .font(RunaFonts.heading(32))
                    .lineSpacing(14)
                    .foregroundStyle(runaTheme.heading)
                Spacer()
                Text(L.onboardingHint)
                    .font(RunaFonts.body(13))
                    .tracking(6)
                    .foregroundStyle(runaTheme.subtle)
                    .frame(maxWidth: .infinity)
                    .padding(16)
                    .contentShape(Rectangle())
                    .onTapGesture(perform: onNext)
            }
            .padding(.horizontal, RunaSpacing.lg)
            .padding(.vertical, RunaSpacing.xl)
        }
    }
}

/// Notification permission (④). A quiet night-time request: the moon with a small
/// moonlight-pink bell badge, a poetic 明朝 line, and a gentle ask.
struct NotificationPermissionView: View {
    @Environment(\.runaTheme) private var runaTheme
    let onContinue: () -> Void
    let onSkip: () -> Void

    var body: some View {
        ZStack {
            runaTheme.background.ignoresSafeArea()

            VStack(spacing: 0) {
                Spacer()
                NotificationMoon(diameter: 156)
                Text(L.notifTitle)
                    .font(RunaFonts.heading(26))
                    .foregroundStyle(runaTheme.heading)
                    .multilineTextAlignment(.center)
                    .padding(.top, RunaSpacing.lg)
                Text(L.notifBody)
                    .font(RunaFonts.body(15))
                    .foregroundStyle(runaTheme.subtle)
                    .multilineTextAlignment(.center)
                    .padding(.top, RunaSpacing.sm)

                Button(action: requestAuthorization) {
                    Text(L.notifAllow)
                        .font(RunaFonts.body(16))
                        .frame(maxWidth: .infinity)
                        .frame(height: 56)
                        .background(runaTheme.accent)
                        .foregroundStyle(runaTheme.background)
                        .clipShape(RoundedRectangle(cornerRadius: 16))
                }
                .padding(.top, RunaSpacing.lg)

                Text(L.actionSkip)
                    .font(RunaFonts.body(13))
                    .tracking(4)
                    .foregroundStyle(runaTheme.subtle)
                    .padding(12)
                    .padding(.top, RunaSpacing.xs)
                    .onTapGesture(perform: onSkip)
                Spacer()
            }
            .padding(.horizontal, RunaSpacing.lg)
        }
    }

    /// Fire the real POST-notification authorization request, then advance whether
    /// granted or denied — a denial must never break onboarding (DoD#3).
    private func requestAuthorization() {
        UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .sound]) { _, _ in
            DispatchQueue.main.async { onContinue() }
        }
    }
}

#Preview {
    OnboardingView(title: L.onboarding2Title, onNext: {})
        .preferredColorScheme(.dark)
}
