import SwiftUI

/// 設定 — empty shell. Reached from the Home tab's top bar.
/// TODO: implement settings here.
struct SettingsView: View {
    var body: some View {
        ScreenScaffold(
            title: "設定",
            placeholder: "設定項目がここに表示されます。"
        )
        .navigationTitle("設定")
        .navigationBarTitleDisplayMode(.inline)
    }
}

#Preview {
    NavigationStack {
        SettingsView()
    }
    .preferredColorScheme(.dark)
}
