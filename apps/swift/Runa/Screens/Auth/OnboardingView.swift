import SwiftUI

/// Onboarding shell (screens ①②) + the notification-permission shell (④). All
/// intentionally spare — a glyph, a heading, a line of copy, and a button — and
/// wired straight into the sign-in flow. Real content/permissions land later.
struct OnboardingView: View {
    let glyph: String
    let title: String
    let message: String
    let primaryLabel: String
    let onPrimary: () -> Void
    /// Optional secondary action (used for the notification "skip").
    var secondaryLabel: String? = nil
    var onSecondary: (() -> Void)? = nil

    var body: some View {
        ZStack {
            RunaColors.background.ignoresSafeArea()

            VStack(spacing: RunaSpacing.md) {
                Spacer()
                Text(glyph)
                    .font(.system(size: 48))
                    .foregroundStyle(RunaColors.accent)
                Text(title)
                    .font(RunaFonts.heading(26))
                    .foregroundStyle(RunaColors.heading)
                    .multilineTextAlignment(.center)
                Text(message)
                    .font(RunaFonts.body(15))
                    .foregroundStyle(RunaColors.subtle)
                    .multilineTextAlignment(.center)
                Spacer()
                Button(action: onPrimary) {
                    Text(primaryLabel)
                        .font(RunaFonts.body(16))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, RunaSpacing.sm)
                        .background(RunaColors.accent)
                        .foregroundStyle(RunaColors.background)
                        .clipShape(RoundedRectangle(cornerRadius: 12))
                }
                if let secondaryLabel, let onSecondary {
                    Button(action: onSecondary) {
                        Text(secondaryLabel)
                            .font(RunaFonts.body(14))
                            .foregroundStyle(RunaColors.subtle)
                    }
                }
            }
            .padding(.horizontal, RunaSpacing.md)
            .padding(.vertical, RunaSpacing.lg)
        }
    }
}

#Preview {
    OnboardingView(
        glyph: "☾",
        title: "しずかに、記す。",
        message: "一日のおわりに、月へ言葉を預ける場所。",
        primaryLabel: "次へ",
        onPrimary: {}
    )
    .preferredColorScheme(.dark)
}
