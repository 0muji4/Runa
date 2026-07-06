import SwiftUI

/// ダイアリー — empty shell.
/// TODO: implement the diary feature here.
struct DiaryView: View {
    var body: some View {
        ScreenScaffold(
            title: "ダイアリー",
            placeholder: "日々の記録がここに並びます。"
        )
    }
}

#Preview {
    DiaryView()
        .preferredColorScheme(.dark)
}
