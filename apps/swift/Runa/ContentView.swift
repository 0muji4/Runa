import SwiftUI

/// Root shell: 4 bottom tabs. Settings is NOT a tab — it is reached from the
/// Home tab's top bar (see HomeView's toolbar).
struct ContentView: View {
    var body: some View {
        TabView {
            HomeView()
                .tabItem {
                    Label("ホーム", systemImage: "moon.stars")
                }

            TodaysSongView()
                .tabItem {
                    Label("きょうの一曲", systemImage: "music.note")
                }

            DiaryView()
                .tabItem {
                    Label("ダイアリー", systemImage: "book.closed")
                }

            GalleryView()
                .tabItem {
                    Label("ギャラリー", systemImage: "photo.on.rectangle")
                }
        }
        .tint(RunaColors.accent)
    }
}

#Preview {
    ContentView()
        .preferredColorScheme(.dark)
}
