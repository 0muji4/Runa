import SwiftUI

/// Runa design-system fonts.
///
/// Family names are the shared contract:
///   - headings: "Shippori Mincho"
///   - body:     "Zen Kaku Gothic New"
///   - logo:     "Cormorant Garamond"
///
/// `Font.custom(_:size:)` falls back to the system font automatically when the
/// named family is not bundled, so the app still builds and runs WITHOUT the
/// font binaries. To ship the real typefaces, drop the .ttf/.otf files into
/// Runa/Fonts/ and uncomment the UIAppFonts block in Info.plist
/// (see Runa/Fonts/README.md).
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
}
