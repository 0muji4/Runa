import SwiftUI

/// ギャラリー — empty shell.
/// TODO: implement the gallery feature here.
struct GalleryView: View {
    var body: some View {
        ScreenScaffold(
            title: "ギャラリー",
            placeholder: "思い出のギャラリーがここに表示されます。"
        )
    }
}

#Preview {
    GalleryView()
        .preferredColorScheme(.dark)
}
