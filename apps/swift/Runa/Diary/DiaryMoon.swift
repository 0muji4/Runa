import CoreGraphics
import Foundation

/// The moon phase for a diary entry's day. Kept self-contained on the Swift side
/// (no Kotlin/KMM call needed) but a faithful port of the shared
/// `MoonPhaseCalculator` — same constants, same buckets — so it matches Android's
/// diary moons. Purely offline.
struct DiaryMoon {
    let illumination: CGFloat // 0 (new) .. 1 (full)
    let waxing: Bool          // lit limb on the right
    let name: String          // shared Japanese phase name
}

enum DiaryMoonCalc {
    private static let synodicMonth = 29.530588853
    private static let referenceNewMoonJD = 2_451_550.1
    private static let unixEpochJD = 2_440_587.5
    private static let millisPerDay = 86_400_000.0

    /// Synodic order, matching `MoonPhaseKey` / `moonPhaseNameJa` in shared.
    private static let names = [
        "新月", "三日月", "上弦の月", "十三夜", "満月", "寝待月", "下弦の月", "有明月",
    ]

    static func moon(epochMs: Int64) -> DiaryMoon {
        var cal = Calendar(identifier: .gregorian)
        cal.timeZone = .current
        let date = Date(timeIntervalSince1970: Double(epochMs) / 1000)
        // Represent the day at local noon — a stable mid-day instant (mirrors shared).
        var comps = cal.dateComponents([.year, .month, .day], from: date)
        comps.hour = 12
        comps.minute = 0
        let noon = cal.date(from: comps) ?? date
        let julianDay = noon.timeIntervalSince1970 * 1000 / millisPerDay + unixEpochJD

        var age = (julianDay - referenceNewMoonJD).truncatingRemainder(dividingBy: synodicMonth)
        if age < 0 { age += synodicMonth }
        let fraction = age / synodicMonth
        let illumination = max(0, min(1, (1 - cos(2 * Double.pi * fraction)) / 2))
        let index = Int(floor(fraction * 8 + 0.5)) % 8

        return DiaryMoon(
            illumination: CGFloat(illumination),
            waxing: age < synodicMonth / 2,
            name: names[index]
        )
    }
}
