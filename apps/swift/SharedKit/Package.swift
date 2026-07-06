// swift-tools-version:5.9
import PackageDescription

// Local Swift package that vends the shared KMP module to the iOS app.
//
// The Kotlin/Native + SKIE build produces a binary XCFramework named
// `Shared.xcframework`. We wrap it in a `.binaryTarget` and expose it as the
// library product `Shared`, so the app can `import Shared`.
//
// PATH NOTE: the path below is relative to THIS file
// (apps/swift/SharedKit/Package.swift) and resolves to
// apps/kotlin/shared/build/XCFrameworks/release/Shared.xcframework — the output
// of `./gradlew :shared:assembleSharedReleaseXCFramework` (run from apps/kotlin).
// Build the debug variant instead? Swap `release` for `debug` in the path.
let package = Package(
    name: "SharedKit",
    platforms: [
        .iOS(.v16)
    ],
    products: [
        .library(name: "Shared", targets: ["Shared"])
    ],
    targets: [
        .binaryTarget(
            name: "Shared",
            path: "../../kotlin/shared/build/XCFrameworks/release/Shared.xcframework"
        )
    ]
)
