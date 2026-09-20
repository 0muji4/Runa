import SwiftUI
import Shared

/// Push targets within the diary tab's navigation stack.
enum DiaryRoute: Hashable {
    case editorNew
    case editor(clientId: String)
    case detail(clientId: String)
    case calendar
    case insight
    case dayRecords(isoDate: String)
    case writeOn(isoDate: String)
}

/// ダイアリー list; owns the diary tab's `NavigationPath` and its route destinations.
struct DiaryListView: View {
    @Environment(\.runaTheme) private var runaTheme
    @EnvironmentObject private var pending: PendingRouteObservable
    @StateObject private var model = DiaryListObservable()
    @State private var path = NavigationPath()

    var body: some View {
        NavigationStack(path: $path) {
            ZStack(alignment: .bottomTrailing) {
                runaTheme.background.ignoresSafeArea()
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
                case .calendar:
                    CalendarView(path: $path)
                case .insight:
                    InsightView()
                case .dayRecords(let isoDate):
                    DayRecordsView(isoDate: isoDate, path: $path)
                case .writeOn(let isoDate):
                    DiaryEditorView(backdateIsoDate: isoDate)
                }
            }
        }
        .tint(runaTheme.accent)
        // Both hooks: the route may predate this tab (cold-start tap) or arrive while it is visible.
        .onAppear { openPendingRoute() }
        .onChange(of: pending.route) { _ in openPendingRoute() }
    }

    private func openPendingRoute() {
        guard pending.route == .diaryEditorNew else { return }
        path.append(DiaryRoute.editorNew)
        pending.consume()
    }

    private var isContent: Bool {
        if case .content = model.ui { return true }
        return false
    }

    @ViewBuilder private var content: some View {
        switch model.ui {
        case .loading:
            Color.clear
        case .content(let entries, let sync):
            listBody(entries: entries, sync: sync)
        case .empty:
            emptyState()
        case .failure(let error):
            RunaFailureView(error: error, onRetry: { model.refresh() })
        }
    }

    private func listBody(entries: [DiaryEntry], sync: SyncPhase) -> some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                // Outside the spaced stack so the gap below the title is the header's own.
                RunaScreenHeader(title: L.diaryListTitle) {
                    HStack(spacing: 16) {
                        insightLink
                        calendarLink
                    }
                }
                VStack(alignment: .leading, spacing: 16) {
                    RunaSyncBanner(phase: sync)
                    ForEach(entries, id: \.clientId) { entry in
                        DiaryCardRow(entry: entry)
                            .contentShape(Rectangle())
                            .onTapGesture { path.append(DiaryRoute.detail(clientId: entry.clientId)) }
                    }
                }
            }
            .padding(.horizontal, 20)
            .padding(.bottom, 120)
        }
        .scrollIndicators(.hidden)
        .refreshable { model.refresh() }
    }

    private func emptyState() -> some View {
        VStack(spacing: 0) {
            RunaScreenHeader(title: L.diaryListTitle) {
                HStack(spacing: 16) {
                    insightLink
                    calendarLink
                }
            }
            .padding(.horizontal, 20)
            RunaEmptyView(
                title: L.diaryEmptyTitle,
                message: L.diaryEmptyBody,
                ctaLabel: L.diaryEmptyCta,
                onCta: { path.append(DiaryRoute.editorNew) }
            )
        }
    }

    private var calendarLink: some View {
        Button { path.append(DiaryRoute.calendar) } label: {
            Text(L.diaryOpenCalendar)
                .font(RunaFonts.headerLabel)
                .foregroundStyle(runaTheme.accent)
        }
    }

    private var insightLink: some View {
        Button { path.append(DiaryRoute.insight) } label: {
            Text(L.diaryOpenInsight)
                .font(RunaFonts.headerLabel)
                .foregroundStyle(runaTheme.accent)
        }
    }

    private var plusFab: some View {
        Button { path.append(DiaryRoute.editorNew) } label: {
            ZStack {
                Circle().fill(runaTheme.accent).frame(width: 64, height: 64)
                Canvas { ctx, size in
                    let c = CGPoint(x: size.width / 2, y: size.height / 2)
                    let arm = size.width * 0.34
                    var h = Path(); h.move(to: CGPoint(x: c.x - arm, y: c.y)); h.addLine(to: CGPoint(x: c.x + arm, y: c.y))
                    var v = Path(); v.move(to: CGPoint(x: c.x, y: c.y - arm)); v.addLine(to: CGPoint(x: c.x, y: c.y + arm))
                    ctx.stroke(h, with: .color(runaTheme.background), style: StrokeStyle(lineWidth: 3, lineCap: .round))
                    ctx.stroke(v, with: .color(runaTheme.background), style: StrokeStyle(lineWidth: 3, lineCap: .round))
                }
                .frame(width: 22, height: 22)
            }
        }
        .buttonStyle(.plain)
        .padding(28)
    }

}

private struct DiaryCardRow: View {
    @Environment(\.runaTheme) private var runaTheme
    let entry: DiaryEntry

    var body: some View {
        let moon = DiaryMoonCalc.moon(epochMs: entry.createdAtEpochMs)
        return VStack(alignment: .leading, spacing: 12) {
            HStack(spacing: 10) {
                MoonPhaseDisc(illumination: moon.illumination, waxing: moon.waxing, diameter: 20)
                Text(DiaryDate.day(entry.createdAtEpochMs))
                    .font(RunaFonts.body(13))
                    .foregroundStyle(runaTheme.body)
                Text(moon.name)
                    .font(RunaFonts.body(13))
                    .foregroundStyle(runaTheme.subtle)
            }
            Text(entry.bodyText)
                .font(RunaFonts.heading(16))
                .foregroundStyle(runaTheme.body)
                .lineLimit(2)
                .lineSpacing(6)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 22)
        .padding(.vertical, 20)
        .background(runaTheme.surface)
        .clipShape(RoundedRectangle(cornerRadius: 22))
    }
}
