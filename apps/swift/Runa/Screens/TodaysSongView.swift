import SwiftUI
import Shared

/// Bridge over the shared `SongPlayerViewModel` (a Koin single, so every screen shares one player).
@MainActor
final class SongPlayerObservable: ObservableObject {
    @Published private(set) var state: PlayerUiState?

    private let viewModel: SongPlayerViewModel
    private var collectTask: Task<Void, Never>?

    init(viewModel: SongPlayerViewModel = resolveSongPlayerViewModel()) {
        self.viewModel = viewModel
        collectTask = Task { [weak self] in
            guard let self else { return }
            let stateFlow: SkieSwiftStateFlow<PlayerUiState> = self.viewModel.state
            for await value in stateFlow {
                self.state = value
            }
        }
    }

    func play(_ song: SongDto) { viewModel.play(song: song) }
    func togglePlayPause() { viewModel.togglePlayPause() }

    deinit { collectTask?.cancel() }
}

/// きょうの一曲: introduces the day's track (defaults to today's song from `HomeObservable`).
/// Apple's Promo Content terms (docs/dd/todays-song-itunes-preview.md): badge and
/// attribution on this screen, no seek.
struct TodaysSongView: View {
    @Environment(\.runaTheme) private var runaTheme
    @StateObject private var player = SongPlayerObservable()
    @StateObject private var home = HomeObservable()

    private var song: SongDto? { player.state?.song ?? home.todaySong }

    var body: some View {
        NavigationStack {
            ZStack {
                runaTheme.background.ignoresSafeArea()
                VStack(spacing: 0) {
                    RunaScreenHeader(title: L.tabTodaysSong) {
                        NavigationLink(destination: SongArchiveView()) {
                            Text(L.todaySongOpenArchive)
                                .font(RunaFonts.headerLabel)
                                .foregroundStyle(runaTheme.accent)
                        }
                    }
                    .padding(.horizontal, RunaSpacing.md)

                    ZStack {
                        if let song {
                            introduction(song)
                        } else {
                            RunaEmptyView(
                                title: L.todaySongNone,
                                message: L.todaySongNoneBody
                            )
                        }
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                }
            }
            .toolbar(.hidden, for: .navigationBar)
        }
    }

    private func introduction(_ song: SongDto) -> some View {
        let ps = player.state
        let isPlaying = ps?.isPlaying ?? false
        let duration = Double(ps?.durationMs ?? 0)
        let position = Double(ps?.positionMs ?? 0)
        let progress = duration > 0 ? min(max(position / duration, 0), 1) : 0

        // Artwork is capped so the badge, preview and attribution always fit
        // above the tab bar on a 4.7"-class screen.
        return VStack(spacing: RunaSpacing.sm) {
            AsyncImage(url: URL(string: song.artworkUrl)) { image in
                image.resizable().aspectRatio(1, contentMode: .fit)
            } placeholder: {
                runaTheme.surface
            }
            .frame(maxWidth: 280)
            .aspectRatio(1, contentMode: .fit)
            .clipShape(RoundedRectangle(cornerRadius: 16))

            VStack(spacing: 4) {
                Text(song.title).font(RunaFonts.heading(26)).foregroundStyle(runaTheme.heading)
                Text(song.artist).font(RunaFonts.body(16)).foregroundStyle(runaTheme.subtle)
            }
            .padding(.top, RunaSpacing.xs)

            AppleMusicBadge(storeUrl: song.storeUrl, height: 48)
                .padding(.top, RunaSpacing.sm)

            HStack(spacing: 12) {
                Button {
                    if player.state?.song == nil { player.play(song) } else { player.togglePlayPause() }
                } label: {
                    Image(systemName: isPlaying ? "pause.circle.fill" : "play.circle.fill")
                        .font(.system(size: 40))
                        .foregroundStyle(runaTheme.accent)
                }
                .accessibilityLabel(isPlaying ? L.playerPause : L.playerPlay)

                VStack(alignment: .leading, spacing: 8) {
                    Text(L.songPreviewLabel).font(RunaFonts.body(12)).foregroundStyle(runaTheme.subtle)
                    ProgressView(value: progress)
                        .tint(runaTheme.accent)
                }
            }
            .padding(.top, RunaSpacing.sm)

            ITunesCourtesyLine()
                .padding(.top, RunaSpacing.xs)
        }
        .padding(.horizontal, RunaSpacing.lg)
    }
}

#Preview {
    TodaysSongView().preferredColorScheme(.dark)
}
