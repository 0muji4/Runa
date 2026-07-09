import SwiftUI
import Shared

/// Push targets within the diary tab's navigation stack.
enum DiaryRoute: Hashable {
    case editorNew
    case editor(clientId: String)
    case detail(clientId: String)
}

/// Diary list (screen 09). A still vertical list of record cards over the Runa
/// background, with pull-to-refresh, a subtle sync banner, a moon-motif empty
/// state, and a quiet "綴る" entry point into the editor.
struct DiaryListView: View {
    @StateObject private var model = DiaryListObservable()
    @State private var path = NavigationPath()

    var body: some View {
        NavigationStack(path: $path) {
            ZStack {
                RunaColors.background.ignoresSafeArea()
                content
            }
            .navigationTitle("ダイアリー")
            .toolbar {
                ToolbarItem(placement: .primaryAction) {
                    Button {
                        path.append(DiaryRoute.editorNew)
                    } label: {
                        Image(systemName: "square.and.pencil")
                    }
                }
            }
            .navigationDestination(for: DiaryRoute.self) { route in
                switch route {
                case .editorNew:
                    DiaryEditorView(clientId: nil)
                case .editor(let clientId):
                    DiaryEditorView(clientId: clientId)
                case .detail(let clientId):
                    DiaryDetailView(clientId: clientId, model: model, path: $path)
                }
            }
        }
        .tint(RunaColors.accent)
    }

    @ViewBuilder private var content: some View {
        if let state = model.state {
            switch onEnum(of: state) {
            case .loading:
                Color.clear
            case .content(let c):
                listBody(entries: c.entries, banner: c.banner)
            case .empty(let e):
                emptyState(banner: e.banner)
            }
        } else {
            Color.clear
        }
    }

    private func listBody(entries: [DiaryEntry], banner: SyncBanner) -> some View {
        VStack(spacing: 0) {
            bannerLine(banner)
            List {
                ForEach(entries, id: \.clientId) { entry in
                    NavigationLink(value: DiaryRoute.detail(clientId: entry.clientId)) {
                        DiaryCardRow(entry: entry)
                    }
                    .listRowBackground(RunaColors.surface)
                    .listRowSeparator(.hidden)
                    .listRowInsets(EdgeInsets(top: 6, leading: 16, bottom: 6, trailing: 16))
                }
            }
            .listStyle(.plain)
            .scrollContentBackground(.hidden)
            .refreshable { model.refresh() }
        }
    }

    private func emptyState(banner: SyncBanner) -> some View {
        VStack(spacing: RunaSpacing.sm) {
            bannerLine(banner)
            Spacer()
            Text("☾")
                .font(RunaFonts.logo(48))
                .foregroundStyle(RunaColors.subtle)
            Text("まだ、何も綴られていない")
                .font(RunaFonts.heading(22))
                .foregroundStyle(RunaColors.heading)
            Text("月のしずけさの中で、はじめの一行を。")
                .font(RunaFonts.body(14))
                .foregroundStyle(RunaColors.subtle)
            Spacer()
            Spacer()
        }
        .multilineTextAlignment(.center)
        .padding(.horizontal, RunaSpacing.lg)
    }

    @ViewBuilder private func bannerLine(_ banner: SyncBanner) -> some View {
        if let text = bannerText(banner) {
            Text(text)
                .font(RunaFonts.body(13))
                .foregroundStyle(RunaColors.subtle)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 10)
                .background(RunaColors.surface)
        }
    }

    private func bannerText(_ banner: SyncBanner) -> String? {
        switch banner {
        case .offline: return "オフライン ― 変更は端末に保存され、復帰後に同期されます"
        case .error: return "同期に失敗しました"
        default: return nil // .none / .syncing (the pull indicator covers syncing)
        }
    }
}

/// One quiet record card in the list.
private struct DiaryCardRow: View {
    let entry: DiaryEntry

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(entry.bodyText)
                .font(RunaFonts.body(16))
                .foregroundStyle(RunaColors.body)
                .lineLimit(3)
            Text(DiaryDate.string(entry.createdAtEpochMs))
                .font(RunaFonts.body(12))
                .foregroundStyle(RunaColors.subtle)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.vertical, 12)
    }
}
