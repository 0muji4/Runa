import SwiftUI
import Shared

/// 12 ふりかえりカレンダー. A quiet month grid: a serif "YYYY M月" header flanked by
/// prev/next chevrons (the title taps back to today), the 日〜土 row, then the days.
/// Per the confirmed design the moon lights ONLY on days that hold a record
/// ("記録のある日に、月あかり"); today wears a soft moonlight-pink outline. Tapping a
/// day opens its records, or the backdated writer when it has none.
struct CalendarView: View {
    @Binding var path: NavigationPath
    @StateObject private var model = CalendarObservable()
    @Environment(\.dismiss) private var dismiss

    private let columns = Array(repeating: GridItem(.flexible(), spacing: 0), count: 7)
    private let weekdays = ["日", "月", "火", "水", "木", "金", "土"]

    var body: some View {
        ZStack(alignment: .top) {
            RunaColors.background.ignoresSafeArea()
            VStack(alignment: .leading, spacing: 0) {
                Button { dismiss() } label: {
                    Text("‹ 戻る").font(RunaFonts.body(13)).foregroundStyle(RunaColors.subtle)
                }
                .padding(.top, 14)
                .padding(.vertical, 6)

                content
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
            .padding(.horizontal, 20)
        }
        .toolbar(.hidden, for: .navigationBar)
    }

    @ViewBuilder private var content: some View {
        if let state = model.state {
            switch onEnum(of: state) {
            case .content(let c): monthBody(c)
            case .loading: Spacer()
            }
        } else {
            Spacer()
        }
    }

    private func monthBody(_ c: CalendarUiStateContent) -> some View {
        VStack(spacing: 0) {
            header(c)
            Spacer().frame(height: 24)
            weekdayRow
            Spacer().frame(height: 8)
            LazyVGrid(columns: columns, spacing: 8) {
                ForEach(0..<Int(c.firstDayOfWeek), id: \.self) { _ in
                    Color.clear.frame(height: 58)
                }
                ForEach(c.days, id: \.day) { day in
                    dayCell(day)
                }
            }
            Spacer()
            legend(c.banner)
            Spacer().frame(height: 28)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private func header(_ c: CalendarUiStateContent) -> some View {
        HStack {
            Button { model.showPreviousMonth() } label: {
                Text("‹").font(RunaFonts.logo(34)).foregroundStyle(RunaColors.subtle)
            }
            Spacer()
            Button { model.showToday() } label: {
                HStack(spacing: 14) {
                    Text("\(c.year)").font(RunaFonts.logo(32)).foregroundStyle(RunaColors.heading)
                    Text("\(c.month)月").font(RunaFonts.heading(30)).foregroundStyle(RunaColors.heading)
                }
            }
            .buttonStyle(.plain)
            Spacer()
            Button { model.showNextMonth() } label: {
                Text("›").font(RunaFonts.logo(34)).foregroundStyle(RunaColors.subtle)
            }
        }
    }

    private var weekdayRow: some View {
        HStack(spacing: 0) {
            ForEach(weekdays, id: \.self) { d in
                Text(d)
                    .font(RunaFonts.body(12))
                    .foregroundStyle(RunaColors.subtle)
                    .frame(maxWidth: .infinity)
            }
        }
    }

    private func dayCell(_ day: CalendarDay) -> some View {
        let bright = day.isToday || day.entryCount > 0
        return VStack(spacing: 4) {
            Text("\(day.day)")
                .font(RunaFonts.body(15))
                .foregroundStyle(bright ? RunaColors.heading : RunaColors.subtle)
            if day.entryCount > 0 {
                MoonPhaseDisc(
                    illumination: CGFloat(day.illumination),
                    waxing: moonIsWaxing(key: day.phaseKey),
                    diameter: 18
                )
            }
        }
        .frame(maxWidth: .infinity, minHeight: 58)
        .overlay(
            day.isToday
                ? RoundedRectangle(cornerRadius: 14).stroke(RunaColors.accent.opacity(0.8), lineWidth: 1)
                : nil
        )
        .contentShape(Rectangle())
        .onTapGesture { onTap(day) }
    }

    private func legend(_ banner: CalendarBanner) -> some View {
        VStack(spacing: 12) {
            if let text = bannerText(banner) {
                Text(text).font(RunaFonts.body(13)).foregroundStyle(RunaColors.subtle)
            }
            HStack(spacing: 10) {
                Circle().fill(RunaColors.subAccent).frame(width: 8, height: 8)
                Text("記録のある日に、月あかり")
                    .font(RunaFonts.heading(13))
                    .foregroundStyle(RunaColors.subtle)
            }
        }
        .frame(maxWidth: .infinity)
    }

    private func bannerText(_ banner: CalendarBanner) -> String? {
        switch banner {
        case .offline: return "オフライン。綴った言葉は、端末に守られています。"
        case .error: return "同期に、少しつまずいています。"
        default: return nil // .none / .syncing stay silent
        }
    }

    private func onTap(_ day: CalendarDay) {
        let iso = isoDate(day)
        if day.entryCount > 0 {
            path.append(DiaryRoute.dayRecords(isoDate: iso))
        } else {
            path.append(DiaryRoute.writeOn(isoDate: iso))
        }
    }

    private func isoDate(_ day: CalendarDay) -> String {
        func pad(_ n: Int32) -> String { n < 10 ? "0\(n)" : "\(n)" }
        return "\(day.year)-\(pad(day.month))-\(pad(day.day))"
    }
}
