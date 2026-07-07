import SwiftUI
import Shared

/// The unauthenticated flow: onboarding ①② → notification shell → sign-in. Shown
/// by the root gate whenever the shared `AuthState` is not `Authenticated`.
/// Actions delegate to the shared `AuthObservable`; provider-side errors (Apple
/// cancelled, Google unconfigured, ...) are surfaced locally alongside the shared
/// `AuthState.Error`.
struct AuthFlowView: View {
    @ObservedObject var auth: AuthObservable
    let state: AuthState

    private enum Step { case onboarding1, onboarding2, notifications, signIn }
    @State private var step: Step = .onboarding1
    @State private var localError: String?
    @State private var google = GoogleWebSignIn()

    var body: some View {
        switch step {
        case .onboarding1:
            OnboardingView(
                glyph: "☾",
                title: "しずかに、記す。",
                message: "一日のおわりに、月へ言葉を預ける場所。",
                primaryLabel: "次へ",
                onPrimary: { step = .onboarding2 }
            )
        case .onboarding2:
            OnboardingView(
                glyph: "☽",
                title: "あなたの夜を、そっと。",
                message: "飾らず、余白を大切に。続けられる日記を。",
                primaryLabel: "次へ",
                onPrimary: { step = .notifications }
            )
        case .notifications:
            OnboardingView(
                glyph: "🔔",
                title: "通知の許可",
                message: "やさしい時刻に、そっとお知らせします。設定はあとからでも変えられます。",
                primaryLabel: "次へ",
                onPrimary: { step = .signIn },
                secondaryLabel: "スキップ",
                onSecondary: { step = .signIn }
            )
        case .signIn:
            SignInView(
                isBusy: isBusy,
                errorMessage: localError ?? stateError,
                onApple: { idToken, name in
                    localError = nil
                    auth.clearError()
                    auth.loginApple(idToken: idToken, displayName: name)
                },
                onAppleError: { message in localError = message },
                onGoogle: {
                    localError = nil
                    auth.clearError()
                    google.signIn(
                        onIdToken: { token in auth.loginGoogle(idToken: token) },
                        onError: { message in localError = message }
                    )
                },
                onEmailSubmit: { isSignup, email, password, name in
                    localError = nil
                    auth.clearError()
                    if isSignup {
                        auth.signupEmail(email: email, password: password, displayName: name.isEmpty ? nil : name)
                    } else {
                        auth.loginEmail(email: email, password: password)
                    }
                }
            )
        }
    }

    private var isBusy: Bool {
        if case .authenticating = onEnum(of: state) { return true }
        return false
    }

    private var stateError: String? {
        if case .error(let error) = onEnum(of: state) { return error.message }
        return nil
    }
}
