import SwiftUI

/// Runa design-system color tokens. Dark theme ONLY.
///
/// These hex values are the shared contract and MUST stay identical to the
/// Android theme. Do not tweak per-platform.
enum RunaColors {
    /// #0E0E12 — app background.
    static let background = Color(hex: 0x0E0E12)
    /// #16161C — cards / surfaces.
    static let surface = Color(hex: 0x16161C)
    /// #F5F3EF — headings.
    static let heading = Color(hex: 0xF5F3EF)
    /// #C8C6CE — body text.
    static let body = Color(hex: 0xC8C6CE)
    /// #9A9AA5 — subtle / secondary text.
    static let subtle = Color(hex: 0x9A9AA5)
    /// #F4A9C0 — accent (moonlight).
    static let accent = Color(hex: 0xF4A9C0)
    /// #E8E2D0 — sub accent.
    static let subAccent = Color(hex: 0xE8E2D0)
}

/// Generous, minimal spacing scale (moon motif — lots of breathing room).
enum RunaSpacing {
    static let xs: CGFloat = 8
    static let sm: CGFloat = 16
    static let md: CGFloat = 24
    static let lg: CGFloat = 40
    static let xl: CGFloat = 64
}

extension Color {
    /// Builds a fully-opaque Color from a 0xRRGGBB integer literal.
    init(hex: UInt32) {
        let r = Double((hex >> 16) & 0xFF) / 255.0
        let g = Double((hex >> 8) & 0xFF) / 255.0
        let b = Double(hex & 0xFF) / 255.0
        self.init(.sRGB, red: r, green: g, blue: b, opacity: 1.0)
    }
}
