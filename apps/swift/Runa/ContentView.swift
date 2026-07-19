import SwiftUI

/// Authenticated tab shell: 4 bottom tabs. Settings is NOT a tab — it is reached
/// from the Home tab's top bar (see HomeView's toolbar), which also hosts
/// sign-out. [displayName] comes from the shared /me lookup; [onSignOut] flips
/// the shared auth state back to unauthenticated.
struct ContentView: View {
    @Environment(\.runaTheme) private var runaTheme
    let displayName: String
    let onSignOut: () -> Void

    var body: some View {
        TabView {
            HomeView(displayName: displayName, onSignOut: onSignOut)
                .tabItem {
                    Label("ホーム", systemImage: "moon.stars")
                }

            TodaysSongView()
                .tabItem {
                    Label("きょうの一曲", systemImage: "music.note")
                }

            DiaryListView()
                .tabItem {
                    Label("ダイアリー", systemImage: "doc.text")
                }

            GalleryView()
                .tabItem {
                    Label("ギャラリー", systemImage: "photo.on.rectangle")
                }
        }
        .tint(runaTheme.accent)
    }
}

#Preview {
    ContentView(displayName: "Runa", onSignOut: {})
        .preferredColorScheme(.dark)
}
