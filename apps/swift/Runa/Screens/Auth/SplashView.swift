import SwiftUI

/// Quiet startup splash, shown while the shared auth state is `Restoring` (the
/// app is checking the stored session). The glowing moon over the LUNA wordmark.
struct SplashView: View {
    @Environment(\.runaTheme) private var runaTheme
    var body: some View {
        ZStack {
            runaTheme.background.ignoresSafeArea()

            VStack(spacing: 0) {
                GlowingMoon(diameter: 152)
                Text("LUNA")
                    .font(RunaFonts.logo(44))
                    .tracking(14)
                    .foregroundStyle(runaTheme.heading)
                    .padding(.top, RunaSpacing.md)
                Text("月あかりの記録")
                    .font(RunaFonts.body(14))
                    .foregroundStyle(runaTheme.subtle)
                    .padding(.top, RunaSpacing.xs)
                ProgressView()
                    .tint(runaTheme.accent)
                    .padding(.top, RunaSpacing.lg)
            }
        }
    }
}

#Preview {
    SplashView().preferredColorScheme(.dark)
}
