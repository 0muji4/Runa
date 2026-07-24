import Foundation
import Shared

/// The diary list's page state, mapped from the shared `UiState<List<DiaryEntry>>`
/// into a native Swift enum in the observable so the view never touches SKIE
/// generics. Offline/sync ride along on `.content` as a `SyncPhase` (the quiet
/// banner); local-first means `.failure` effectively never occurs.
enum DiaryListUi {
    case loading
    case empty
    case content(entries: [DiaryEntry], sync: SyncPhase)
    case failure(AppError)
}

/// ObservableObject bridge over the shared `DiaryListViewModel`. Collects the
/// SKIE-bridged `StateFlow<UiState<…>>`, maps each emission to [DiaryListUi], and
/// republishes on the main actor; action methods forward straight to the shared VM.
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

    init(clientId: String?) {
        self.viewModel = resolveDiaryEditorViewModel(clientId: clientId)
        startCollecting()
    }

    /// New entry backdated to a calendar day (12 の空の日から綴る), created_at set to
    /// that day's local noon.
    init(backdateEpochMs: Int64) {
        self.viewModel = resolveNewDiaryEditorViewModelOn(createdAtEpochMs: backdateEpochMs)
        startCollecting()
    }

    private func startCollecting() {
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

    deinit { collectTask?.cancel() }
}

// Mood options live in the shared `DiaryMood` (com.runa.shared.feature.diary), the
// single source the editor's chips and the insight aggregation both read. iOS reaches
// them through the shared free functions `diaryMoods()` / `diaryMoodValue(mood:)` /
// `diaryMoodLabelJa(mood:)` (see DiaryMoodKt), the same way the moon phase name is
// read — no local mirror to drift out of sync.

/// Quiet Japanese date formatting for the diary. The design shows the day and the
/// moon phase — never a clock time — so the record stays timeless.
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

    /// e.g. 日曜 — short Japanese weekday for the editor/detail headers.
    static func weekday(_ epochMs: Int64) -> String {
        var cal = Calendar(identifier: .gregorian)
        cal.timeZone = .current
        let date = Date(timeIntervalSince1970: Double(epochMs) / 1000.0)
        let names = ["日曜", "月曜", "火曜", "水曜", "木曜", "金曜", "土曜"]
        return names[(cal.component(.weekday, from: date) - 1) % 7]
    }
}
