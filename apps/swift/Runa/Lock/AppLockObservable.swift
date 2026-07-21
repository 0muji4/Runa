import Foundation
import Shared

/// Bridge over the shared `AppLockViewModel`. Drives both the privacy-lock gate
/// (`state`) and the 22 プライバシー・ロック toggle (`lockEnabled`). This is a layer
/// separate from auth: the gate hides content until the biometric prompt succeeds.
/// Seeded synchronously so the gate is correct before the first frame (no content
/// flashes behind an engaged lock).
@MainActor
final class AppLockObservable: ObservableObject {
    @Published private(set) var state: AppLockUiState
    @Published private(set) var lockEnabled: Bool

    private let viewModel: AppLockViewModel
    private var stateTask: Task<Void, Never>?
    private var enabledTask: Task<Void, Never>?

    init(viewModel: AppLockViewModel = resolveAppLockViewModel()) {
        self.viewModel = viewModel
        self.state = viewModel.currentState()
        self.lockEnabled = viewModel.currentLockEnabled()
        stateTask = Task { [weak self] in
            guard let self else { return }
            let flow: SkieSwiftStateFlow<AppLockUiState> = self.viewModel.state
            for await value in flow { self.state = value }
        }
        enabledTask = Task { [weak self] in
            guard let self else { return }
            let flow: SkieSwiftStateFlow<KotlinBoolean> = self.viewModel.lockEnabled
            for await value in flow { self.lockEnabled = value.boolValue }
        }
    }

    func onAppForegrounded() { viewModel.onAppForegrounded() }
    func onAppBackgrounded() { viewModel.onAppBackgrounded() }
    func authenticate() { viewModel.authenticate() }
    func setLockEnabled(_ on: Bool) { viewModel.setLockEnabled(enabled: on) }
    func biometricAvailable() -> Bool { viewModel.biometricAvailable() }

    deinit {
        stateTask?.cancel()
        enabledTask?.cancel()
    }
}
