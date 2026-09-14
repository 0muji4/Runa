import SwiftUI

/// The one screen header; screens write no header of their own. `onBack == nil` means a
/// bottom-tab root, otherwise a pushed screen with the「‹ 戻る」row. No horizontal padding:
/// the caller's container supplies it.
struct RunaScreenHeader<Actions: View>: View {
    @Environment(\.runaTheme) private var runaTheme

    let title: String?
    let onBack: (() -> Void)?
    @ViewBuilder var actions: () -> Actions

    init(
        title: String? = nil,
        onBack: (() -> Void)? = nil,
        @ViewBuilder actions: @escaping () -> Actions = { EmptyView() }
    ) {
        self.title = title
        self.onBack = onBack
        self.actions = actions
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            if let onBack {
                Text("‹ " + L.actionBack)
                    .font(RunaFonts.headerLabel)
                    .foregroundStyle(runaTheme.subtle)
                    // Widens the touch target without moving the glyph off the margin.
                    .padding(.vertical, 6)
                    .padding(.trailing, 12)
                    .contentShape(Rectangle())
                    .onTapGesture(perform: onBack)
                    .padding(.top, RunaHeaderMetrics.topPushed)

                if title != nil { Spacer().frame(height: RunaHeaderMetrics.backGap) }
            }

            if let title {
                HStack(alignment: .firstTextBaseline) {
                    Text(title)
                        .font(RunaFonts.screenTitle)
                        .foregroundStyle(runaTheme.heading)
                    Spacer()
                    actions()
                }
                .padding(.top, onBack == nil ? RunaHeaderMetrics.topTab : 0)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.bottom, RunaHeaderMetrics.bottom)
    }
}
