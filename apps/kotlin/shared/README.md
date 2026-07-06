# Runa — shared（KMP 共有モジュール）

Android と iOS が共有する Kotlin Multiplatform モジュール。**ロジック・API・状態管理をここに集約し、各 OS はこれを購読して描画する UI 層に徹する**。OS 固有機能は `expect`/`actual` フレームの背後に閉じ込める。

ウォーキングスケルトンの段階では、疎通確認パス（`healthz`）1 本と、各 OS 固有機能の `expect` 枠（中身は `actual` 側の TODO スタブ）だけが通っている。

## ターゲット

`androidTarget` / `iosArm64` / `iosSimulatorArm64` / `iosX64`。iOS へは SKIE を通した `Shared.xcframework` として公開する（`assembleSharedReleaseXCFramework`）。

## 公開 API（Android は直接、iOS は SKIE 経由で利用）

| シンボル | 役割 |
| --- | --- |
| `fun initKoin(baseUrl: String)` | DI 起点。各 OS が起動時に自分の Base URL（host+port のみ、`/api/v1` は含めない）を渡す |
| `fun resolveHealthzViewModel(): HealthzViewModel` | Koin グラフから VM を解決する iOS 向け入口（Android は `koinInject()`） |
| `interface ApiClient { suspend fun healthz(): HealthzResponse }` | API 呼び出しの境界。`baseUrl` に `/api/v1/healthz` を付けて叩く |
| `data class HealthzResponse(val status: String)` | `@Serializable` レスポンス DTO |
| `class HealthzViewModel { val state: StateFlow<HealthzUiState>; fun check() }` | 疎通状態を保持。`init` で `check()` を実行 |
| `sealed interface HealthzUiState` | `Loading` / `Ok(status)` / `Error(message)` |

## `expect`/`actual` フレーム（中身は各 OS の TODO スタブ）

`commonMain` に `expect`、`androidMain` / `iosMain` に `actual` を置く。現状 `httpClientEngine()` のみ実装済み（Android=OkHttp / iOS=Darwin）、他は TODO。

- `SecureStorage` — セキュアストレージ（Keychain / Keystore）
- `PushTokenProvider` — プッシュトークン取得
- `BillingClient` — 課金
- `BiometricAuthenticator` — 生体認証
- `httpClientEngine(): HttpClientEngine` — Ktor エンジン（実装済み）
- `createSqlDriver(): SqlDriver` — SQLDelight ドライバ（空スキーマ）

## モジュール構成

```
shared/src/
  commonMain/kotlin/com/runa/shared/
    di/Koin.kt                     # initKoin / resolveHealthzViewModel
    network/{ApiClient,HttpClientFactory}.kt
    network/dto/HealthzResponse.kt
    feature/health/{HealthzViewModel,HealthzUiState}.kt
    platform/Platform.kt           # expect 宣言
  commonMain/sqldelight/…/db/      # 空スキーマ
  androidMain/…/platform/Platform.android.kt   # actual（OkHttp ほか）
  iosMain/…/platform/Platform.ios.kt           # actual（Darwin ほか）
```

## ビルド・利用

- **Android**: 同一 Gradle ルート（`apps/kotlin`）の `androidApp` が `implementation(project(":shared"))` で直接利用する。手順は [../README.md](../README.md) を参照。
- **iOS**: `./gradlew :shared:assembleSharedReleaseXCFramework` で XCFramework を生成し、`apps/swift` がローカル Swift Package 経由で取り込む。手順は [../../ios/README.md](../../ios/README.md) を参照。

バージョン整合（Kotlin / Ktor / SQLDelight / Koin / SKIE 等）はリポジトリルート [../../README.md](../../README.md) の「バージョン整合」節に従う。
