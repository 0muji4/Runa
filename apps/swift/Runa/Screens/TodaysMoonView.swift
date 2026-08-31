import SwiftUI
import Shared

/// 15 今日の月. A large, quiet moon under the "今日の月" label, its phase name, 月齢 and
/// date, a hushed 明朝 line for the phase, and a whisper of the next principal phase.
/// Reached from the home screen's moon. Fully offline — every value is computed on
/// device by the shared moon calculator.
struct TodaysMoonView: View {
    @Environment(\.runaTheme) private var runaTheme
    @StateObject private var model = TodayMoonObservable()
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        ZStack {
            runaTheme.background.ignoresSafeArea()
            VStack(alignment: .leading, spacing: 0) {
                RunaScreenHeader(title: L.todaysMoonLabel, onBack: { dismiss() })
                content
            }
            .padding(.horizontal, 32)
        }
        .toolbar(.hidden, for: .navigationBar, .tabBar)
    }

    @ViewBuilder private var content: some View {
        // Pure local computation: settles to Content synchronously.
        if case .content(let moon) = model.ui {
            moonView(moon)
        } else {
            RunaLoadingView()
        }
    }

    private func moonView(_ moon: TodayMoon) -> some View {
        VStack(spacing: 0) {
            Spacer()
            MoonPhaseDisc(
                illumination: CGFloat(moon.illumination),
                waxing: moonIsWaxing(key: moon.phaseKey),
                diameter: 236
            )

            Spacer().frame(height: 40)
            Text(moonPhaseNameJa(key: moon.phaseKey))
                .font(RunaFonts.heading(34))
                .foregroundStyle(runaTheme.heading)
            Spacer().frame(height: 10)
            Text(L.todaysMoonAge(String(format: "%.1f", moon.ageDays), moon.dateLabel))
                .font(RunaFonts.body(13))
                .foregroundStyle(runaTheme.subtle)

            Spacer().frame(height: 36)
            Text(moon.phrase)
                .font(RunaFonts.heading(20))
                .foregroundStyle(runaTheme.body)
                .multilineTextAlignment(.center)
                .lineSpacing(8)

            Spacer()
            Text(L.todaysMoonNext(moon.nextPhaseDateLabel, moonPhaseNameJa(key: moon.nextPhaseKey)))
                .font(RunaFonts.body(13))
                .foregroundStyle(runaTheme.subtle)
                .padding(.bottom, 40)
        }
        .frame(maxWidth: .infinity)
    }
}
