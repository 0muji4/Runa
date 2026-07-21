# Runa — Kotlin（KMP shared + Android app）

This Gradle root hosts the Kotlin Multiplatform **shared** module and the Android
**Jetpack Compose** app for Runa (a moon-themed walking app). It is a walking
skeleton: full DI + navigation wiring, a single `GET /api/v1/healthz` round trip,
and empty tab shells — **no product features yet**.

The iOS app consumes the same shared module as an XCFramework; the Xcode project
itself is not in this Gradle root (it lives alongside, added when iOS UI work
begins).

## Layout

```
apps/kotlin/
├── settings.gradle.kts          # includes :shared and :androidApp
├── build.gradle.kts             # plugins declared apply-false (versions from catalog)
├── gradle.properties            # jvmargs, AndroidX, Kotlin/Native flags
├── gradle/
│   ├── libs.versions.toml       # single source of truth for versions
│   └── wrapper/
│       └── gradle-wrapper.properties   # pinned to gradle-8.9-bin.zip
├── shared/                      # KMP module (com.runa.shared)
│   └── src/
│       ├── commonMain/          # DI, network, health feature, expect declarations, .sq schema
│       ├── androidMain/         # OkHttp engine + platform actual stubs
│       └── iosMain/             # Darwin engine + platform actual stubs
└── androidApp/                  # Android UI only (com.runa.android, applicationId com.runa)
    └── src/main/
        ├── kotlin/com/runa/android/   # Application, Activity, theme, navigation, screens
        └── res/                       # strings, dark theme, XML adaptive launcher icon (fonts: see androidApp/FONTS.md)
```

## First-time setup

The Gradle wrapper JAR and `gradlew`/`gradlew.bat` scripts are **not committed**
(binary). Generate them once with your system Gradle (Homebrew `gradle` works):

```bash
cd apps/kotlin
gradle wrapper --gradle-version 8.9
```

After that, use `./gradlew ...` for all builds.

> A `local.properties` with `sdk.dir=/path/to/Android/sdk` is required for the
> Android build (created automatically when you open the project in Android
> Studio, or write it yourself). It is git-ignored.

## Build & run — Android

Requires the Android SDK (compileSdk 34).

- Open `apps/kotlin/` in **Android Studio** (Giraffe+ / a build compatible with
  AGP 8.5.2), let it sync, then run the `androidApp` configuration on an
  emulator or device.
- Or from the CLI:

  ```bash
  ./gradlew :androidApp:assembleDebug
  ```

The app injects the dev base URL `http://10.0.2.2:8080` (the emulator's alias for
the host loopback) via `BuildConfig.BASE_URL`. Start the backend locally on
`:8080` and the Home tab shows **接続OK**; otherwise **接続エラー** with the message.

## Build — iOS shared framework

Assemble the XCFramework the iOS app links against:

```bash
./gradlew :shared:assembleSharedXCFramework
```

(The `XCFramework("Shared")` config also produces per-target
`link*` / `assemble*` tasks. iOS injects `http://localhost:8080` as its base URL.)

## API contract

- Base URL passed to `initKoin` is **host+port only**, without `/api/v1`.
- The shared `ApiClient` appends `/api/v1/healthz` itself.
- Health: `GET /api/v1/healthz` → `200 {"status":"ok"}`, `application/json`.

## Where feature code goes (縦切り / vertical slices)

Each tab is an empty shell today. A feature slice adds, per feature:

- **shared**: `feature/<name>/` (UI state + view model), DTOs under
  `network/dto/`, new methods on `ApiClient`, Koin registrations in `di/Koin.kt`,
  and real tables in `commonMain/sqldelight/.../db/` if it needs persistence (the
  `SqlDriver` is already bound per-platform in `platformModule()` — see below).
- **androidApp**: replace the `PlaceholderScreen(...)` call in the matching
  `ui/screens/*Screen.kt` with the real UI; add routes/args in
  `navigation/RunaApp.kt` if the slice needs sub-screens.

The remaining platform `expect` frames (`PushTokenProvider`, `BillingClient`) are
declared but their actuals are `TODO` stubs — fill them in when a slice first needs
them. Secure storage is implemented by the auth slice; the SQLDelight driver +
audio player are bound in `platformModule()` by the today slice; and the
notification/lock slice adds the `LocalNotificationScheduler` + `BiometricAuthenticator`
platform bindings (see below). `BiometricAuthenticator` moved from an `expect class`
stub to a `feature/lock/` common interface bound in `platformModule()` (it needs the
current Activity on Android).

## Auth slice

The first product feature (authentication) is wired end to end. Shared pieces:

- `feature/auth/`: `AuthRepository` (`signupEmail` / `loginEmail` /
  `loginApple` / `loginGoogle` / `refresh` / `logout` / `getMe` /
  `restoreSession`), `AuthViewModel`, and `AuthState`
  (`Restoring` / `Unauthenticated` / `Authenticating` / `Authenticated(user)` /
  `Error`). Subscribe to `AuthViewModel.state` (or `AuthRepository.authState`).
- `network/auth/`: `TokenStore` + `SecureKeyValueStore` (secure token
  persistence) and the Ktor `HttpSend` interceptor that auto-refreshes on 401 and
  replays the request. Android backs the store with **EncryptedSharedPreferences**,
  iOS with the **Keychain** (via multiplatform-settings).
- **DI**: Android calls the `initKoin(context, baseUrl)` overload
  (`di/Koin.android.kt`) because the secure store needs a `Context`; iOS keeps
  `initKoin(baseUrl)`.
- **Tests**: `./gradlew :shared:testDebugUnitTest` runs `commonTest` on the JVM
  (401 auto-refresh + replay, refresh failure, token restore, each sign-in path)
  with a Ktor `MockEngine`.

### Native sign-in config (Android)

The app gets an ID token natively and passes it to the shared repository; the
backend verifies it. **Email + password works with no extra config.** Set these
Gradle properties (e.g. in `~/.gradle/gradle.properties` or `-P` flags) to enable
the social buttons:

| Property | Purpose |
| --- | --- |
| `RUNA_GOOGLE_SERVER_CLIENT_ID` | Google **Web** OAuth client ID — Credential Manager's `serverClientId` (also a backend `GOOGLE_CLIENT_IDS` audience). |
| `RUNA_APPLE_SERVICE_ID` | Apple **Service ID** for the Sign in with Apple web flow. |
| `RUNA_APPLE_REDIRECT_URI` | The Apple web-flow https redirect you control. |

Google uses Credential Manager (`androidx.credentials` + `googleid`). Apple on
Android uses a Custom Tabs web flow; completing its redirect round trip requires
the backend redirect endpoint and is finished once real credentials exist.

## Today slice

The third feature ("today") powers the home screen. Its core is the **moon-phase
calculator** — the reason this slice matters.

- **`feature/today/moon/MoonPhaseCalculator.kt`** — a pure `commonMain`,
  `kotlin.math`-only calculator (Meeus-simplified moon-age method; synodic month
  `29.530588853` d, epoch `JD 2451550.1`). Because it uses no platform APIs, Android
  and iOS run identical IEEE-754 math and return identical results. This is proven
  by `commonTest/.../MoonPhaseCalculatorTest.kt` against real new/full/quarter
  moons — a green run on any target is the cross-platform guarantee. The moon works
  fully **offline** (nothing is fetched). Glyph + Japanese name live in shared
  `feature/today/moon/MoonPhasePresentation.kt` so both apps render the same.
- **Repositories** (`feature/today/`): `TodayRepository.getToday()` fetches the
  day's quote/song, caches them in SQLDelight, and composes them with the computed
  moon — falling back to the cache (plus the still-computed moon) when offline.
  `SongRepository` reads the archive, records/observes play history.
- **SQLDelight is now wired** (first real use): tables in
  `commonMain/sqldelight/.../db/Today.sq`, and the `SqlDriver` is bound in each
  `platformModule()` (Android `AndroidSqliteDriver`, iOS `NativeSqliteDriver`) with
  `RunaDatabase` in `sharedModule`.
- **Audio playback is a platform actual** behind the common `AudioPlayer` interface
  (`feature/today/player/`): `ExoAudioPlayer` (Media3) on Android, `AvAudioPlayer`
  (AVFoundation) on iOS, both bound in `platformModule()`. The shared
  `SongPlayerViewModel` owns the play/pause/seek intent + current-song state; the
  UI only drives it. `HomeViewModel` / `SongArchiveViewModel` complete the trio.
- **Tests**: `commonTest` (moon reference dates) plus `androidUnitTest`
  (`TodayRepositoryTest`, `SongRepositoryTest`, `SongPlayerViewModelTest`) which use
  a JVM in-memory SQLite driver + Ktor `MockEngine`.

## Notification / privacy-lock slice (OS-native, slice 8)

The nightly reminder and biometric lock are OS-feature-heavy: `shared` holds the
settings + state, and each platform's actual does the real scheduling / auth.

- **Reminder settings** live in `feature/notification/`: `NotificationSettingsRepository`
  (persist `reminderEnabled` / `reminderTime` via multiplatform-settings, observe as
  `StateFlow`, and drive the `LocalNotificationScheduler`), `NotificationSettingsViewModel`,
  and the shared poetic copy (`ReminderNotificationText`). Time is a pre-formatted
  `ReminderTime` (`label` = "HH:MM") so the UI needs no kotlinx-datetime.
- **Android reminder actual** (`shared/androidMain/.../feature/notification/`):
  `AndroidLocalNotificationScheduler` arms `AlarmManager.setAndAllowWhileIdle`
  (inexact, so **no `SCHEDULE_EXACT_ALARM`**); `ReminderReceiver` posts the
  notification (`NotificationManagerCompat` + a channel, `androidx.core`) and re-arms
  the next day; `BootReceiver` re-arms after reboot. Both receivers + the
  `POST_NOTIFICATIONS` / `RECEIVE_BOOT_COMPLETED` permissions are declared in the
  **shared androidMain `AndroidManifest.xml`** (merged into the app); the small icon
  is `shared/androidMain/res/drawable/ic_reminder_moon.xml`. The runtime
  `POST_NOTIFICATIONS` request (API 33+) happens in onboarding ④
  (`NotificationPermissionScreen`) and when enabling in 通知設定 (21).
- **Privacy lock** lives in `feature/lock/`: `AppLockRepository` (persist
  `lockEnabled`) and `AppLockViewModel` (states `Unlocked/Locked/Authenticating/Unavailable`)
  — a layer **separate from auth**. `BiometricAuthenticator` is a common interface
  (not `expect class`) bound in `platformModule()`. **Android actual**:
  `AndroidBiometricAuthenticator` uses `androidx.biometric` `BiometricPrompt` with
  `BIOMETRIC_STRONG or DEVICE_CREDENTIAL` (device-passcode fallback); it reads the
  current Activity from `CurrentActivityHolder`, which `MainActivity` (now a
  **`FragmentActivity`**, BiometricPrompt's requirement) registers from its
  lifecycle. `MainActivity` also wraps the app in `AppLockGate` and drives
  foreground/background from `onResume`/`onPause`.
- **Deps added** (`androidMain` of `:shared`): `androidx.biometric`, `androidx.core`;
  (`:androidApp`): `androidx.fragment`.
- **Tests**: `commonTest` covers persistence + schedule instructions
  (`NotificationSettingsRepositoryTest`, `AppLockRepositoryTest`) and the state
  machine with a fake authenticator (`AppLockViewModelTest`,
  `NotificationSettingsViewModelTest`).

## Fonts

Design fonts (Shippori Mincho / Zen Kaku Gothic New / Cormorant Garamond) are not
committed. See `androidApp/FONTS.md` to enable them; until
then the theme falls back to the system font.

## Version matrix (pinned starting point)

Defined in `gradle/libs.versions.toml`. **These may need local alignment** with
your installed Android Studio / Kotlin / SDK — treat them as a coherent starting
set, not gospel.

| Component | Version |
|---|---|
| Kotlin | 2.0.21 |
| Android Gradle Plugin | 8.5.2 |
| Gradle wrapper | 8.9 |
| JDK | 17 |
| compileSdk / minSdk / targetSdk | 34 / 26 / 34 |
| Ktor | 3.0.1 |
| kotlinx-coroutines | 1.9.0 |
| kotlinx-serialization-json | 1.7.3 |
| kotlinx-datetime | 0.6.1 |
| SQLDelight | 2.0.2 |
| Koin | 4.0.0 |
| multiplatform-settings | 1.2.0 |
| SKIE | 0.10.1 |
| Compose BOM | 2024.10.01 |
| activity-compose | 1.9.3 |
| navigation-compose | 2.8.3 |
| lifecycle-viewmodel-compose | 2.8.6 |
| Kotlin compose plugin | 2.0.21 |
| iOS deployment target | 16.0 |
