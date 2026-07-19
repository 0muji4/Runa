import SwiftUI

/// Runa moon motif, drawn with SwiftUI `Canvas` — NO SF Symbols, NO emoji, NO text
/// glyphs. Mirrors the Android `MoonArt.kt` so both platforms share one world-view:
/// a glowing hero moon, a phase disc for diary records, and the empty / offline /
/// error / notification emblems.

// The moon is a FIXED cross-theme motif (design decision): it never recolors with
// the app theme, so it holds its own dark-motif constants rather than reading the
// theme tokens. Values match the original dark palette.
private let moonLit = Color(hex: 0xF7F2E4)
private let moonCream = Color(hex: 0xE8E2D0)
private let moonDark = Color(hex: 0x34343E)
private let moonRing = Color(hex: 0x3E3E48)
private let moonMuted = Color(hex: 0x9A9AA5)      // subtle
private let moonAccent = Color(hex: 0xF4A9C0)     // moonlight accent
private let moonBackground = Color(hex: 0x0E0E12) // background

private func drawGlow(_ ctx: inout GraphicsContext, center: CGPoint, radius: CGFloat, tint: Color, alpha: Double) {
    guard radius > 0 else { return }
    let rect = CGRect(x: center.x - radius, y: center.y - radius, width: radius * 2, height: radius * 2)
    let shading = GraphicsContext.Shading.radialGradient(
        Gradient(colors: [tint.opacity(alpha), .clear]),
        center: center, startRadius: 0, endRadius: radius
    )
    ctx.fill(Path(ellipseIn: rect), with: shading)
}

private func disc(_ center: CGPoint, _ r: CGFloat) -> Path {
    Path(ellipseIn: CGRect(x: center.x - r, y: center.y - r, width: r * 2, height: r * 2))
}

/// A full, softly glowing moon — the hero mark for sign-in / onboarding / splash.
struct GlowingMoon: View {
    var diameter: CGFloat = 132
    var haloTint: Color = moonCream

    var body: some View {
        Canvas { ctx, size in
            let c = CGPoint(x: size.width / 2, y: size.height / 2)
            let moonR = min(size.width, size.height) * 0.30
            drawGlow(&ctx, center: c, radius: min(size.width, size.height) * 0.5, tint: haloTint, alpha: 0.22)
            drawGlow(&ctx, center: c, radius: moonR * 1.7, tint: moonLit, alpha: 0.30)
            let shading = GraphicsContext.Shading.radialGradient(
                Gradient(colors: [moonLit, moonCream]),
                center: CGPoint(x: c.x - moonR * 0.28, y: c.y - moonR * 0.28),
                startRadius: 0, endRadius: moonR * 1.5
            )
            ctx.fill(disc(c, moonR), with: shading)
        }
        .frame(width: diameter, height: diameter)
        .accessibilityHidden(true)
    }
}

/// A small moon-phase disc for diary cards / detail. [illumination] 0..1 is the lit
/// fraction; [waxing] puts the lit limb on the right. Offset-shadow method.
struct MoonPhaseDisc: View {
    var illumination: CGFloat
    var waxing: Bool
    var diameter: CGFloat = 22

    var body: some View {
        Canvas { ctx, size in
            let c = CGPoint(x: size.width / 2, y: size.height / 2)
            let r = min(size.width, size.height) / 2 * 0.92
            let f = max(0, min(1, illumination))
            if f > 0.55 { drawGlow(&ctx, center: c, radius: r * 1.9, tint: moonCream, alpha: 0.18 * Double(f)) }
            var inner = ctx
            let moon = disc(c, r)
            inner.clip(to: moon)
            inner.fill(moon, with: .color(moonCream))
            let dir: CGFloat = waxing ? -1 : 1 // shadow slides toward the dark limb
            let shadowX = c.x + dir * 2 * r * f
            inner.fill(disc(CGPoint(x: shadowX, y: c.y), r), with: .color(moonDark))
            ctx.stroke(moon, with: .color(moonRing), lineWidth: 1.2)
        }
        .frame(width: diameter, height: diameter)
        .accessibilityHidden(true)
    }
}

/// New-moon emblem for the empty state (24): a dark disc inside a faint ring.
struct NewMoonEmblem: View {
    var diameter: CGFloat = 108
    var body: some View {
        Canvas { ctx, size in
            let c = CGPoint(x: size.width / 2, y: size.height / 2)
            let r = min(size.width, size.height) * 0.32
            drawGlow(&ctx, center: c, radius: min(size.width, size.height) * 0.46, tint: moonCream, alpha: 0.10)
            ctx.stroke(disc(c, r * 1.32), with: .color(moonRing), lineWidth: 1.4)
            let shading = GraphicsContext.Shading.radialGradient(
                Gradient(colors: [Color(hex: 0x26262E), Color(hex: 0x14141A)]),
                center: CGPoint(x: c.x - r * 0.2, y: c.y - r * 0.2),
                startRadius: 0, endRadius: r * 1.3
            )
            ctx.fill(disc(c, r), with: shading)
        }
        .frame(width: diameter, height: diameter)
        .accessibilityHidden(true)
    }
}

/// Clouded moon for the offline state (25): a dim disc crossed by a soft stroke.
struct CloudedMoon: View {
    var diameter: CGFloat = 108
    var body: some View {
        Canvas { ctx, size in
            let c = CGPoint(x: size.width / 2, y: size.height / 2)
            let r = min(size.width, size.height) * 0.30
            let shading = GraphicsContext.Shading.radialGradient(
                Gradient(colors: [Color(hex: 0x3A3A44), Color(hex: 0x24242C)]),
                center: CGPoint(x: c.x - r * 0.25, y: c.y - r * 0.25),
                startRadius: 0, endRadius: r * 1.4
            )
            ctx.fill(disc(c, r), with: shading)
            var line = Path()
            line.move(to: CGPoint(x: c.x - r * 1.15, y: c.y + r * 0.8))
            line.addLine(to: CGPoint(x: c.x + r * 1.15, y: c.y - r * 0.8))
            ctx.stroke(line, with: .color(moonMuted), style: StrokeStyle(lineWidth: 2, lineCap: .round))
        }
        .frame(width: diameter, height: diameter)
        .accessibilityHidden(true)
    }
}

/// Gentle "stumble" emblem for the error state (27): a soft disc holding an ! mark.
struct StumbleEmblem: View {
    var diameter: CGFloat = 108
    var body: some View {
        Canvas { ctx, size in
            let c = CGPoint(x: size.width / 2, y: size.height / 2)
            let r = min(size.width, size.height) * 0.30
            drawGlow(&ctx, center: c, radius: min(size.width, size.height) * 0.44, tint: moonMuted, alpha: 0.12)
            let shading = GraphicsContext.Shading.radialGradient(
                Gradient(colors: [Color(hex: 0xB9B4AC), Color(hex: 0x8F8B84)]),
                center: CGPoint(x: c.x - r * 0.25, y: c.y - r * 0.25),
                startRadius: 0, endRadius: r * 1.4
            )
            ctx.fill(disc(c, r), with: shading)
            var stem = Path()
            stem.move(to: CGPoint(x: c.x, y: c.y - r * 0.42))
            stem.addLine(to: CGPoint(x: c.x, y: c.y + r * 0.10))
            ctx.stroke(stem, with: .color(moonBackground), style: StrokeStyle(lineWidth: r * 0.16, lineCap: .round))
            ctx.fill(disc(CGPoint(x: c.x, y: c.y + r * 0.40), r * 0.09), with: .color(moonBackground))
        }
        .frame(width: diameter, height: diameter)
        .accessibilityHidden(true)
    }
}

/// The notification mark (04): the glowing moon with a small moonlight-pink badge
/// carrying a minimal drawn bell.
struct NotificationMoon: View {
    var diameter: CGFloat = 132
    var body: some View {
        Canvas { ctx, size in
            let c = CGPoint(x: size.width / 2, y: size.height / 2)
            let moonR = min(size.width, size.height) * 0.28
            drawGlow(&ctx, center: c, radius: min(size.width, size.height) * 0.5, tint: moonCream, alpha: 0.20)
            drawGlow(&ctx, center: c, radius: moonR * 1.7, tint: moonLit, alpha: 0.28)
            let shading = GraphicsContext.Shading.radialGradient(
                Gradient(colors: [moonLit, moonCream]),
                center: CGPoint(x: c.x - moonR * 0.28, y: c.y - moonR * 0.28),
                startRadius: 0, endRadius: moonR * 1.5
            )
            ctx.fill(disc(c, moonR), with: shading)

            let badgeC = CGPoint(x: c.x + moonR * 0.95, y: c.y - moonR * 0.95)
            let badgeR = moonR * 0.6
            drawGlow(&ctx, center: badgeC, radius: badgeR * 2.1, tint: moonAccent, alpha: 0.35)
            ctx.fill(disc(badgeC, badgeR), with: .color(moonAccent))
            drawBell(&ctx, center: badgeC, s: badgeR * 0.92, color: moonBackground)
        }
        .frame(width: diameter, height: diameter)
        .accessibilityHidden(true)
    }

    private func drawBell(_ ctx: inout GraphicsContext, center c: CGPoint, s: CGFloat, color: Color) {
        let w = s * 0.9, h = s * 1.0
        var body = Path()
        body.move(to: CGPoint(x: c.x - w * 0.5, y: c.y + h * 0.28))
        body.addCurve(
            to: CGPoint(x: c.x, y: c.y - h * 0.5),
            control1: CGPoint(x: c.x - w * 0.5, y: c.y - h * 0.15),
            control2: CGPoint(x: c.x - w * 0.32, y: c.y - h * 0.5)
        )
        body.addCurve(
            to: CGPoint(x: c.x + w * 0.5, y: c.y + h * 0.28),
            control1: CGPoint(x: c.x + w * 0.32, y: c.y - h * 0.5),
            control2: CGPoint(x: c.x + w * 0.5, y: c.y - h * 0.15)
        )
        body.closeSubpath()
        ctx.fill(body, with: .color(color))
        var base = Path()
        base.move(to: CGPoint(x: c.x - w * 0.6, y: c.y + h * 0.30))
        base.addLine(to: CGPoint(x: c.x + w * 0.6, y: c.y + h * 0.30))
        ctx.stroke(base, with: .color(color), style: StrokeStyle(lineWidth: s * 0.14, lineCap: .round))
        ctx.fill(disc(CGPoint(x: c.x, y: c.y + h * 0.5), s * 0.12), with: .color(color))
    }
}
