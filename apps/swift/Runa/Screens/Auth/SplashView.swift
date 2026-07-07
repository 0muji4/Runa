import SwiftUI

/// Quiet startup splash, shown while the shared auth state is `Restoring` (the
/// app is checking the stored session). Moon glyph over the near-black surface.
struct SplashView: View {
    var body: some View {
        ZStack {
            RunaColors.background.ignoresSafeArea()

            VStack(spacing: RunaSpacing.md) {
                Text("◐")
                    .font(.system(size: 56))
                    .foregroundStyle(RunaColors.accent)
                Text("Runa")
                    .font(RunaFonts.logo(40))
                    .foregroundStyle(RunaColors.heading)
                Text("月あかりの記録")
                    .font(RunaFonts.body(14))
                    .foregroundStyle(RunaColors.subtle)
                ProgressView()
                    .tint(RunaColors.accent)
                    .padding(.top, RunaSpacing.sm)
            }
        }
    }
}

#Preview {
    SplashView().preferredColorScheme(.dark)
}
