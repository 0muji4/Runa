import SwiftUI
import Shared

/// Home page state, decoded from the shared `UiState<Today>` (offline rides on `.content`).
enum HomeUi {
    case loading
    case content(today: Today, sync: SyncPhase)
    case failure(AppError)
}

/// ObservableObject bridge over the shared `HomeViewModel`.
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

    /// Today's song, if loaded (the player's default).
    var todaySong: SongDto? {
        if case .content(let today, _) = ui { return today.song }
        return nil
    }

    deinit { collectTask?.cancel() }
}

/// Home: the day's moon + date over the daily quote.
struct HomeView: View {
    @Environment(\.runaTheme) private var runaTheme
    let displayName: String
    let onSignOut: () -> Void

    @StateObject private var home = HomeObservable()

    var body: some View {
        NavigationStack {
            ZStack {
                runaTheme.background.ignoresSafeArea()
                RadialGradient(
                    gradient: Gradient(colors: [
                        Color(red: 0.97, green: 0.95, blue: 0.89).opacity(0.10),
                        .clear,
                    ]),
                    center: UnitPoint(x: 0.5, y: 0.16),
                    startRadius: 0,
                    endRadius: 340
                )
                .ignoresSafeArea()
                content
            }
            .toolbar(.hidden, for: .navigationBar)
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
        VStack(spacing: 0) {
            // Home has no screen title, but starts at the shared tab offset so all four tabs align.
            ZStack {
                NavigationLink {
                    TodaysMoonView()
                } label: {
                    HStack(spacing: 12) {
                        MoonPhaseDisc(
                            illumination: CGFloat(today.moon.illumination),
                            waxing: moonIsWaxing(key: today.moon.phaseKey),
                            diameter: 30
                        )
                        Text(today.dateLabel)
                            .font(RunaFonts.heading(22)).foregroundStyle(runaTheme.heading)
                        Text(moonPhaseNameJa(key: today.moon.phaseKey))
                            .font(RunaFonts.body(14)).foregroundStyle(runaTheme.subtle)
                    }
                }
                .buttonStyle(.plain)

                HStack {
                    Spacer()
                    NavigationLink(destination: SettingsView(onSignOut: onSignOut)) {
                        Image(systemName: "gearshape").foregroundStyle(runaTheme.subAccent)
                    }
                    .accessibilityLabel(L.tabSettings)
                }
            }
            .padding(.top, RunaHeaderMetrics.topTab)
            .padding(.horizontal, RunaSpacing.md)

            Spacer()

            Text(today.quote?.bodyText ?? L.homeNoQuote)
                .font(RunaFonts.heading(26))
                .foregroundStyle(runaTheme.heading)
                .multilineTextAlignment(.center)
                .padding(.horizontal, RunaSpacing.md)

            if today.quote != nil {
                Spacer().frame(height: RunaSpacing.md)
                Text("—  " + L.homeQuoteCaption + "  —")
                    .font(RunaFonts.body(13)).foregroundStyle(runaTheme.subtle)
            }

            if offline {
                Spacer().frame(height: RunaSpacing.md)
                Text(L.homeOfflineHint)
                    .font(RunaFonts.body(13)).foregroundStyle(runaTheme.subtle)
                    .multilineTextAlignment(.center)
            }

            Spacer()
            Spacer().frame(height: RunaSpacing.lg)
        }
    }
}

#Preview {
    HomeView(displayName: "LUNA", onSignOut: {})
        .preferredColorScheme(.dark)
}
