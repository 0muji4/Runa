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
                title: "あなたの夜に、\nそっと寄り添う。",
                onNext: { step = .onboarding2 }
            )
        case .onboarding2:
            OnboardingView(
                title: "三つの、\n静かな時間。",
                onNext: { step = .notifications }
            )
        case .notifications:
            NotificationPermissionView(
                onContinue: { step = .signIn },
                onSkip: { step = .signIn }
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
                },
                // No anonymous/guest session exists yet, so "いまはスキップ" gently
                // steps back to the intro rather than bypassing the auth gate.
                onSkip: {
                    localError = nil
                    auth.clearError()
                    step = .onboarding1
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
