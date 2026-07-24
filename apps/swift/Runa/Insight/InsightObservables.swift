import Foundation
import Shared

/// The insight letter's page state, decoded from the shared `UiState<Insight>` into a
/// native Swift enum. Offline/sync ride on `.content` as a `SyncPhase`.
enum InsightUi {
    case loading
    case empty
    case content(insight: Insight, sync: SyncPhase)
    case failure(AppError)
}

/// ObservableObject bridge over the shared `InsightViewModel` (16 インサイト). Collects
/// the SKIE-bridged `StateFlow`, decodes to [InsightUi], and republishes on the main
/// actor; the period toggle and prev/next forward straight to the shared VM.
@MainActor
final class InsightObservable: ObservableObject {
    @Published private(set) var ui: InsightUi = .loading
    /// The period chrome (label + week/month mode), shown over content and empty alike.
    @Published private(set) var header: InsightHeader?

    private let viewModel: InsightViewModel
    private var collectTask: Task<Void, Never>?
    private var headerTask: Task<Void, Never>?

    init(viewModel: InsightViewModel = resolveInsightViewModel()) {
        self.viewModel = viewModel
        collectTask = Task { [weak self] in
            guard let self else { return }
            for await value in self.viewModel.state {
                switch runaDecode(value, as: Insight.self) {
                case .loading: self.ui = .loading
                case .empty: self.ui = .empty
                case .content(let insight, let sync): self.ui = .content(insight: insight, sync: sync)
                case .failure(let error): self.ui = .failure(error)
                }
            }
        }
        headerTask = Task { [weak self] in
            guard let self else { return }
            let flow: SkieSwiftStateFlow<InsightHeader> = self.viewModel.header
            for await value in flow {
                self.header = value
            }
        }
    }

    func showWeekly() { viewModel.setPeriodType(type: .weekly) }
    func showMonthly() { viewModel.setPeriodType(type: .monthly) }
    func showPrevious() { viewModel.showPrevious() }
    func showNext() { viewModel.showNext() }
    func showCurrent() { viewModel.showCurrent() }
    func refresh() { viewModel.refresh() }

    deinit {
        collectTask?.cancel()
        headerTask?.cancel()
    }
}
