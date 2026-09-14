import SwiftUI

/// Runa design-system fonts. The family strings are the registered FAMILY names inside the
/// bundled .ttf files (Runa/Fonts/); a mismatch silently falls back to the system font.
enum RunaFonts {
    private static let headingFamily = "Shippori Mincho"
    private static let bodyFamily = "Zen Kaku Gothic New"
    private static let logoFamily = "Cormorant Garamond"

    /// Heading font, scaling relative to a Dynamic Type text style.
    static func heading(_ size: CGFloat, relativeTo style: Font.TextStyle = .title) -> Font {
        Font.custom(headingFamily, size: size, relativeTo: style)
    }

    /// Body font, scaling relative to a Dynamic Type text style.
    static func body(_ size: CGFloat, relativeTo style: Font.TextStyle = .body) -> Font {
        Font.custom(bodyFamily, size: size, relativeTo: style)
    }

    /// Logo / display font.
    static func logo(_ size: CGFloat, relativeTo style: Font.TextStyle = .largeTitle) -> Font {
        Font.custom(logoFamily, size: size, relativeTo: style)
    }

    /// Screen-title style; used by `RunaScreenHeader` only.
    static let screenTitle = heading(34, relativeTo: .largeTitle)

    /// 「‹ 戻る」and header-action label style.
    static let headerLabel = body(13, relativeTo: .footnote)
}
