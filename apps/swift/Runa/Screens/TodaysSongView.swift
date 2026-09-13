import SwiftUI
import Shared

/// ObservableObject bridge over the shared `SongPlayerViewModel`. It is resolved
/// from the Koin single, so every screen that plays a song shares one player.
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

/// 07 きょうの一曲. Introduces the day's track from Apple's catalog: artwork, title,
/// the Apple Music badge as the main action, and a 30-second preview below it.
/// Defaults to today's song (from the shared `HomeObservable`); once a preview is
/// playing (today's or one chosen from the archive) it reflects the shared
/// `SongPlayerViewModel`'s live state. The layout follows Apple's Promo Content
/// terms (docs/dd/todays-song-itunes-preview.md, Q4): badge and attribution on
/// the same screen, no seek.
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
                    // No nav bar — the header is the page, like the other tabs.
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

            // The badge is the screen's main action: the preview below only
            // introduces the track, so it sits under the badge and cannot be scrubbed.
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
