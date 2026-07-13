import SwiftUI
import Shared

/// Push targets within the diary tab's navigation stack.
enum DiaryRoute: Hashable {
    case editorNew
    case editor(clientId: String)
    case detail(clientId: String)
}

/// Diary list (09) — "日々の記録". A large 明朝 heading over a still column of record
/// cards, each led by its day's moon phase. Pull-to-refresh, a whisper-quiet sync
/// line, a new-moon empty state, and a round moonlight-pink FAB into the editor.
struct DiaryListView: View {
    @StateObject private var model = DiaryListObservable()
    @State private var path = NavigationPath()

    var body: some View {
        NavigationStack(path: $path) {
            ZStack(alignment: .bottomTrailing) {
                RunaColors.background.ignoresSafeArea()
                content
                if isContent { plusFab }
            }
            .toolbar(.hidden, for: .navigationBar)
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

    private var isContent: Bool {
        if let state = model.state, case .content = onEnum(of: state) { return true }
        return false
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
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                Text("日々の記録")
                    .font(RunaFonts.heading(34))
                    .foregroundStyle(RunaColors.heading)
                    .padding(.top, 40)
                    .padding(.bottom, 4)
                bannerLine(banner)
                ForEach(entries, id: \.clientId) { entry in
                    DiaryCardRow(entry: entry)
                        .contentShape(Rectangle())
                        .onTapGesture { path.append(DiaryRoute.detail(clientId: entry.clientId)) }
                }
            }
            .padding(.horizontal, 20)
            .padding(.bottom, 120)
        }
        .scrollIndicators(.hidden)
        .refreshable { model.refresh() }
    }

    private func emptyState(banner: SyncBanner) -> some View {
        VStack(spacing: 0) {
            bannerLine(banner)
            Spacer()
            NewMoonEmblem(diameter: 116)
            Text("まだ、なにもない夜。")
                .font(RunaFonts.heading(26))
                .foregroundStyle(RunaColors.heading)
                .padding(.top, RunaSpacing.md)
            Text("最初のひとことを、\nそっと綴ってみませんか。")
                .font(RunaFonts.body(14))
                .foregroundStyle(RunaColors.subtle)
                .multilineTextAlignment(.center)
                .padding(.top, RunaSpacing.sm)
            Button { path.append(DiaryRoute.editorNew) } label: {
                Text("綴りはじめる")
                    .font(RunaFonts.body(16))
                    .foregroundStyle(RunaColors.accent)
                    .padding(.horizontal, 32)
                    .padding(.vertical, 14)
                    .overlay(
                        RoundedRectangle(cornerRadius: 28)
                            .stroke(RunaColors.accent.opacity(0.7), lineWidth: 1)
                    )
            }
            .padding(.top, RunaSpacing.lg)
            Spacer()
            Spacer()
        }
        .frame(maxWidth: .infinity)
        .padding(.horizontal, RunaSpacing.lg)
    }

    private var plusFab: some View {
        Button { path.append(DiaryRoute.editorNew) } label: {
            ZStack {
                Circle().fill(RunaColors.accent).frame(width: 64, height: 64)
                Canvas { ctx, size in
                    let c = CGPoint(x: size.width / 2, y: size.height / 2)
                    let arm = size.width * 0.34
                    var h = Path(); h.move(to: CGPoint(x: c.x - arm, y: c.y)); h.addLine(to: CGPoint(x: c.x + arm, y: c.y))
                    var v = Path(); v.move(to: CGPoint(x: c.x, y: c.y - arm)); v.addLine(to: CGPoint(x: c.x, y: c.y + arm))
                    ctx.stroke(h, with: .color(RunaColors.background), style: StrokeStyle(lineWidth: 3, lineCap: .round))
                    ctx.stroke(v, with: .color(RunaColors.background), style: StrokeStyle(lineWidth: 3, lineCap: .round))
                }
                .frame(width: 22, height: 22)
            }
        }
        .buttonStyle(.plain)
        .padding(28)
    }

    @ViewBuilder private func bannerLine(_ banner: SyncBanner) -> some View {
        if let text = bannerText(banner) {
            Text(text)
                .font(RunaFonts.body(13))
                .foregroundStyle(RunaColors.subtle)
        }
    }

    private func bannerText(_ banner: SyncBanner) -> String? {
        switch banner {
        case .offline: return "オフライン。綴った言葉は、端末に守られています。"
        case .error: return "同期に、少しつまずいています。"
        default: return nil // .none / .syncing (the pull indicator covers syncing)
        }
    }
}

/// One quiet record card, led by its day's moon phase.
private struct DiaryCardRow: View {
    let entry: DiaryEntry

    var body: some View {
        let moon = DiaryMoonCalc.moon(epochMs: entry.createdAtEpochMs)
        return VStack(alignment: .leading, spacing: 12) {
            HStack(spacing: 10) {
                MoonPhaseDisc(illumination: moon.illumination, waxing: moon.waxing, diameter: 20)
                Text(DiaryDate.day(entry.createdAtEpochMs))
                    .font(RunaFonts.body(13))
                    .foregroundStyle(RunaColors.body)
                Text(moon.name)
                    .font(RunaFonts.body(13))
                    .foregroundStyle(RunaColors.subtle)
            }
            Text(entry.bodyText)
                .font(RunaFonts.heading(16))
                .foregroundStyle(RunaColors.body)
                .lineLimit(2)
                .lineSpacing(6)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 22)
        .padding(.vertical, 20)
        .background(RunaColors.surface)
        .clipShape(RoundedRectangle(cornerRadius: 22))
    }
}
