import SwiftUI
import Shared

@main
struct RunaApp: App {
    init() {
        // Give AsyncImage a real disk cache so gallery images fetched from their
        // presigned GET URLs survive going offline (the gallery delegates the image
        // body to the OS image cache; the shared layer only manages the URL's expiry).
        URLCache.shared = URLCache(memoryCapacity: 32 * 1024 * 1024, diskCapacity: 256 * 1024 * 1024)

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
            // ThemedRoot injects the selected app theme (dark/light/pink) into the
            // environment and drives the color scheme, so every screen recolors on a
            // theme change. RootView is the auth gate below it.
            ThemedRoot {
                RootView()
            }
        }
    }
}
