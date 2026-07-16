import SwiftUI
import Shared

/// The records of one calendar day, reached by tapping a day that has entries. A
/// quiet "M月d日" header over the day's record cards (each taps into the existing
/// diary detail), plus a subtle "この日を綴る" invitation into the backdated writer.
struct DayRecordsView: View {
    let isoDate: String
    @Binding var path: NavigationPath
    @StateObject private var model: DayRecordsObservable
    @Environment(\.dismiss) private var dismiss

    init(isoDate: String, path: Binding<NavigationPath>) {
        self.isoDate = isoDate
        self._path = path
        self._model = StateObject(wrappedValue: DayRecordsObservable(isoDate: isoDate))
    }

    var body: some View {
        ZStack {
            RunaColors.background.ignoresSafeArea()
            VStack(alignment: .leading, spacing: 0) {
                Button { dismiss() } label: {
                    Text("‹ 戻る").font(RunaFonts.body(13)).foregroundStyle(RunaColors.subtle)
                }
                .padding(.top, 14)
                .padding(.vertical, 6)

                Text(model.dateLabel)
                    .font(RunaFonts.heading(30))
                    .foregroundStyle(RunaColors.heading)
                    .padding(.top, 12)
                    .padding(.bottom, 20)

                ScrollView {
                    VStack(spacing: 16) {
                        ForEach(model.entries, id: \.clientId) { entry in
                            card(entry)
                                .contentShape(Rectangle())
                                .onTapGesture { path.append(DiaryRoute.detail(clientId: entry.clientId)) }
                        }
                    }
                }
                .scrollIndicators(.hidden)

                Button { path.append(DiaryRoute.writeOn(isoDate: isoDate)) } label: {
                    Text("この日を綴る")
                        .font(RunaFonts.body(16))
                        .foregroundStyle(RunaColors.accent)
                        .padding(.horizontal, 32)
                        .padding(.vertical, 14)
                        .overlay(
                            RoundedRectangle(cornerRadius: 28)
                                .stroke(RunaColors.accent.opacity(0.7), lineWidth: 1)
                        )
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 20)
            }
            .padding(.horizontal, 20)
        }
        .toolbar(.hidden, for: .navigationBar)
    }

    private func card(_ entry: DiaryEntry) -> some View {
        let moon = DiaryMoonCalc.moon(epochMs: entry.createdAtEpochMs)
        return VStack(alignment: .leading, spacing: 12) {
            HStack(spacing: 10) {
                MoonPhaseDisc(illumination: moon.illumination, waxing: moon.waxing, diameter: 20)
                Text(DiaryDate.weekday(entry.createdAtEpochMs))
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
