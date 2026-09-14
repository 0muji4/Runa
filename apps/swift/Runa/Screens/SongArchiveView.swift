import SwiftUI
import Shared

/// ObservableObject bridge over the shared `SongArchiveViewModel`.
@MainActor
final class SongArchiveObservable: ObservableObject {
    @Published private(set) var state: ArchiveUiState?

    private let viewModel: SongArchiveViewModel
    private var collectTask: Task<Void, Never>?

    init(viewModel: SongArchiveViewModel = resolveSongArchiveViewModel()) {
        self.viewModel = viewModel
        collectTask = Task { [weak self] in
            guard let self else { return }
            let stateFlow: SkieSwiftStateFlow<ArchiveUiState> = self.viewModel.state
            for await value in stateFlow {
                self.state = value
            }
        }
    }

    func loadNextPage() { viewModel.loadNextPage(reset: false) }
    func reload() { viewModel.loadNextPage(reset: true) }

    deinit { collectTask?.cancel() }
}

/// これまでの一曲: the song archive plus the local play history. Apple's Promo Content terms
/// require the badge and iTunes attribution here (docs/dd/todays-song-itunes-preview.md).
struct SongArchiveView: View {
    @Environment(\.runaTheme) private var runaTheme
    @StateObject private var archive = SongArchiveObservable()
    @StateObject private var player = SongPlayerObservable()
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        ZStack {
            runaTheme.background.ignoresSafeArea()
            VStack(spacing: 0) {
                RunaScreenHeader(title: L.songArchiveTitle, onBack: { dismiss() })
                    .padding(.horizontal, RunaSpacing.md)
                archiveList
            }

            // State overlay only while no page has landed; the list is offline-tolerant after that.
            if archive.state?.songs.isEmpty ?? true {
                stateOverlay
            }
        }
        .toolbar(.hidden, for: .navigationBar)
        .toolbar(.hidden, for: .tabBar)
    }

    private var archiveList: some View {
        List {
            Section {
                if archive.state?.songs.isEmpty == false {
                    ITunesCourtesyLine()
                        .listRowBackground(runaTheme.background)
                }
                ForEach(archive.state?.songs ?? [], id: \.id) { song in
                    Button {
                        player.play(song)
                        dismiss()
                    } label: {
                        songRow(song)
                    }
                    .listRowBackground(runaTheme.background)
                }
                if archive.state?.canLoadMore == true {
                    Button(L.songArchiveLoadMore) { archive.loadNextPage() }
                        .foregroundStyle(runaTheme.accent)
                        .listRowBackground(runaTheme.background)
                }
            }

            let history = archive.state?.history ?? []
            if !history.isEmpty {
                Section(L.songArchiveHistory) {
                    ForEach(history, id: \.id) { entry in
                        Text("\(entry.title) · \(entry.artist)")
                            .font(RunaFonts.body(13)).foregroundStyle(runaTheme.subtle)
                            .listRowBackground(runaTheme.background)
                    }
                }
            }
        }
        .listStyle(.plain)
        .scrollContentBackground(.hidden)
    }

    @ViewBuilder private var stateOverlay: some View {
        if let state = archive.state {
            if state.isLoading {
                RunaLoadingView()
            } else if let error = state.error {
                RunaFailureView(error: error, onRetry: { archive.reload() })
            } else {
                RunaEmptyView(
                    title: L.songArchiveEmpty,
                    message: L.songArchiveEmptyBody
                )
            }
        } else {
            RunaLoadingView()
        }
    }

    private func songRow(_ song: SongDto) -> some View {
        HStack(spacing: RunaSpacing.sm) {
            AsyncImage(url: URL(string: song.artworkUrl)) { image in
                image.resizable().aspectRatio(1, contentMode: .fill)
            } placeholder: {
                runaTheme.surface
            }
            .frame(width: 56, height: 56)
            .clipShape(RoundedRectangle(cornerRadius: 8))

            VStack(alignment: .leading, spacing: 4) {
                Text(song.title).font(RunaFonts.heading(18)).foregroundStyle(runaTheme.heading).lineLimit(2)
                Text("\(song.artist) · \(song.date)").font(RunaFonts.body(13)).foregroundStyle(runaTheme.subtle)
            }
            Spacer()
            AppleMusicBadge(storeUrl: song.storeUrl, height: 28)
        }
    }
}

#Preview {
    NavigationStack { SongArchiveView() }.preferredColorScheme(.dark)
}
