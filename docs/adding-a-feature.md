# 機能を追加する（縦切りの型）

このリポジトリはウォーキングスケルトンである。バックエンド・共有ロジック・各 OS の UI を貫く配線と、単一の疎通確認パス（`healthz`）だけが通っている。機能はこの骨組みの上に**縦切り（vertical slice）**で載せる。

## 原則

- 1 つの機能は、全レイヤー（backend → shared → Android/iOS UI）を上から下まで貫く 1 本の縦串として実装する。
- レイヤーごとに機能を横断的に足さない。「backend にだけエンドポイントを増やす」「UI にだけ画面を足す」ではなく、疎通する 1 本を通し切る。
- OS 固有の関心事（セキュアストレージ・プッシュトークン・生体認証・課金・SQL ドライバ）は `shared` の `expect`/`actual` フレームの背後に閉じ込め、共通ロジックからは `expect` 越しに触る。

### なぜ

- 各スライスが端から端まで疎通するので、途中のレイヤーだけが宙に浮いた「未接続の実装」を残さない。
- レイヤー境界（API 契約・`ApiClient`・`expect`/`actual`）が既に決まっているため、機能追加は「境界に沿って埋める」作業になり、設計判断が局所化する。
- 骨組みが薄いうちに型を固定しておくことで、機能が増えても配線の一貫性が保たれる。

## どこに何を足すか（レイヤー順）

必ず **backend → shared → androidApp → ios** の順で足す。契約（API とその OpenAPI）を先に確定させ、クライアントは実在するエンドポイントに対してビルドする。

### 1. backend（`apps/go/`）

| 順序 | 追加先 | 役割 |
| --- | --- | --- |
| 1 | `migrations/` | スキーマ変更（テーブル追加など）のマイグレーション |
| 2 | `internal/repository/` | 永続化層。DB アクセスを閉じ込める |
| 3 | `internal/service/` | ドメインロジック。repository を束ねる |
| 4 | `internal/handler/` | HTTP ハンドラ。service を呼び、リクエスト/レスポンスを変換する |
| 5 | `internal/server/` | ルート登録。`/api/v1` 配下に新エンドポイントを結線する |
| 6 | `api/openapi.yaml` | 追加したエンドポイントを契約として記述する |

### 2. shared（`apps/kotlin/shared/`）

| 順序 | 追加先 | 役割 |
| --- | --- | --- |
| 1 | `network/dto/` | リクエスト/レスポンスの `@Serializable` DTO |
| 2 | `ApiClient` | 新エンドポイントを叩く `suspend` メソッドを追加 |
| 3 | `feature/<name>/` | 機能ごとの `ViewModel` と `UiState`（`sealed interface`） |
| 4 | `di/Koin.kt` | 追加した `ApiClient` メソッド利用箇所・`ViewModel` を Koin に登録 |

`ViewModel` と `UiState` は `HealthzViewModel` / `HealthzUiState`（`Loading` / `Ok` / `Error`）を手本にする。UI 状態は `StateFlow` で公開し、Android は直接、iOS は SKIE 経由で購読する。

### 3. androidApp（`apps/kotlin/androidApp/`）

| 追加先 | 役割 |
| --- | --- |
| `ui/screens/<Name>Screen.kt` | 共有 `ViewModel` を Koin から解決し、`state` を購読する Compose 画面 |
| ナビゲーション | `navigation-compose` のルートを 1 つ追加し、タブまたは遷移先に結線する |

### 4. ios（`apps/swift/`）

| 追加先 | 役割 |
| --- | --- |
| `<Name>View.swift` | 共有 `ViewModel`（SKIE でブリッジ）を購読する SwiftUI ビュー |
| ナビゲーション | タブまたはルートを 1 つ追加する |

## 具体例: `profile` スライスの雛形（コードなし・パスのみ）

実際のロジックは持たない、触るファイルパスだけを示すテンプレート。新機能ではこの `profile` を機能名に読み替える。

```
apps/go/
  migrations/0002_create_profiles.sql          # プロフィール用テーブル
  internal/repository/profile_repository.go
  internal/service/profile_service.go
  internal/handler/profile_handler.go
  internal/server/router.go                     # GET /api/v1/profile を登録
  api/openapi.yaml                              # /api/v1/profile を追記

apps/kotlin/shared/src/commonMain/kotlin/com/runa/shared/
  network/dto/ProfileDto.kt                     # @Serializable data class
  network/ApiClient.kt                          # suspend fun profile(): ProfileDto を追加
  feature/profile/ProfileViewModel.kt
  feature/profile/ProfileUiState.kt             # Loading / Ok / Error
  di/Koin.kt                                     # ProfileViewModel を登録

apps/kotlin/androidApp/src/main/kotlin/com/runa/android/
  ui/screens/ProfileScreen.kt                   # ProfileViewModel を購読
  <nav>                                         # ルートを追加

apps/swift/Runa/Screens/
  ProfileView.swift                             # ProfileViewModel(SKIE) を購読
  <nav>                                         # タブ/ルートを追加
```

> これはパスの型を示すための雛形であり、実コードではない。エンドポイント名・テーブル名・画面名は機能ごとに置き換える。

## PR への分け方（CONTRIBUTING に従う）

規約は [../CONTRIBUTING.md](../CONTRIBUTING.md) に従う。**1 Issue → 1 PR → 1 commit** を一対一で対応させ、タイトルは [hack/prefix.yaml](../hack/prefix.yaml) のカテゴリ接頭辞で始める。

1 本のスライスは、レイヤー境界で複数の PR に割る。

| 順序 | PR のカテゴリ | 含める変更 |
| --- | --- | --- |
| 1 | `[Server]` | backend の migration〜handler〜route と `api/openapi.yaml` |
| 2 | `[Android/iOS]` | shared の DTO・`ApiClient`・`ViewModel`/`UiState`・Koin 登録 |
| 3 | `[Android]` | androidApp の画面とナビゲーション |
| 4 | `[iOS]` | ios の画面とナビゲーション |

### なぜこの順序か

- **契約を先に確定させる**: backend と `openapi.yaml` を最初にマージすれば、クライアントは実在するエンドポイントに対してビルド・疎通できる。契約が動く前に UI を書くと、後から契約変更でクライアントが手戻る。
- **shared は `[Android/iOS]` にまとめる**: shared の変更は Android と iOS の両方に影響する。専用の Shared カテゴリは設けず、CONTRIBUTING の方針どおり複合接頭辞 `[Android/iOS]` で影響 OS を明示する。
- **UI は最後に、OS ごとに割る**: shared の `ViewModel` が入ってから各 OS の画面 PR を出す。Android と iOS は独立した PR にし、レビュー単位を OS 担当に沿わせる。
