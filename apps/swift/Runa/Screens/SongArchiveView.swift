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

    deinit { collectTask?.cancel() }
}

/// 08 これまでの一曲. The song archive (newest first) plus the local play history.
/// Tapping a song plays it through the shared player and returns to the player (07).
struct SongArchiveView: View {
    @StateObject private var archive = SongArchiveObservable()
    @StateObject private var player = SongPlayerObservable()
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        ZStack {
            RunaColors.background.ignoresSafeArea()
            List {
                Section {
                    ForEach(archive.state?.songs ?? [], id: \.id) { song in
                        Button {
                            player.play(song)
                            dismiss()
                        } label: {
                            songRow(song)
                        }
                        .listRowBackground(RunaColors.background)
                    }
                    if archive.state?.canLoadMore == true {
                        Button("もっと見る") { archive.loadNextPage() }
                            .foregroundStyle(RunaColors.accent)
                            .listRowBackground(RunaColors.background)
                    }
                }

                let history = archive.state?.history ?? []
                if !history.isEmpty {
                    Section("再生の記録") {
                        ForEach(history, id: \.id) { entry in
                            Text("\(entry.title) · \(entry.artist)")
                                .font(RunaFonts.body(13)).foregroundStyle(RunaColors.subtle)
                                .listRowBackground(RunaColors.background)
                        }
                    }
                }
            }
            .listStyle(.plain)
            .scrollContentBackground(.hidden)

            if (archive.state?.songs.isEmpty ?? true), archive.state?.isLoading == false {
                Text("アーカイブはまだありません。")
                    .font(RunaFonts.body(16)).foregroundStyle(RunaColors.subtle)
            }
        }
        .navigationTitle("これまでの一曲")
        .navigationBarTitleDisplayMode(.inline)
    }

    private func songRow(_ song: SongDto) -> some View {
        HStack(spacing: RunaSpacing.sm) {
            AsyncImage(url: URL(string: song.artworkUrl)) { image in
                image.resizable().aspectRatio(1, contentMode: .fill)
            } placeholder: {
                RunaColors.surface
            }
            .frame(width: 56, height: 56)
            .clipShape(RoundedRectangle(cornerRadius: 8))

            VStack(alignment: .leading, spacing: 4) {
                Text(song.title).font(RunaFonts.heading(18)).foregroundStyle(RunaColors.heading)
                Text("\(song.artist) · \(song.date)").font(RunaFonts.body(13)).foregroundStyle(RunaColors.subtle)
            }
            Spacer()
        }
    }
}

#Preview {
    NavigationStack { SongArchiveView() }.preferredColorScheme(.dark)
}
