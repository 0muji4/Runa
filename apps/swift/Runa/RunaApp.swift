import SwiftUI
import Shared

@main
struct RunaApp: App {
    @UIApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate
    @Environment(\.scenePhase) private var scenePhase
    // App-lifetime lock gate driven by the scene phase (see LockGateView); wraps the auth gate.
    @StateObject private var lock = AppLockObservable()

    init() {
        // Disk cache for AsyncImage: gallery image bodies live here, not in the shared layer.
        URLCache.shared = URLCache(memoryCapacity: 32 * 1024 * 1024, diskCapacity: 256 * 1024 * 1024)

        // Host+port only, WITHOUT /api/v1 — the shared module appends the API path.
        let baseUrl = (Bundle.main.object(forInfoDictionaryKey: "BASE_URL") as? String)
            ?? "http://localhost:8080"

        // Kotlin `initKoin` is exported as `doInitKoin`: Kotlin/Native avoids the ObjC init family.
        doInitKoin(baseUrl: baseUrl)
    }

    var body: some Scene {
        WindowGroup {
            ThemedRoot {
                LockGateView(lock: lock) {
                    RootView()
                }
            }
            .onChange(of: scenePhase) { phase in
                if phase == .active { resolveDeviceRegistrar().refresh() }
            }
        }
    }
}
