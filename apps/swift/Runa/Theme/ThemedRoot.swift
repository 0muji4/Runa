import SwiftUI

/// Injects the selected app theme into the environment and drives the system color scheme.
struct ThemedRoot<Content: View>: View {
    @StateObject private var theme = ThemeObservable()
    @ViewBuilder var content: () -> Content

    var body: some View {
        content()
            .environment(\.runaTheme, RunaTheme.forId(theme.themeId))
            .preferredColorScheme(theme.themeId == "light" ? .light : .dark)
    }
}
