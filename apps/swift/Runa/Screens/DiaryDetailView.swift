import SwiftUI
import Shared

/// Diary detail (11) — reading a record back. A moon-led header (phase disc, date,
/// phase · weekday) over the body in #C8C6CE 明朝 for calm legibility, with quiet
/// edit/delete affordances. The entry is read from the (shared) list model's cache.
struct DiaryDetailView: View {
    @Environment(\.runaTheme) private var runaTheme
    let clientId: String
    @ObservedObject var model: DiaryListObservable
    @Binding var path: NavigationPath

    @State private var confirmDelete = false

    var body: some View {
        ZStack {
            runaTheme.background.ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 0) {
                    topBar
                    if let entry = model.entry(clientId: clientId) {
                        let moon = DiaryMoonCalc.moon(epochMs: entry.createdAtEpochMs)
                        HStack(spacing: 14) {
                            MoonPhaseDisc(illumination: moon.illumination, waxing: moon.waxing, diameter: 44)
                            VStack(alignment: .leading, spacing: 2) {
                                Text(DiaryDate.day(entry.createdAtEpochMs))
                                    .font(RunaFonts.heading(26))
                                    .foregroundStyle(runaTheme.heading)
                                Text("\(moon.name)　・　\(DiaryDate.weekday(entry.createdAtEpochMs))")
                                    .font(RunaFonts.body(13))
                                    .foregroundStyle(runaTheme.subtle)
                            }
                        }
                        .padding(.top, RunaSpacing.md)

                        Text(entry.bodyText)
                            .font(RunaFonts.heading(18, relativeTo: .body))
                            .foregroundStyle(runaTheme.body)
                            .lineSpacing(8)
                            .padding(.top, RunaSpacing.lg)
                    }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.horizontal, RunaSpacing.md)
                .padding(.top, RunaSpacing.sm)
            }
        }
        .toolbar(.hidden, for: .navigationBar)
        .alert("この記録を削除しますか", isPresented: $confirmDelete) {
            Button("やめる", role: .cancel) {}
            Button("削除する", role: .destructive) {
                model.delete(clientId: clientId)
                if !path.isEmpty { path.removeLast() }
            }
        } message: {
            Text("削除した記録は元に戻せません。")
        }
    }

    private var topBar: some View {
        HStack(alignment: .center) {
            Text("‹ 記録")
                .font(RunaFonts.body(16))
                .foregroundStyle(runaTheme.subtle)
                .onTapGesture { if !path.isEmpty { path.removeLast() } }
            Spacer()
            Text("編集")
                .font(RunaFonts.body(13))
                .foregroundStyle(runaTheme.subtle)
                .padding(8)
                .onTapGesture { path.append(DiaryRoute.editor(clientId: clientId)) }
            Text("削除")
                .font(RunaFonts.body(13))
                .foregroundStyle(runaTheme.accent)
                .padding(8)
                .onTapGesture { confirmDelete = true }
        }
        .padding(.vertical, 6)
    }
}
