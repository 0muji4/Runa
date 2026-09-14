import Foundation
import Shared

/// The diary list's page state, decoded from the shared `UiState<List<DiaryEntry>>`.
enum DiaryListUi {
    case loading
    case empty
    case content(entries: [DiaryEntry], sync: SyncPhase)
    case failure(AppError)
}

/// ObservableObject bridge over the shared `DiaryListViewModel`.
final class DiaryListObservable: ObservableObject {
    @Published private(set) var ui: DiaryListUi = .loading

    private let viewModel: DiaryListViewModel
    private var collectTask: Task<Void, Never>?

    init(viewModel: DiaryListViewModel = resolveDiaryListViewModel()) {
        self.viewModel = viewModel
        collectTask = Task { [weak self] in
            guard let self else { return }
            for await value in self.viewModel.state {
                let mapped: DiaryListUi
                switch runaDecode(value, as: [DiaryEntry].self) {
                case .loading: mapped = .loading
                case .empty: mapped = .empty
                case .content(let entries, let sync): mapped = .content(entries: entries, sync: sync)
                case .failure(let error): mapped = .failure(error)
                }
                await MainActor.run { self.ui = mapped }
            }
        }
    }

    func refresh() { viewModel.refresh() }
    func delete(clientId: String) { viewModel.delete(clientId: clientId) }

    /// Finds a cached entry by its local id (used by the detail screen).
    func entry(clientId: String) -> DiaryEntry? {
        if case .content(let entries, _) = ui {
            return entries.first { $0.clientId == clientId }
        }
        return nil
    }

    deinit { collectTask?.cancel() }
}

/// ObservableObject bridge over a per-entry shared `DiaryEditorViewModel`.
final class DiaryEditorObservable: ObservableObject {
    @Published private(set) var state: DiaryEditorState?

    private let viewModel: DiaryEditorViewModel
    private var collectTask: Task<Void, Never>?
    // Koin の factory 束縛で画面ごとに新しい実体になるため、deinit で破棄する。
    private let owner = ViewModelOwner()

    init(clientId: String?) {
        self.viewModel = resolveDiaryEditorViewModel(clientId: clientId)
        startCollecting()
    }

    /// New entry backdated to a calendar day; created_at is that day's local noon.
    init(backdateEpochMs: Int64) {
        self.viewModel = resolveNewDiaryEditorViewModelOn(createdAtEpochMs: backdateEpochMs)
        startCollecting()
    }

    private func startCollecting() {
        owner.own(viewModel: viewModel)
        collectTask = Task { [weak self] in
            guard let self else { return }
            let flow: SkieSwiftStateFlow<DiaryEditorState> = self.viewModel.state
            for await value in flow {
                await MainActor.run { self.state = value }
            }
        }
    }

    func onBodyChange(_ text: String) { viewModel.onBodyChange(text: text) }
    func onMoodChange(_ mood: String?) { viewModel.onMoodChange(mood: mood) }
    func saveNow() { viewModel.saveNow() }

    deinit {
        collectTask?.cancel()
        owner.dispose()
    }
}

// Mood options come from the shared `diaryMoods()` / `diaryMoodLabelJa(mood:)`; no local mirror.

/// Japanese date formatting for the diary (day and weekday only, never a clock time).
enum DiaryDate {
    private static func formatter(_ pattern: String) -> DateFormatter {
        let f = DateFormatter()
        f.locale = Locale(identifier: "ja_JP")
        f.dateFormat = pattern
        return f
    }

    private static let dayFmt = formatter("M月d日")

    /// e.g. 7月4日 (no time).
    static func day(_ epochMs: Int64) -> String {
        dayFmt.string(from: Date(timeIntervalSince1970: Double(epochMs) / 1000.0))
    }

    /// e.g. 日曜.
    static func weekday(_ epochMs: Int64) -> String {
        var cal = Calendar(identifier: .gregorian)
        cal.timeZone = .current
        let date = Date(timeIntervalSince1970: Double(epochMs) / 1000.0)
        let names = ["日曜", "月曜", "火曜", "水曜", "木曜", "金曜", "土曜"]
        return names[(cal.component(.weekday, from: date) - 1) % 7]
    }
}
