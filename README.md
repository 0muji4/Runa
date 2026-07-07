# Runa

Runa は月をモチーフにしたモバイルアプリ（Android / iOS）である。日々の記憶を静かに書き留め、一日を月の光のように振り返るための場所を提供することを目指す。UI はダークテーマのみで、余白を広くとった最小限の装飾と月の意匠で統一する。

このリポジトリは Runa の**ウォーキングスケルトン**である。バックエンド・共有ロジック・各 OS の UI を貫く縦の配線と、単一の疎通確認パス（`healthz`）だけが通っており、各画面は空のシェル（タイトルとプレースホルダのみ）に留めてある。プロダクト機能は、この骨組みの上に**縦切り（vertical slice）**で 1 本ずつ載せていく。機能の足し方は [docs/adding-a-feature.md](docs/adding-a-feature.md) を参照する。

## アーキテクチャ

- **クライアント**: モバイルアプリ（Android / iOS）。
- **サーバ**: Go バックエンド（API・認証・データ保存など）。
- **インフラ**: マネージド DB・スケジュール実行などをマネージド部品に寄せ、自前で運用するのはアプリケーションのロジックに絞る。

<!-- TODO: 確定した要件（Why/What）とアーキテクチャ（How）はドキュメント（docs/）で定める。 -->

## リポジトリ構成

モノレポは 4 つのレイヤーで構成する。各ディレクトリの詳細はそれぞれの README を参照する。

| ディレクトリ | レイヤー | 内容 | README |
| --- | --- | --- | --- |
| `apps/go/` | [Server] | Go + PostgreSQL のレイヤードアーキテクチャ（`cmd/api`、`internal/{config,server,handler,service,repository}`、`migrations`、`api/openapi.yaml`）。app + postgres の docker-compose 付き。 | [apps/go/README.md](apps/go/README.md) |
| `apps/kotlin/shared/` | [Android/iOS] | KMP 共有モジュール（Ktor / serialization / datetime / SQLDelight / Koin / multiplatform-settings / SKIE・XCFramework）。`healthz` パスと `expect` フレームを持つ。 | [apps/kotlin/shared/README.md](apps/kotlin/shared/README.md) |
| `apps/kotlin/androidApp/` | [Android] | Jetpack Compose の UI のみ。共有モジュールを直接利用する。 | [apps/kotlin/README.md](apps/kotlin/README.md) |
| `apps/swift/` | [iOS] | Xcode + SwiftUI の UI のみ。共有 XCFramework をローカル Swift Package 経由で利用する（プロジェクトは XcodeGen で生成）。 | [apps/swift/README.md](apps/swift/README.md) |
| `docs/` | [Docs] | 設計・運用ドキュメント（本 README が参照する [縦切りの型](docs/adding-a-feature.md) など。今後 PRD / DD を追加）。 | — |

## 全体起動手順

### backend

```sh
cd apps/go
docker compose up
curl http://localhost:8080/api/v1/healthz
# => {"status":"ok"}
```

### Android

`apps/kotlin` を Android Studio で開き、`androidApp` をエミュレータで実行する。Home タブに疎通結果（成功時「接続OK」）が表示される。エミュレータからホストの backend へは `http://10.0.2.2:8080` で届く。

### iOS

共有 XCFramework をビルドし、`apps/swift` で `xcodegen generate` を実行してから、`Runa` を iOS 16 以上のシミュレータで実行する。シミュレータからホストの backend へは `http://localhost:8080` で届く。

> クライアントに注入する Base URL はホスト + ポートのみ（`/api/v1` は含めない）。エミュレータは `10.0.2.2`、シミュレータは `localhost` を使う点に注意する。

## 環境変数

### backend

`:8080`（環境変数 `PORT`、既定 `8080`）で待ち受け、`/api/v1` を base path とする。

| 変数 | 用途 | 既定 |
| --- | --- | --- |
| `PORT` | 待ち受けポート | `8080` |
| `DATABASE_URL` | PostgreSQL 接続文字列 | — |
| `LOG_LEVEL` | ログレベル | — |
| `CORS_ALLOWED_ORIGINS` | 許可する CORS オリジン | — |
| `APP_ENV` | 実行環境（dev / prod など） | — |
| `JWT_SECRET` | アクセストークン(HS256)の署名鍵。**本番は必ず上書き** | `dev-insecure-...` |
| `ACCESS_TOKEN_TTL` | アクセストークン有効期限（Go duration） | `15m` |
| `REFRESH_TOKEN_TTL` | リフレッシュトークン有効期限 | `720h`(30日) |
| `APPLE_CLIENT_IDS` | Apple IDトークンの許容 audience（Bundle ID / Service ID をカンマ区切り） | 空 |
| `GOOGLE_CLIENT_IDS` | Google IDトークンの許容 audience（OAuth Client ID をカンマ区切り） | 空 |

### クライアントへの Base URL 注入

- **Android**: `BuildConfig` の Base URL を `initKoin(context, baseUrl)` に渡す（dev は `http://10.0.2.2:8080`）。認証のセキュアストレージ（EncryptedSharedPreferences）が `Context` を必要とするため、Android は Context を伴う 2 引数版を使う。
- **iOS**: `Info.plist` の Base URL を `initKoin(baseUrl)` に渡す（dev は `http://localhost:8080`）。

## 認証（最初の縦切り機能）

最初のプロダクト機能として「認証」を BE → shared → 各 OS UI に E2E で通してある。以降の機能は「認証済み前提」で `AuthRepository.authState` を購読して乗る。

### API（`/api/v1`、詳細は [apps/go/api/openapi.yaml](apps/go/api/openapi.yaml)）

| Method / Path | 内容 |
| --- | --- |
| `POST /auth/signup` | メール＋パスワードで登録 |
| `POST /auth/login` | メール＋パスワードでログイン |
| `POST /auth/apple` | Apple IDトークン検証→ログイン/作成 |
| `POST /auth/google` | Google IDトークン検証→ログイン/作成 |
| `POST /auth/refresh` | リフレッシュトークンをローテーションし新トークン発行 |
| `POST /auth/logout` | リフレッシュトークンを失効（冪等） |
| `GET /me` | **要 Bearer**。動作確認用の保護エンドポイント |

- アクセストークンは短命 JWT(HS256)、リフレッシュは長命の不透明トークン（DB には SHA-256 ハッシュのみ保存、`/auth/refresh` でローテーション）。
- パスワードは **Argon2id**（OWASP 推奨パラメータ）でハッシュ。Apple/Google の IDトークンは各 JWKS で**署名検証**（`iss`/`aud`/`exp`）。
- エラー形式は全機能共通の `{"error":{"code","message","details?}}`。`login`/`signup` は IP 単位のレート制限あり。

### クライアント（shared）

- `AuthRepository`: `signupEmail / loginEmail / loginApple(idToken) / loginGoogle(idToken) / refresh / logout / getMe / restoreSession`。
- 状態 `AuthState`: `Restoring`（起動時復元中）/ `Unauthenticated` / `Authenticating` / `Authenticated(user)` / `Error(message)`。`AuthViewModel.state` として公開し、Android は直接、iOS は SKIE 経由で購読する。
- トークンはセキュアストレージに永続化（Android: EncryptedSharedPreferences、iOS: Keychain）。保護リクエストが 401 を返すと HTTP 層が**自動でリフレッシュ→元リクエストを再送**し、失敗時は未認証へ遷移する。
- 起動フロー: 保存トークンから復元し `GET /me` で確認。未認証は導入→サインイン、認証済はタブ本体を出し分け、サインアウトで戻る。認証済みホームは `/me` の `display_name` を表示する。

### 各 OS のネイティブ設定（実クレデンシャル）

ネイティブは IDトークンを取得して shared の `loginApple/loginGoogle` に渡すだけ。検証は BE が行う。**メール＋パスワードは設定なしで E2E 動作**する。

- **backend**: `APPLE_CLIENT_IDS` / `GOOGLE_CLIENT_IDS` に許容 audience を設定（[apps/go/README.md](apps/go/README.md)）。
- **Android**: Google は Credential Manager（Gradle property `RUNA_GOOGLE_SERVER_CLIENT_ID` に Google の**Web**クライアント ID）。Apple は Web フロー（`RUNA_APPLE_SERVICE_ID` / `RUNA_APPLE_REDIRECT_URI`）。詳細は [apps/kotlin/README.md](apps/kotlin/README.md)。
- **iOS**: Apple はネイティブ（`Runa/Runa.entitlements` の Sign in with Apple、App ID にケイパビリティ付与）。Google は `Info.plist` の `GIDClientID`（iOS クライアント ID）。詳細は [apps/swift/README.md](apps/swift/README.md)。

### 動作確認

```sh
# backend（DB 込みで起動）
cd apps/go && docker compose up --build
# 単体・結合テスト（DB 不要で green）
cd apps/go && go test ./...
# 手動疎通
curl -XPOST localhost:8080/api/v1/auth/signup \
  -H 'Content-Type: application/json' \
  -d '{"email":"a@b.com","password":"password123","display_name":"Runa"}'
# 返却 access で保護エンドポイントを叩く
curl localhost:8080/api/v1/me -H "Authorization: Bearer <access_token>"
```

shared のユニットテスト（401 自動リフレッシュ・再送、トークン復元、各サインイン経路）は `cd apps/kotlin && ./gradlew :shared:testDebugUnitTest`。

## バージョン整合

ビルドツールチェーンは以下の版で固定して開始する。ローカル環境によっては手元での整合合わせが必要になる点に留意する。

- Kotlin 2.0.21 / Android Gradle Plugin 8.5.2 / Gradle wrapper 8.9 / JDK 17
- Android compileSdk 34 / minSdk 26 / targetSdk 34
- Ktor 3.0.1 / kotlinx-coroutines 1.9.0 / kotlinx-serialization-json 1.7.3 / kotlinx-datetime 0.6.1
- SQLDelight 2.0.2 / Koin 4.0.0 / multiplatform-settings 1.2.0 / SKIE 0.10.1
- Jetpack Compose BOM 2024.10.01 / activity-compose 1.9.3 / navigation-compose 2.8.3 / compose plugin 2.0.21
- iOS deployment target 16.0

## 開発に参加する

ブランチ・PR・コミットの規約は [CONTRIBUTING.md](CONTRIBUTING.md) に従う。要点は **Issue → PR → コミットを 1:1 で対応させ、`main` を直線履歴に保つ**こと。CI が全 PR で検証する。
