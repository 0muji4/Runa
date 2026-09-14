import SwiftUI

/// Authenticated tab shell. Settings is not a tab; it is reached from HomeView's toolbar.
struct ContentView: View {
    @Environment(\.runaTheme) private var runaTheme
    let displayName: String
    let onSignOut: () -> Void

    var body: some View {
        TabView {
            HomeView(displayName: displayName, onSignOut: onSignOut)
                .tabItem {
                    Label(L.tabHome, systemImage: "moon.stars")
                }

            TodaysSongView()
                .tabItem {
                    Label(L.tabTodaysSong, systemImage: "music.note")
                }

            DiaryListView()
                .tabItem {
                    Label(L.tabDiary, systemImage: "doc.text")
                }

            GalleryView()
                .tabItem {
                    Label(L.tabGallery, systemImage: "photo.on.rectangle")
                }
        }
        .tint(runaTheme.accent)
    }
}

#Preview {
    ContentView(displayName: "LUNA", onSignOut: {})
        .preferredColorScheme(.dark)
}
