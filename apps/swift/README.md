# Runa — iOS app

SwiftUI (iOS 16+) shell for Runa. It consumes the shared Kotlin Multiplatform
module as a **SKIE-enhanced XCFramework**, imported through a local Swift package
(`SharedKit`). The Xcode project is generated reproducibly with **XcodeGen** —
no `.xcodeproj` is committed.

## Prerequisites

- Xcode 15+ with an iOS 16+ simulator
- [XcodeGen](https://github.com/yonaskolb/XcodeGen) (`brew install xcodegen`)
- The KMP toolchain (JDK 17 + the `apps/kotlin` Gradle project) to build the
  shared framework

## Build & run

### 1. Build the shared XCFramework (from `apps/kotlin`)

The iOS app links a binary `Shared.xcframework` produced by the KMP + SKIE build.
Build it first:

```bash
cd apps/kotlin
./gradlew :shared:assembleSharedReleaseXCFramework
```

This is expected to emit:

```
apps/kotlin/shared/build/XCFrameworks/release/Shared.xcframework
```

which is exactly the path referenced by `apps/swift/SharedKit/Package.swift`.

> **Path check:** the assemble task name and output folder depend on the
> `apps/kotlin` module's `XCFramework { }` DSL (that module is authored by the
> shared/KMP layer). If your task differs (e.g. a debug variant, or a custom
> output dir), build it, note where the `.xcframework` lands, and update the
> `path:` in `SharedKit/Package.swift` (and this section) to match.

### 2. Generate the Xcode project

```bash
cd apps/swift
xcodegen generate
```

This reads `project.yml` and produces `Runa.xcodeproj` (git-ignored).

### 3. Open the project

```bash
open Runa.xcodeproj
```

### 4. Run on an iOS 16+ simulator

Select the `Runa` scheme and an iOS 16+ simulator, then Run.

### 5. Confirm connectivity

Start the backend so it listens on `:8080` (serving
`GET /api/v1/healthz -> {"status":"ok"}`). The iOS **simulator** reaches the host
machine via `http://localhost:8080` (already set as `BASE_URL` in `Info.plist`;
the shared module appends `/api/v1/healthz`).

On the **Home** tab you should see:

- a spinner while `Loading`
- **接続OK** once the health check succeeds
- **接続エラー** plus the message on failure

## How the shared module is consumed (SKIE)

[SKIE](https://skie.touchlab.co/) enhances the Kotlin→Swift interop so that:

- Kotlin `suspend` functions become Swift `async`/`throws`
- Kotlin `Flow` / `StateFlow` become Swift `AsyncSequence` (iterable with
  `for await`)
- Kotlin `sealed` types become native Swift `enum`s (exhaustive `switch`)

`HomeView.swift` relies on all three: `HealthzObservable` collects the exposed
`StateFlow<HealthzUiState>` and `switch`es over the sealed `HealthzUiState`.

### Verified SKIE symbol mappings

Reconciled against a real XCFramework build (`import Shared`):

- `initKoin(baseUrl:)` → global Swift func **`doInitKoin(baseUrl:)`** — Kotlin/Native
  prefixes `init`-named functions with `do`. Used in `RunaApp.swift`.
- `resolveHealthzViewModel()` → global Swift func **`resolveHealthzViewModel()`**
  (SKIE exposes top-level Kotlin funcs as global Swift funcs, not `KoinKt.*`).
- `HealthzViewModel.state` (`StateFlow`) bridges to **`SkieSwiftStateFlow<HealthzUiState>`**
  (an `AsyncSequence`); iterate with `for await`. Used in `HomeView.swift`.
- The sealed `HealthzUiState` is matched via SKIE's **`onEnum(of:)`** →
  `.loading` / `.ok(HealthzUiStateOk)` / `.error(HealthzUiStateError)`
  (with `.status` / `.message`).

If you regenerate the framework with a different SKIE/Kotlin version and a symbol
moves, open the generated `Shared` Swift interface and re-reconcile.

## Fonts

Custom fonts are not committed. See [`Runa/Fonts/README.md`](Runa/Fonts/README.md).
`Font.custom` falls back to the system font until the files are added, so the app
builds without them.

## Version notes

Deployment target is iOS 16.0. The shared framework must be built against a
compatible KMP/SKIE toolchain (see the repo-root version matrix); versions may
need local alignment.
