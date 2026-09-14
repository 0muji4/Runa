import SwiftUI
import Shared

/// Privacy-lock gate, separate from the auth gate. While locked the real `content` is NOT
/// built, so nothing private can flash behind the lock screen. Foregrounds the view model
/// on `.active` and, at launch, via `.onAppear`.
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

/// The lock screen: an unlock affordance, or a waiting note while the biometric prompt is up.
private struct LockScreen: View {
    @Environment(\.runaTheme) private var runaTheme
    let authenticating: Bool
    let onUnlock: () -> Void

    var body: some View {
        VStack(spacing: 0) {
            Spacer()
            GlowingMoon(diameter: 132)
            Text(L.lockGateMessage)
                .font(RunaFonts.heading(22))
                .foregroundStyle(runaTheme.heading)
                .multilineTextAlignment(.center)
                .padding(.top, RunaSpacing.lg)
            if authenticating {
                Text(L.lockGateAuthenticating)
                    .font(RunaFonts.body(14))
                    .foregroundStyle(runaTheme.subtle)
                    .padding(.top, RunaSpacing.xl)
            } else {
                Text(L.lockGateUnlock)
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
