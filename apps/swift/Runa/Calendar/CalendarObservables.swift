import Foundation
import Shared

/// ObservableObject bridge over the shared `CalendarViewModel`. Mirrors the other
/// observables: collect the SKIE-bridged `StateFlow` and republish on the main
/// actor; month navigation forwards straight to the shared VM.
@MainActor
final class CalendarObservable: ObservableObject {
    @Published private(set) var state: CalendarUiState?

    private let viewModel: CalendarViewModel
    private var collectTask: Task<Void, Never>?

    init(viewModel: CalendarViewModel = resolveCalendarViewModel()) {
        self.viewModel = viewModel
        collectTask = Task { [weak self] in
            guard let self else { return }
            let flow: SkieSwiftStateFlow<CalendarUiState> = self.viewModel.state
            for await value in flow {
                self.state = value
            }
        }
    }

    func showPreviousMonth() { viewModel.showPreviousMonth() }
    func showNextMonth() { viewModel.showNextMonth() }
    func showToday() { viewModel.showToday() }

    deinit { collectTask?.cancel() }
}

/// ObservableObject bridge over the shared `TodayMoonViewModel` (15 今日の月).
@MainActor
final class TodayMoonObservable: ObservableObject {
    @Published private(set) var state: TodayMoonUiState?

    private let viewModel: TodayMoonViewModel
    private var collectTask: Task<Void, Never>?

    init(viewModel: TodayMoonViewModel = resolveTodayMoonViewModel()) {
        self.viewModel = viewModel
        collectTask = Task { [weak self] in
            guard let self else { return }
            let flow: SkieSwiftStateFlow<TodayMoonUiState> = self.viewModel.state
            for await value in flow {
                self.state = value
            }
        }
    }

    deinit { collectTask?.cancel() }
}

/// ObservableObject bridge over a per-day `DayRecordsViewModel`. The tapped day is
/// passed as an ISO `yyyy-MM-dd` string; the shared VM streams that day's entries.
@MainActor
final class DayRecordsObservable: ObservableObject {
    @Published private(set) var entries: [DiaryEntry] = []
    let dateLabel: String

    private let viewModel: DayRecordsViewModel
    private var collectTask: Task<Void, Never>?

    init(isoDate: String) {
        self.viewModel = resolveDayRecordsViewModel(isoDate: isoDate)
        self.dateLabel = viewModel.dateLabel
        collectTask = Task { [weak self] in
            guard let self else { return }
            let flow: SkieSwiftStateFlow<[DiaryEntry]> = self.viewModel.state
            for await value in flow {
                self.entries = value
            }
        }
    }

    deinit { collectTask?.cancel() }
}
