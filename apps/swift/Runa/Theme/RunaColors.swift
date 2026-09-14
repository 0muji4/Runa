import SwiftUI

/// The app-wide color palette (seven semantic tokens). Hex values are the shared 3-theme
/// tokens; Android carries the same set.
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

    /// あさ（ライト）.
    static let light = RunaTheme(
        background: Color(hex: 0xFAF7F5),
        surface: Color(hex: 0xFFFFFF),
        heading: Color(hex: 0x2A2620),
        body: Color(hex: 0x4E483F),
        subtle: Color(hex: 0x8C8579),
        accent: Color(hex: 0xE79CB6),
        subAccent: Color(hex: 0xC9B8A0)
    )

    /// ピンク×ピンク.
    static let pink = RunaTheme(
        background: Color(hex: 0x141017),
        surface: Color(hex: 0x1E1622),
        heading: Color(hex: 0xF6EEF2),
        body: Color(hex: 0xD6C4CE),
        subtle: Color(hex: 0xA08E99),
        accent: Color(hex: 0xF4A9C0),
        subAccent: Color(hex: 0xE8B7C8)
    )

    /// Maps the shared `AppTheme.id` string to its palette.
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
    /// The active app palette, injected once at the root by `ThemedRoot`.
    var runaTheme: RunaTheme {
        get { self[RunaThemeKey.self] }
        set { self[RunaThemeKey.self] = newValue }
    }
}

/// Spacing scale.
enum RunaSpacing {
    static let xs: CGFloat = 8
    static let sm: CGFloat = 16
    static let md: CGFloat = 24
    static let lg: CGFloat = 40
    static let xl: CGFloat = 64
}

/// Header metrics; every screen header is built from these by `RunaScreenHeader`, screens
/// do not set their own.
enum RunaHeaderMetrics {
    /// Top of a bottom-tab root's header, inside the safe area.
    static let topTab: CGFloat = 40

    /// Top of a pushed screen's「‹ 戻る」row.
    static let topPushed: CGFloat = 14

    /// 「‹ 戻る」to the title.
    static let backGap: CGFloat = 24

    /// Title to the body below it.
    static let bottom: CGFloat = 24
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
