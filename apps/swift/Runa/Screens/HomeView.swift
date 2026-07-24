import SwiftUI
import Shared

/// The home's page state, decoded from the shared `UiState<Today>` into a native Swift
/// enum in the observable so the view never touches SKIE generics. The home always has
/// content (the moon is computed locally); offline rides on `.content` as a `SyncPhase`.
enum HomeUi {
    case loading
    case content(today: Today, sync: SyncPhase)
    case failure(AppError)
}

/// ObservableObject bridge over the shared `HomeViewModel`. Collects the SKIE-bridged
/// `StateFlow<UiState<Today>>`, decodes each emission to [HomeUi], and republishes on
/// the main actor.
@MainActor
final class HomeObservable: ObservableObject {
    @Published private(set) var ui: HomeUi = .loading

    private let viewModel: HomeViewModel
    private var collectTask: Task<Void, Never>?

    init(viewModel: HomeViewModel = resolveHomeViewModel()) {
        self.viewModel = viewModel
        collectTask = Task { [weak self] in
            guard let self else { return }
            for await value in self.viewModel.state {
                switch runaDecode(value, as: Today.self) {
                case .content(let today, let sync): self.ui = .content(today: today, sync: sync)
                case .failure(let error): self.ui = .failure(error)
                case .loading, .empty: self.ui = .loading
                }
            }
        }
    }

    func reload() { viewModel.load() }

    /// Today's song, if the home has content (used by the player as its default).
    var todaySong: SongDto? {
        if case .content(let today, _) = ui { return today.song }
        return nil
    }

    deinit { collectTask?.cancel() }
}

/// 06 Home. A quiet screen: a large 明朝 daily quote centered in generous
/// whitespace, with the day's moon phase + date above it. The quote and moon still
/// render when offline (the moon is always computed on-device).
struct HomeView: View {
    @Environment(\.runaTheme) private var runaTheme
    let displayName: String
    let onSignOut: () -> Void

    @StateObject private var home = HomeObservable()

    var body: some View {
        NavigationStack {
            ZStack {
                runaTheme.background.ignoresSafeArea()
                content
            }
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    NavigationLink(destination: SettingsView(onSignOut: onSignOut)) {
                        Image(systemName: "gearshape").foregroundStyle(runaTheme.subAccent)
                    }
                    .accessibilityLabel("設定")
                }
            }
        }
    }

    @ViewBuilder
    private var content: some View {
        switch home.ui {
        case .content(let today, let sync): todayView(today, offline: isOffline(sync))
        case .loading: RunaLoadingView()
        case .failure(let error): RunaFailureView(error: error, onRetry: { home.reload() })
        }
    }

    private func isOffline(_ phase: SyncPhase) -> Bool {
        switch phase {
        case .offline: return true
        default: return false
        }
    }

    private func todayView(_ today: Today, offline: Bool) -> some View {
        VStack(spacing: RunaSpacing.sm) {
            // Tapping the moon opens 15 今日の月.
            NavigationLink {
                TodaysMoonView()
            } label: {
                Text(moonPhaseGlyph(key: today.moon.phaseKey))
                    .font(.system(size: 56))
            }
            .buttonStyle(.plain)
            Text("\(today.dateLabel) · \(moonPhaseNameJa(key: today.moon.phaseKey))")
                .font(RunaFonts.heading(22)).foregroundStyle(runaTheme.heading)
            Text("照度 \(Int((today.moon.illumination * 100).rounded()))%")
                .font(RunaFonts.body(13)).foregroundStyle(runaTheme.subtle)

            Spacer().frame(height: RunaSpacing.lg)

            Text(today.quote?.bodyText ?? "今日の言葉は、まだ紡がれていません。")
                .font(RunaFonts.heading(26))
                .foregroundStyle(runaTheme.heading)
                .multilineTextAlignment(.center)
                .padding(.horizontal, RunaSpacing.md)

            if offline {
                Spacer().frame(height: RunaSpacing.md)
                Text("オフライン表示中（月あかりは端末で算出しています）")
                    .font(RunaFonts.body(13)).foregroundStyle(runaTheme.subtle)
                    .multilineTextAlignment(.center)
            }
        }
    }
}

#Preview {
    HomeView(displayName: "Runa", onSignOut: {})
        .preferredColorScheme(.dark)
}
