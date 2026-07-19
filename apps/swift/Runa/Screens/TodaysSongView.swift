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
    func seek(_ positionMs: Int64) { viewModel.seekTo(positionMs: positionMs) }

    deinit { collectTask?.cancel() }
}

/// 07 きょうの一曲. A spacious, refined player. Defaults to today's song (from the
/// shared `HomeViewModel`); once a song is playing (today's or one chosen from the
/// archive) it reflects the shared `SongPlayerViewModel`'s live state.
struct TodaysSongView: View {
    @Environment(\.runaTheme) private var runaTheme
    @StateObject private var player = SongPlayerObservable()
    @StateObject private var home = HomeObservable()
    @State private var scrubbing: Double?

    private var song: SongDto? { player.state?.song ?? home.todaySong }

    var body: some View {
        NavigationStack {
            ZStack {
                runaTheme.background.ignoresSafeArea()
                if let song {
                    playerBody(song)
                } else {
                    Text("今日の一曲は、まだ選ばれていません。")
                        .font(RunaFonts.body(16)).foregroundStyle(runaTheme.subtle)
                }
            }
            .navigationTitle("きょうの一曲")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    NavigationLink(destination: SongArchiveView()) {
                        Text("これまでの一曲").foregroundStyle(runaTheme.accent)
                    }
                }
            }
        }
    }

    private func playerBody(_ song: SongDto) -> some View {
        let ps = player.state
        let isPlaying = ps?.isPlaying ?? false
        let duration = Double(ps?.durationMs ?? 0)
        let position = scrubbing ?? Double(ps?.positionMs ?? 0)

        return VStack(spacing: RunaSpacing.md) {
            AsyncImage(url: URL(string: song.artworkUrl)) { image in
                image.resizable().aspectRatio(1, contentMode: .fit)
            } placeholder: {
                runaTheme.surface
            }
            .frame(maxWidth: .infinity)
            .aspectRatio(1, contentMode: .fit)
            .clipShape(RoundedRectangle(cornerRadius: 16))

            Text(song.title).font(RunaFonts.heading(26)).foregroundStyle(runaTheme.heading)
            Text(song.artist).font(RunaFonts.body(16)).foregroundStyle(runaTheme.subtle)

            if duration > 0 {
                Slider(
                    value: Binding(get: { position }, set: { scrubbing = $0 }),
                    in: 0...duration,
                    onEditingChanged: { editing in
                        if !editing, let s = scrubbing { player.seek(Int64(s)); scrubbing = nil }
                    }
                )
                .tint(runaTheme.accent)
                HStack {
                    Text(timeLabel(Int64(position))).font(RunaFonts.body(13)).foregroundStyle(runaTheme.subtle)
                    Spacer()
                    Text(timeLabel(ps?.durationMs ?? 0)).font(RunaFonts.body(13)).foregroundStyle(runaTheme.subtle)
                }
            }

            Button {
                if player.state?.song == nil { player.play(song) } else { player.togglePlayPause() }
            } label: {
                Image(systemName: isPlaying ? "pause.circle.fill" : "play.circle.fill")
                    .font(.system(size: 56))
                    .foregroundStyle(runaTheme.accent)
            }
            .accessibilityLabel(isPlaying ? "一時停止" : "再生")
        }
        .padding(.horizontal, RunaSpacing.lg)
    }

    private func timeLabel(_ ms: Int64) -> String {
        let total = max(ms / 1000, 0)
        return String(format: "%d:%02d", total / 60, total % 60)
    }
}

#Preview {
    TodaysSongView().preferredColorScheme(.dark)
}
