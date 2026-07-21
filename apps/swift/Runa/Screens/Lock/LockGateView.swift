import SwiftUI
import Shared

/// Privacy-lock gate — a layer SEPARATE from the auth gate. While the lock is
/// engaged the real `content` is NOT built (nothing private can flash behind the
/// lock); a quiet moon-motif lock screen with an unlock affordance is shown
/// instead. When the lock is off (or auth succeeds, or the device has no security)
/// the content renders normally.
///
/// Drives the shared view model from the scene lifecycle: foreground on `.active`
/// (and at launch, via `.onAppear`), background on `.background`.
struct LockGateView<Content: View>: View {
    @ObservedObject var lock: AppLockObservable
    @Environment(\.scenePhase) private var scenePhase
    @ViewBuilder var content: () -> Content

    var body: some View {
        Group {
            switch onEnum(of: lock.state) {
            case .unlocked, .unavailable:
                content()
            case .locked:
                LockScreen(authenticating: false) { lock.authenticate() }
            case .authenticating:
                LockScreen(authenticating: true) {}
            }
        }
        .onAppear { lock.onAppForegrounded() }
        .onChange(of: scenePhase) { phase in
            switch phase {
            case .active: lock.onAppForegrounded()
            case .background: lock.onAppBackgrounded()
            default: break
            }
        }
    }
}

/// The quiet lock screen: a glowing moon, a poetic line, and an unlock affordance
/// (or a "確認しています…" note while the biometric prompt is up).
private struct LockScreen: View {
    @Environment(\.runaTheme) private var runaTheme
    let authenticating: Bool
    let onUnlock: () -> Void

    var body: some View {
        VStack(spacing: 0) {
            Spacer()
            GlowingMoon(diameter: 132)
            Text("あなたの夜は、\n守られています。")
                .font(RunaFonts.heading(22))
                .foregroundStyle(runaTheme.heading)
                .multilineTextAlignment(.center)
                .padding(.top, RunaSpacing.lg)
            if authenticating {
                Text("確認しています…")
                    .font(RunaFonts.body(14))
                    .foregroundStyle(runaTheme.subtle)
                    .padding(.top, RunaSpacing.xl)
            } else {
                Text("ひらく")
                    .font(RunaFonts.body(16)).tracking(4)
                    .foregroundStyle(runaTheme.accent)
                    .padding(.horizontal, 40)
                    .padding(.vertical, 16)
                    .background(runaTheme.surface, in: RoundedRectangle(cornerRadius: 16))
                    .padding(.top, RunaSpacing.xl)
                    .contentShape(Rectangle())
                    .onTapGesture(perform: onUnlock)
            }
            Spacer()
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(runaTheme.background.ignoresSafeArea())
    }
}
