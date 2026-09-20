import Foundation
import Shared

/// Bridge over the shared `PendingRoute`; seeded synchronously so a cold-start tap is visible on the first frame.
@MainActor
final class PendingRouteObservable: ObservableObject {
    @Published private(set) var route: PendingRouteKind?

    private let pendingRoute: PendingRoute
    private var collectTask: Task<Void, Never>?

    init(pendingRoute: PendingRoute = resolvePendingRoute()) {
        self.pendingRoute = pendingRoute
        self.route = pendingRoute.currentRoute()
        collectTask = Task { [weak self] in
            guard let self else { return }
            for await value in self.pendingRoute.route { self.route = value }
        }
    }

    func consume() {
        // Cleared locally too, so a second observer in the same run loop does not route again.
        route = nil
        pendingRoute.consume()
    }

    deinit { collectTask?.cancel() }
}
