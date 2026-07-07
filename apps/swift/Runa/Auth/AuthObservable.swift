import Foundation
import Shared

/// ObservableObject bridge over the shared `AuthViewModel`.
///
/// SKIE bridges the Kotlin `StateFlow<AuthState>` to a `SkieSwiftStateFlow` (an
/// `AsyncSequence`); we collect it and republish each emission on the main actor
/// as a `@Published` value. Action methods forward straight to the shared view
/// model — they only enqueue coroutines, touch no `@Published` state, and are safe
/// to call from any context, so SwiftUI closures can invoke them directly.
final class AuthObservable: ObservableObject {
    /// Latest auth state. `nil` before the first emission (treated as restoring).
    @Published private(set) var state: AuthState?

    private let viewModel: AuthViewModel
    private var collectTask: Task<Void, Never>?

    // `resolveAuthViewModel()` is a top-level Kotlin fun in shared/di/Koin.kt,
    // exported by SKIE as this global Swift func. It pulls the shared VM from the
    // Koin graph (the same instance the health check uses).
    init(viewModel: AuthViewModel = resolveAuthViewModel()) {
        self.viewModel = viewModel
        collectTask = Task { [weak self] in
            guard let self else { return }
            let stateFlow: SkieSwiftStateFlow<AuthState> = self.viewModel.state
            for await value in stateFlow {
                await MainActor.run { self.state = value }
            }
        }
    }

    func loginEmail(email: String, password: String) {
        viewModel.loginEmail(email: email, password: password)
    }

    func signupEmail(email: String, password: String, displayName: String?) {
        viewModel.signupEmail(email: email, password: password, displayName: displayName)
    }

    func loginApple(idToken: String, displayName: String?) {
        viewModel.loginApple(idToken: idToken, displayName: displayName)
    }

    func loginGoogle(idToken: String) {
        viewModel.loginGoogle(idToken: idToken)
    }

    func logout() {
        viewModel.logout()
    }

    func clearError() {
        viewModel.clearError()
    }

    deinit {
        collectTask?.cancel()
    }
}
