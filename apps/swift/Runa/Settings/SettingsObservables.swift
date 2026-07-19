import Foundation
import Shared

/// Bridge over the shared `ThemeViewModel`. Publishes the theme id string and drives
/// selection by id — this keeps the Swift side independent of the bridged Kotlin
/// enum's case names. Seeded synchronously so the first frame uses the saved theme.
@MainActor
final class ThemeObservable: ObservableObject {
    @Published private(set) var themeId: String

    private let viewModel: ThemeViewModel
    private var collectTask: Task<Void, Never>?

    init(viewModel: ThemeViewModel = resolveThemeViewModel()) {
        self.viewModel = viewModel
        self.themeId = viewModel.currentThemeId()
        collectTask = Task { [weak self] in
            guard let self else { return }
            let flow: SkieSwiftStateFlow<AppTheme> = self.viewModel.theme
            for await value in flow {
                self.themeId = value.id
            }
        }
    }

    func select(_ id: String) {
        viewModel.selectId(id: id)
    }

    deinit { collectTask?.cancel() }
}

/// Bridge over the shared `AccountViewModel` for the account-data screen (23).
@MainActor
final class AccountObservable: ObservableObject {
    @Published private(set) var state: AccountUiState?

    private let viewModel: AccountViewModel
    private var collectTask: Task<Void, Never>?

    init(viewModel: AccountViewModel = resolveAccountViewModel()) {
        self.viewModel = viewModel
        collectTask = Task { [weak self] in
            guard let self else { return }
            let flow: SkieSwiftStateFlow<AccountUiState> = self.viewModel.state
            for await value in flow {
                self.state = value
            }
        }
    }

    func loadProfile() { viewModel.loadProfile() }
    func startEditName() { viewModel.startEditName() }
    func onDisplayNameChange(_ value: String) { viewModel.onDisplayNameChange(value: value) }
    func cancelEditName() { viewModel.cancelEditName() }
    func saveName() { viewModel.saveName() }
    func export() { viewModel.export() }
    func clearExport() { viewModel.clearExport() }
    func requestDelete() { viewModel.requestDelete() }
    func cancelDelete() { viewModel.cancelDelete() }
    func confirmDelete() { viewModel.confirmDelete() }

    deinit { collectTask?.cancel() }
}
