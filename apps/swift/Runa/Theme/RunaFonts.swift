import SwiftUI

/// Runa design-system fonts.
///
/// Family names are the shared contract:
///   - headings: "Shippori Mincho"
///   - body:     "Zen Kaku Gothic New"
///   - logo:     "Cormorant Garamond"
///
/// The OFL binaries are bundled under Runa/Fonts/ and registered via `UIAppFonts`
/// in Info.plist (XcodeGen copies the folder's files as bundle resources). The
/// strings below are the registered FAMILY names — they must match the family name
/// inside each .ttf (verified: "Shippori Mincho", "Zen Kaku Gothic New",
/// "Cormorant Garamond"). `Font.custom(_:size:)` still falls back to the system
/// font gracefully if a family is ever missing. See Runa/Fonts/README.md.
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
