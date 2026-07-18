import Foundation
import Shared

/// ObservableObject bridge over the shared `InsightViewModel` (16 インサイト). Mirrors
/// the other observables: collect the SKIE-bridged `StateFlow` and republish on the
/// main actor; the period toggle and prev/next forward straight to the shared VM.
@MainActor
final class InsightObservable: ObservableObject {
    @Published private(set) var state: InsightUiState?

    private let viewModel: InsightViewModel
    private var collectTask: Task<Void, Never>?

    init(viewModel: InsightViewModel = resolveInsightViewModel()) {
        self.viewModel = viewModel
        collectTask = Task { [weak self] in
            guard let self else { return }
            let flow: SkieSwiftStateFlow<InsightUiState> = self.viewModel.state
            for await value in flow {
                self.state = value
            }
        }
    }

    func showWeekly() { viewModel.setPeriodType(type: .weekly) }
    func showMonthly() { viewModel.setPeriodType(type: .monthly) }
    func showPrevious() { viewModel.showPrevious() }
    func showNext() { viewModel.showNext() }
    func showCurrent() { viewModel.showCurrent() }

    deinit { collectTask?.cancel() }
}
