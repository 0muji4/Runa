import SwiftUI

/// The one screen header. Every screen that carries a title uses this and writes no
/// header of its own, so the title size, the back affordance and the vertical rhythm
/// are identical everywhere (README「画面ヘッダー（全画面共通の型）」is the canon).
///
/// `onBack` decides the variant: `nil` means a bottom-tab root (starts at
/// `RunaHeaderMetrics.topTab`, no back row), otherwise a pushed screen (starts at
/// `topPushed` with the「‹ 戻る」row above the title). Pushed screens hide the system
/// navigation bar and pass their own dismiss here, so the affordance reads the same
/// on both platforms.
///
/// `title` is nil on the one pushed screen that carries no screen title (the
/// retrospective calendar, whose month stepper is content rather than a heading), so
/// it still gets the same back affordance and the same top offset as its siblings.
///
/// Horizontal padding is deliberately absent — the caller's container supplies it, so
/// the title always lines up with the body beneath it rather than with a value chosen
/// here.
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
