import SwiftUI

/// Shared empty-shell layout: a heading + a placeholder line on the Runa
/// background. Feature screens will replace `content` with real UI later.
struct ScreenScaffold<Content: View>: View {
    @Environment(\.runaTheme) private var runaTheme
    let title: String
    let placeholder: String
    @ViewBuilder var content: () -> Content

    init(
        title: String,
        placeholder: String,
        @ViewBuilder content: @escaping () -> Content = { EmptyView() }
    ) {
        self.title = title
        self.placeholder = placeholder
        self.content = content
    }

    var body: some View {
        ZStack {
            runaTheme.background.ignoresSafeArea()

            VStack(alignment: .leading, spacing: RunaSpacing.md) {
                Text(title)
                    .font(RunaFonts.heading(28))
                    .foregroundStyle(runaTheme.heading)

                Text(placeholder)
                    .font(RunaFonts.body(16))
                    .foregroundStyle(runaTheme.subtle)

                content()

                Spacer()
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.horizontal, RunaSpacing.md)
            .padding(.top, RunaSpacing.lg)
        }
    }
}
