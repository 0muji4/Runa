import SwiftUI
import Shared

/// Authenticated tab shell. Settings is not a tab; it is reached from HomeView's toolbar.
struct ContentView: View {
    private enum ContentTab: Hashable { case home, todaysSong, diary, gallery }

    @Environment(\.runaTheme) private var runaTheme
    @EnvironmentObject private var pending: PendingRouteObservable
    @State private var selection: ContentTab = .home
    let displayName: String
    let onSignOut: () -> Void

    var body: some View {
        TabView(selection: $selection) {
            HomeView(displayName: displayName, onSignOut: onSignOut)
                .tabItem {
                    Label(L.tabHome, systemImage: "moon.stars")
                }
                .tag(ContentTab.home)

            TodaysSongView()
                .tabItem {
                    Label(L.tabTodaysSong, systemImage: "music.note")
                }
                .tag(ContentTab.todaysSong)

            DiaryListView()
                .tabItem {
                    Label(L.tabDiary, systemImage: "doc.text")
                }
                .tag(ContentTab.diary)

            GalleryView()
                .tabItem {
                    Label(L.tabGallery, systemImage: "photo.on.rectangle")
                }
                .tag(ContentTab.gallery)
        }
        .tint(runaTheme.accent)
        // `onChange` misses a route set before this view existed (cold-start tap), hence `onAppear` too.
        .onAppear { selectTab(for: pending.route) }
        .onChange(of: pending.route) { route in selectTab(for: route) }
    }

    // Uses the delivered value: `DiaryListView` may consume the route before this handler runs.
    private func selectTab(for route: PendingRouteKind?) {
        if route == .diaryEditorNew { selection = .diary }
    }
}

#Preview {
    ContentView(displayName: "LUNA", onSignOut: {})
        .preferredColorScheme(.dark)
}
