import Foundation
import Shared

/// The calendar's page state, decoded from the shared `UiState<CalendarMonth>` into a
/// native Swift enum (local-first: effectively loading → content; offline rides on
/// `.content` as a `SyncPhase`).
enum CalendarUi {
    case loading
    case content(month: CalendarMonth, sync: SyncPhase)
    case failure(AppError)
}

/// ObservableObject bridge over the shared `CalendarViewModel`. Collects the SKIE-bridged
/// `StateFlow`, decodes to [CalendarUi], and republishes on the main actor; month
/// navigation forwards straight to the shared VM.
@MainActor
final class CalendarObservable: ObservableObject {
    @Published private(set) var ui: CalendarUi = .loading

    private let viewModel: CalendarViewModel
    private var collectTask: Task<Void, Never>?
    // Koin では factory 束縛なので画面ごとに新しい実体になる。Android の
    // ViewModelStore に相当する破棄を、この所有者が deinit で行う。
    private let owner = ViewModelOwner()

    init(viewModel: CalendarViewModel = resolveCalendarViewModel()) {
        self.viewModel = viewModel
        owner.own(viewModel: viewModel)
        collectTask = Task { [weak self] in
            guard let self else { return }
            for await value in self.viewModel.state {
                switch runaDecode(value, as: CalendarMonth.self) {
                case .content(let month, let sync): self.ui = .content(month: month, sync: sync)
                case .failure(let error): self.ui = .failure(error)
                case .loading, .empty: self.ui = .loading
                }
            }
        }
    }

    func showPreviousMonth() { viewModel.showPreviousMonth() }
    func showNextMonth() { viewModel.showNextMonth() }
    func showToday() { viewModel.showToday() }
    func refresh() { viewModel.refresh() }

    deinit {
        collectTask?.cancel()
        owner.dispose()
    }
}

/// The today's-moon page state (pure local computation: loading → content).
enum TodayMoonUi {
    case loading
    case content(moon: TodayMoon)
}

/// ObservableObject bridge over the shared `TodayMoonViewModel` (15 今日の月).
@MainActor
final class TodayMoonObservable: ObservableObject {
    @Published private(set) var ui: TodayMoonUi = .loading

    private let viewModel: TodayMoonViewModel
    private var collectTask: Task<Void, Never>?

    init(viewModel: TodayMoonViewModel = resolveTodayMoonViewModel()) {
        self.viewModel = viewModel
        collectTask = Task { [weak self] in
            guard let self else { return }
            for await value in self.viewModel.state {
                if case .content(let moon, _) = runaDecode(value, as: TodayMoon.self) {
                    self.ui = .content(moon: moon)
                } else {
                    self.ui = .loading
                }
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
    // Koin では factory 束縛なので画面ごとに新しい実体になる。Android の
    // ViewModelStore に相当する破棄を、この所有者が deinit で行う。
    private let owner = ViewModelOwner()

    init(isoDate: String) {
        self.viewModel = resolveDayRecordsViewModel(isoDate: isoDate)
        owner.own(viewModel: viewModel)
        self.dateLabel = viewModel.dateLabel
        collectTask = Task { [weak self] in
            guard let self else { return }
            let flow: SkieSwiftStateFlow<[DiaryEntry]> = self.viewModel.state
            for await value in flow {
                self.entries = value
            }
        }
    }

    deinit {
        collectTask?.cancel()
        owner.dispose()
    }
}
