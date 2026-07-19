import SwiftUI

/// Injects the selected app theme into the environment so every screen recolors in
/// place on a theme change — nav state is preserved and the theme picker previews
/// live (unlike an identity-reset approach). Also drives the system color scheme so
/// status-bar chrome stays legible under light/dark.
struct ThemedRoot<Content: View>: View {
    @StateObject private var theme = ThemeObservable()
    @ViewBuilder var content: () -> Content

    var body: some View {
        content()
            .environment(\.runaTheme, RunaTheme.forId(theme.themeId))
            .preferredColorScheme(theme.themeId == "light" ? .light : .dark)
    }
}
