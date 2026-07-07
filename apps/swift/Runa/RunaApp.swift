import SwiftUI
import Shared

@main
struct RunaApp: App {
    init() {
        // Read the backend host+port injected via Info.plist (host+port ONLY,
        // WITHOUT /api/v1 — the shared module appends the API path itself).
        let baseUrl = (Bundle.main.object(forInfoDictionaryKey: "BASE_URL") as? String)
            ?? "http://localhost:8080"

        // Boot the shared Koin DI graph with this platform's base URL.
        // SKIE exports the top-level Kotlin `fun initKoin(baseUrl:)` as the global
        // Swift function `doInitKoin(baseUrl:)` — Kotlin/Native prefixes functions
        // whose name starts with `init` with `do` to avoid the ObjC init family.
        doInitKoin(baseUrl: baseUrl)
    }

    var body: some Scene {
        WindowGroup {
            // RootView is the auth gate: splash while restoring, the sign-in flow
            // when unauthenticated, the tab body when authenticated. It applies
            // the dark color scheme itself (belt-and-suspenders with Info.plist's
            // UIUserInterfaceStyle=Dark).
            RootView()
        }
    }
}
