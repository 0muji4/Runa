import SwiftUI

/// Apple's official badge opening the track's Apple Music page. Apple's Promo Content terms
/// require it next to any preview or artwork; the asset is Apple's own, never redrawn.
struct AppleMusicBadge: View {
    let storeUrl: String
    var height: CGFloat = 40

    @Environment(\.openURL) private var openURL

    var body: some View {
        Button {
            if let url = URL(string: storeUrl) { openURL(url) }
        } label: {
            Image("AppleMusicBadge")
                .resizable()
                .aspectRatio(contentMode: .fit)
                .frame(height: height)
        }
        .buttonStyle(.plain)
        .accessibilityLabel(L.songListenOnAppleMusic)
    }
}

/// The attribution Apple's terms require wherever a preview is offered.
struct ITunesCourtesyLine: View {
    @Environment(\.runaTheme) private var runaTheme

    var body: some View {
        Text(L.songCourtesy)
            .font(RunaFonts.body(11))
            .foregroundStyle(runaTheme.subtle)
    }
}
