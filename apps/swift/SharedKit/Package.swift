// swift-tools-version:5.9
import PackageDescription

// Wraps the KMP XCFramework built by `./gradlew :shared:assembleSharedReleaseXCFramework`
// (run from apps/kotlin). For the debug variant swap `release` for `debug` in the path.
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
