import SwiftUI
import Shared

/// ObservableObject bridge over the shared `HomeViewModel`.
///
/// SKIE bridges the Kotlin `StateFlow<HomeUiState>` to a `SkieSwiftStateFlow` (an
/// `AsyncSequence`); we collect it and republish each emission on the main actor.
@MainActor
final class HomeObservable: ObservableObject {
    @Published private(set) var state: HomeUiState?

    private let viewModel: HomeViewModel
    private var collectTask: Task<Void, Never>?

    init(viewModel: HomeViewModel = resolveHomeViewModel()) {
        self.viewModel = viewModel
        collectTask = Task { [weak self] in
            guard let self else { return }
            let stateFlow: SkieSwiftStateFlow<HomeUiState> = self.viewModel.state
            for await value in stateFlow {
                self.state = value
            }
        }
    }

    /// Today's song, if the home has content (used by the player as its default).
    var todaySong: SongDto? {
        guard let state else { return nil }
        switch onEnum(of: state) {
        case .content(let c): return c.today.song
        case .offline(let o): return o.today.song
        default: return nil
        }
    }

    deinit { collectTask?.cancel() }
}

/// 06 Home. A quiet screen: a large 明朝 daily quote centered in generous
/// whitespace, with the day's moon phase + date above it. The quote and moon still
/// render when offline (the moon is always computed on-device).
struct HomeView: View {
    let displayName: String
    let onSignOut: () -> Void

    @StateObject private var home = HomeObservable()

    var body: some View {
        NavigationStack {
            ZStack {
                RunaColors.background.ignoresSafeArea()
                content
            }
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    NavigationLink(destination: SettingsView(onSignOut: onSignOut)) {
                        Image(systemName: "gearshape").foregroundStyle(RunaColors.subAccent)
                    }
                    .accessibilityLabel("設定")
                }
            }
        }
    }

    @ViewBuilder
    private var content: some View {
        if let state = home.state {
            switch onEnum(of: state) {
            case .content(let c): todayView(c.today, offline: false)
            case .offline(let o): todayView(o.today, offline: true)
            case .error:
                Text("今日をよみこめませんでした。")
                    .font(RunaFonts.body(16)).foregroundStyle(RunaColors.subtle)
            case .loading:
                ProgressView().tint(RunaColors.accent)
            }
        } else {
            ProgressView().tint(RunaColors.accent)
        }
    }

    private func todayView(_ today: Today, offline: Bool) -> some View {
        VStack(spacing: RunaSpacing.sm) {
            Text(moonPhaseGlyph(key: today.moon.phaseKey))
                .font(.system(size: 56))
            Text("\(today.dateLabel) · \(moonPhaseNameJa(key: today.moon.phaseKey))")
                .font(RunaFonts.heading(22)).foregroundStyle(RunaColors.heading)
            Text("照度 \(Int((today.moon.illumination * 100).rounded()))%")
                .font(RunaFonts.body(13)).foregroundStyle(RunaColors.subtle)

            Spacer().frame(height: RunaSpacing.lg)

            Text(today.quote?.bodyText ?? "今日の言葉は、まだ紡がれていません。")
                .font(RunaFonts.heading(26))
                .foregroundStyle(RunaColors.heading)
                .multilineTextAlignment(.center)
                .padding(.horizontal, RunaSpacing.md)

            if offline {
                Spacer().frame(height: RunaSpacing.md)
                Text("オフライン表示中（月あかりは端末で算出しています）")
                    .font(RunaFonts.body(13)).foregroundStyle(RunaColors.subtle)
                    .multilineTextAlignment(.center)
            }
        }
    }
}

#Preview {
    HomeView(displayName: "Runa", onSignOut: {})
        .preferredColorScheme(.dark)
}
