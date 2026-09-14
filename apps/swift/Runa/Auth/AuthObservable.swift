import Foundation
import Shared

/// ObservableObject bridge over the shared `AuthViewModel`.
/// Action methods only enqueue coroutines and touch no `@Published` state; call from any context.
final class AuthObservable: ObservableObject {
    /// Latest auth state. `nil` before the first emission (treated as restoring).
    @Published private(set) var state: AuthState?

    private let viewModel: AuthViewModel
    private var collectTask: Task<Void, Never>?

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
