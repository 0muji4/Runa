import Foundation
import Shared

/// ObservableObject bridge over the shared `DiaryListViewModel`. Mirrors
/// `AuthObservable`: it collects the SKIE-bridged `StateFlow` and republishes each
/// emission on the main actor; action methods forward straight to the shared VM.
final class DiaryListObservable: ObservableObject {
    @Published private(set) var state: DiaryListState?

    private let viewModel: DiaryListViewModel
    private var collectTask: Task<Void, Never>?

    init(viewModel: DiaryListViewModel = resolveDiaryListViewModel()) {
        self.viewModel = viewModel
        collectTask = Task { [weak self] in
            guard let self else { return }
            let flow: SkieSwiftStateFlow<DiaryListState> = self.viewModel.state
            for await value in flow {
                await MainActor.run { self.state = value }
            }
        }
    }

    func refresh() { viewModel.refresh() }
    func delete(clientId: String) { viewModel.delete(clientId: clientId) }

    /// Finds a cached entry by its local id (used by the detail screen).
    func entry(clientId: String) -> DiaryEntry? {
        guard let state, let content = state as? DiaryListStateContent else { return nil }
        return content.entries.first { $0.clientId == clientId }
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

/// Quiet mood options; `rawValue` is the string persisted through the API.
enum DiaryMood: String, CaseIterable {
    case calm, gentle, tired, hopeful, heavy

    var label: String {
        switch self {
        case .calm: return "しずか"
        case .gentle: return "おだやか"
        case .tired: return "つかれ"
        case .hopeful: return "のぞみ"
        case .heavy: return "おもい"
        }
    }
}

/// Formats an epoch-millis timestamp as a quiet Japanese date line.
enum DiaryDate {
    private static let formatter: DateFormatter = {
        let f = DateFormatter()
        f.locale = Locale(identifier: "ja_JP")
        f.dateFormat = "M月d日 HH:mm"
        return f
    }()

    static func string(_ epochMs: Int64) -> String {
        formatter.string(from: Date(timeIntervalSince1970: Double(epochMs) / 1000.0))
    }
}
