import SwiftUI

/// The app-wide color palette (the seven semantic tokens). A theme change swaps the
/// whole `RunaTheme`; screens read it from the environment as `@Environment(\.runaTheme)`
/// so a switch recolors them in place (nav state preserved, live preview on the
/// theme screen). Hex values are IDENTICAL to Android and the README canonical table.
struct RunaTheme {
    let background: Color
    let surface: Color
    let heading: Color
    let body: Color
    let subtle: Color
    let accent: Color
    let subAccent: Color

    /// 夜（ダーク・既定）.
    static let dark = RunaTheme(
        background: Color(hex: 0x0E0E12),
        surface: Color(hex: 0x16161C),
        heading: Color(hex: 0xF5F3EF),
        body: Color(hex: 0xC8C6CE),
        subtle: Color(hex: 0x9A9AA5),
        accent: Color(hex: 0xF4A9C0),
        subAccent: Color(hex: 0xE8E2D0)
    )

    /// あさ（ライト）. Values beyond the cream background are derived from the design
    /// swatch + spec (pending sign-off).
    static let light = RunaTheme(
        background: Color(hex: 0xFAF7F5),
        surface: Color(hex: 0xFFFFFF),
        heading: Color(hex: 0x2A2620),
        body: Color(hex: 0x4E483F),
        subtle: Color(hex: 0x8C8579),
        accent: Color(hex: 0xE79CB6),
        subAccent: Color(hex: 0xC9B8A0)
    )

    /// ピンク×ピンク. Dark base with the accent pushed further (derived, pending sign-off).
    static let pink = RunaTheme(
        background: Color(hex: 0x141017),
        surface: Color(hex: 0x1E1622),
        heading: Color(hex: 0xF6EEF2),
        body: Color(hex: 0xD6C4CE),
        subtle: Color(hex: 0xA08E99),
        accent: Color(hex: 0xF4A9C0),
        subAccent: Color(hex: 0xE8B7C8)
    )

    /// Maps the shared `AppTheme.id` string to its palette. Using the id keeps this
    /// free of the bridged Kotlin enum's case names.
    static func forId(_ id: String) -> RunaTheme {
        switch id {
        case "light": return .light
        case "pink": return .pink
        default: return .dark
        }
    }
}

private struct RunaThemeKey: EnvironmentKey {
    static let defaultValue: RunaTheme = .dark
}

extension EnvironmentValues {
    /// The active app palette. Injected once at the root (see `ThemedRoot`); read by
    /// every screen as `@Environment(\.runaTheme) private var runaTheme`.
    var runaTheme: RunaTheme {
        get { self[RunaThemeKey.self] }
        set { self[RunaThemeKey.self] = newValue }
    }
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
