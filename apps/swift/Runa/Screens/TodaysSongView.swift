import SwiftUI

/// きょうの一曲 — empty shell.
/// TODO: implement the "today's song" feature here.
struct TodaysSongView: View {
    var body: some View {
        ScreenScaffold(
            title: "きょうの一曲",
            placeholder: "きょうの一曲がここに表示されます。"
        )
    }
}

#Preview {
    TodaysSongView()
        .preferredColorScheme(.dark)
}
