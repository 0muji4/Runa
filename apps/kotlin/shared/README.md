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

- `PushTokenProvider` — プッシュトークン取得
- `BillingClient` — 課金
- `BiometricAuthenticator` — 生体認証
- `httpClientEngine(): HttpClientEngine` — Ktor エンジン（実装済み）
- `platformModule()` — 各 OS の Koin 束縛。`SecureKeyValueStore`（Keychain / EncryptedSharedPreferences）に加え、SQLDelight `SqlDriver`、`NetworkMonitor`（Android=`ConnectivityManager` / iOS=`NWPathMonitor`）、音声再生 `AudioPlayer`（Android=ExoPlayer / iOS=AVPlayer）を提供する。`SqlDriver`/`AudioPlayer` は `expect` 関数ではなく `platformModule` で OS ごとに束ねる

## ダイアリー（オフラインファースト同期）

2 本目の縦スライス。UI は**ローカル DB（SQLDelight）だけ**を購読し、ネットワークは `sync()` の中でのみ触る。

| シンボル | 役割 |
| --- | --- |
| `interface DiaryRepository` | `observeEntries(): Flow<List<DiaryEntry>>`（ローカル購読・即描画）／`createEntry` / `updateEntry` / `deleteEntry`（ローカル先行 → `pending_*`）／`sync()`／`syncStatus: StateFlow<SyncStatus>` |
| `class DiaryListViewModel { val state: StateFlow<DiaryListState>; fun refresh(); fun delete(clientId) }` | 一覧。`DiaryListState = Loading / Content(entries, banner) / Empty(banner)`、`SyncBanner = None/Syncing/Offline/Error` |
| `class DiaryEditorViewModel { val state: StateFlow<DiaryEditorState>; fun onBodyChange; fun onMoodChange; fun saveNow() }` | 書く。`SaveStatus = Editing/Saving/Saved/Error`。初回の非空変更で `pending_create` 作成 → debounce 自動保存 |
| `fun resolveDiaryListViewModel()` / `fun resolveDiaryEditorViewModel(clientId: String?)` | iOS（SKIE）向け解決入口。Android は `koinInject`（エディタは `parametersOf(clientId)`） |

**同期方針**（サーバ契約は [../../go/README.md](../../go/README.md) を参照）:

- **`client_id` 冪等**: 作成時にクライアントが UUID を採番しローカル主キーに。POST は常に同じ `client_id` を送るため、オフライン作成分の再送でも重複しない。
- **`sync_state` 遷移**: `pending_create → (POST) → synced`／`synced → 編集 → pending_update → (PATCH) → synced`／`synced → 削除 → pending_delete → (DELETE) → ローカル物理削除`。未同期の作成を削除した場合はローカルから即破棄する。
- **`sync()`**: pending を作成順に push（`pending_create`=POST / `pending_update`=PATCH / `pending_delete`=DELETE）してから、`GET /diary/sync?since=last_synced_at` で差分を pull。取得した `server_time` を次回の `since` として保持する（`diary_sync_meta` テーブル）。
- **競合解決**: `updated_at` の last-write-wins で単純化。pull 時、未 push のローカル編集が新しければローカルを優先する。
- **自動同期**: `NetworkMonitor` のオフライン→オンライン復帰、各ミューテーション直後、一覧の Pull to refresh／画面復帰で発火。オフラインはネットワーク呼び出しの失敗から検知し、`SyncBanner.Offline` を控えめに表示する。

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
