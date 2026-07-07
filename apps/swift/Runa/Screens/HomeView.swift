import SwiftUI
import Shared

/// ObservableObject bridge over the shared `HealthzViewModel`.
///
/// SKIE bridges the Kotlin `StateFlow<HealthzUiState>` to `SkieSwiftStateFlow`,
/// which conforms to `AsyncSequence`, so we collect it in a Task and republish
/// each emission as a SwiftUI `@Published` value on the main actor.
@MainActor
final class HealthzObservable: ObservableObject {
    /// Latest UI state. `nil` before the first emission (treated as loading).
    @Published private(set) var state: HealthzUiState?

    private let viewModel: HealthzViewModel
    private var collectTask: Task<Void, Never>?

    // `resolveHealthzViewModel()` is a top-level Kotlin fun in shared/di/Koin.kt,
    // exported by SKIE as the global Swift func `resolveHealthzViewModel()`. It
    // pulls the VM out of the shared Koin graph (the same instance Android
    // resolves), so the base URL wired through initKoin reaches the health check.
    init(viewModel: HealthzViewModel = resolveHealthzViewModel()) {
        self.viewModel = viewModel
        startObserving()
    }

    private func startObserving() {
        collectTask = Task { [weak self] in
            guard let self else { return }
            // SKIE bridges the Kotlin StateFlow to SkieSwiftStateFlow, an
            // AsyncSequence, so `for await` iterates each emission directly.
            let stateFlow: SkieSwiftStateFlow<HealthzUiState> = self.viewModel.state
            for await value in stateFlow {
                self.state = value
            }
        }
    }

    /// Re-run the health check (the shared VM also calls check() on init).
    func check() {
        viewModel.check()
    }

    deinit {
        collectTask?.cancel()
    }
}

struct HomeView: View {
    /// The authenticated user's /me display name (auth slice's proof the
    /// protected endpoint works end to end).
    let displayName: String
    let onSignOut: () -> Void

    @StateObject private var healthz = HealthzObservable()

    var body: some View {
        NavigationStack {
            ScreenScaffold(
                title: "こんばんは、\(displayName) さん",
                placeholder: "月あかりのはじまり。ここにホームの内容が入ります。"
            ) {
                connectionStatus
                    .padding(.top, RunaSpacing.sm)
            }
            .toolbar {
                // Settings is reached from the Home top bar (not a bottom tab).
                ToolbarItem(placement: .topBarTrailing) {
                    NavigationLink(destination: SettingsView(onSignOut: onSignOut)) {
                        Image(systemName: "gearshape")
                            .foregroundStyle(RunaColors.subAccent)
                    }
                    .accessibilityLabel("設定")
                }
            }
        }
    }

    /// Maps the shared health state to the three required presentations.
    @ViewBuilder
    private var connectionStatus: some View {
        if let state = healthz.state {
            // SKIE exposes the sealed `HealthzUiState` via `onEnum(of:)` as an
            // exhaustive Swift enum: .loading / .ok(HealthzUiStateOk) /
            // .error(HealthzUiStateError).
            switch onEnum(of: state) {
            case .loading:
                loadingView
            case .ok:
                Text("接続OK")
                    .font(RunaFonts.body(16))
                    .foregroundStyle(RunaColors.accent)
            case .error(let error):
                VStack(alignment: .leading, spacing: RunaSpacing.xs) {
                    Text("接続エラー")
                        .font(RunaFonts.body(16))
                        .foregroundStyle(RunaColors.accent)
                    Text(error.message)
                        .font(RunaFonts.body(13))
                        .foregroundStyle(RunaColors.subtle)
                }
            }
        } else {
            loadingView
        }
    }

    private var loadingView: some View {
        HStack(spacing: RunaSpacing.xs) {
            ProgressView()
                .tint(RunaColors.accent)
            Text("接続確認中…")
                .font(RunaFonts.body(14))
                .foregroundStyle(RunaColors.subtle)
        }
    }
}

#Preview {
    HomeView(displayName: "Runa", onSignOut: {})
        .preferredColorScheme(.dark)
}
