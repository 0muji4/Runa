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
                    let entry = model.entry(clientId: clientId)
                    // The date is the screen title; the moon and weekday sit under it
                    // as meta.
                    RunaScreenHeader(
                        title: entry.map { DiaryDate.day($0.createdAtEpochMs) },
                        onBack: { if !path.isEmpty { path.removeLast() } }
                    ) {
                        Text(L.diaryActionEdit)
                            .font(RunaFonts.headerLabel)
                            .foregroundStyle(runaTheme.subtle)
                            .padding(8)
                            .onTapGesture { path.append(DiaryRoute.editor(clientId: clientId)) }
                        Text(L.diaryActionDelete)
                            .font(RunaFonts.headerLabel)
                            .foregroundStyle(runaTheme.accent)
                            .padding(8)
                            .onTapGesture { confirmDelete = true }
                    }

                    if let entry {
                        let moon = DiaryMoonCalc.moon(epochMs: entry.createdAtEpochMs)
                        HStack(spacing: 14) {
                            MoonPhaseDisc(illumination: moon.illumination, waxing: moon.waxing, diameter: 44)
                            Text("\(moon.name)　・　\(DiaryDate.weekday(entry.createdAtEpochMs))")
                                .font(RunaFonts.headerLabel)
                                .foregroundStyle(runaTheme.subtle)
                        }

                        Text(entry.bodyText)
                            .font(RunaFonts.heading(18, relativeTo: .body))
                            .foregroundStyle(runaTheme.body)
                            .lineSpacing(8)
                            .padding(.top, RunaSpacing.lg)
                    }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.horizontal, RunaSpacing.md)
            }
        }
        .toolbar(.hidden, for: .navigationBar, .tabBar)
        .alert(L.diaryDeleteConfirmTitle, isPresented: $confirmDelete) {
            Button(L.actionCancel, role: .cancel) {}
            Button(L.actionDelete, role: .destructive) {
                model.delete(clientId: clientId)
                if !path.isEmpty { path.removeLast() }
            }
        } message: {
            Text(L.diaryDeleteConfirmBody)
        }
    }
}
