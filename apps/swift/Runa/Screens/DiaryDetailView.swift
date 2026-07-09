import SwiftUI
import Shared

/// Diary detail (screen 11) — reading back a record. The body uses the #C8C6CE
/// body colour in Shippori Mincho for calm legibility, with quiet edit/delete
/// affordances. The entry is read from the (shared) list model's cache.
struct DiaryDetailView: View {
    let clientId: String
    @ObservedObject var model: DiaryListObservable
    @Binding var path: NavigationPath

    @State private var confirmDelete = false

    var body: some View {
        ZStack {
            RunaColors.background.ignoresSafeArea()

            if let entry = model.entry(clientId: clientId) {
                ScrollView {
                    VStack(alignment: .leading, spacing: RunaSpacing.md) {
                        Text(DiaryDate.string(entry.createdAtEpochMs))
                            .font(RunaFonts.body(13))
                            .foregroundStyle(RunaColors.subtle)

                        Text(entry.bodyText)
                            .font(RunaFonts.heading(18, relativeTo: .body))
                            .foregroundStyle(RunaColors.body)

                        if let mood = entry.mood, let label = DiaryMood(rawValue: mood)?.label {
                            Text(label)
                                .font(RunaFonts.body(13))
                                .foregroundStyle(RunaColors.subAccent)
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.horizontal, RunaSpacing.md)
                    .padding(.top, RunaSpacing.sm)
                }
            }
        }
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                HStack(spacing: RunaSpacing.sm) {
                    Button("編集") { path.append(DiaryRoute.editor(clientId: clientId)) }
                    Button("削除", role: .destructive) { confirmDelete = true }
                        .tint(RunaColors.accent)
                }
            }
        }
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
}
