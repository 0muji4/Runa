import Foundation
import Shared

/// Bridge over the shared `NotificationSettingsViewModel` for 通知設定 (21). Publishes
/// the reminder on/off, the chosen time and the preset chips, and forwards the
/// toggle / time-selection intents. Seeded synchronously so the screen opens on the
/// saved state (no flash).
@MainActor
final class NotificationSettingsObservable: ObservableObject {
    @Published private(set) var enabled: Bool
    @Published private(set) var time: ReminderTime
    @Published private(set) var presets: [ReminderTime]

    private let viewModel: NotificationSettingsViewModel
    private var collectTask: Task<Void, Never>?

    init(viewModel: NotificationSettingsViewModel = resolveNotificationSettingsViewModel()) {
        self.viewModel = viewModel
        let initial = viewModel.currentState()
        self.enabled = initial.enabled
        self.time = initial.time
        self.presets = initial.presets
        collectTask = Task { [weak self] in
            guard let self else { return }
            let flow: SkieSwiftStateFlow<NotificationUiState> = self.viewModel.state
            for await value in flow {
                self.enabled = value.enabled
                self.time = value.time
                self.presets = value.presets
            }
        }
    }

    func toggle(_ on: Bool) { viewModel.onToggle(enabled: on) }
    func selectTime(_ time: ReminderTime) { viewModel.onSelectTime(time: time) }

    deinit { collectTask?.cancel() }
}
