# Runa

Runa は月をモチーフにしたモバイルアプリ（Android / iOS）である。日々の記憶を静かに書き留め、一日を月の光のように振り返るための場所を提供することを目指す。UI は 3 つのテーマ（夜／あさ／ピンク×ピンク、既定は夜のダーク）を切り替えられ、余白を広くとった最小限の装飾と月の意匠で統一する。

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

`apps/kotlin` を Android Studio で開き、`androidApp` をエミュレータまたは実機で実行する（`make android-run` でも可）。接続先は `gradle.properties` の `RUNA_BASE_URL` で、既定は dev バックエンド。ローカルの backend を叩くときだけ上書きする（エミュレータは `http://10.0.2.2:8080`、実機は同じ LAN のホスト IP）。

### iOS

共有 XCFramework をビルドし、`apps/swift` で `xcodegen generate` を実行してから、`Runa` を iOS 16 以上のシミュレータまたは実機で実行する（`make ios-run` でも可）。接続先は `project.yml` の `RUNA_BASE_URL` で、既定は dev バックエンド。ローカルの backend を叩くときだけ上書きする（シミュレータは `http://localhost:8080`、実機は同じ LAN のホスト IP）。

> クライアントに注入する Base URL はホスト + ポートのみ（`/api/v1` は含めない）。実機からは `10.0.2.2` も `localhost` も届かないため、既定は dev バックエンドにしてある。ホストの LAN IP は `make lan-ip` で確認できる。

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

- **Android**: `BuildConfig` の Base URL を `initKoin(context, baseUrl)` に渡す（`gradle.properties` の `RUNA_BASE_URL`、既定は dev バックエンド）。認証のセキュアストレージ（EncryptedSharedPreferences）が `Context` を必要とするため、Android は Context を伴う 2 引数版を使う。
- **iOS**: `Info.plist` の Base URL を `initKoin(baseUrl)` に渡す（`project.yml` の `RUNA_BASE_URL` ビルド設定を差し込む、既定は dev バックエンド）。

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

### ローカルファースト（ダイアリー / ふりかえりカレンダー / 今日の月）

- **描画の正はローカル DB＋月相計算**。ふりかえりカレンダー（12）と今日の月（15）は、端末の SQLDelight 日記と `MoonPhaseCalculator` だけで組み立てるため**機内モードでも完全動作**する。`CalendarRepository.observeMonth` が唯一のレンダリング源、`TodayMoonRepository.getTodayMoon` は純粋なオフライン計算。
- **タイムゾーンはユーザーの現地日付でグルーピング**（`kotlinx-datetime`）。日付境界は**現地 0:00–24:00**で、現地 23:30 に書いた記録は UTC で翌日でもその現地日に留まる。各日の月相は現地正午で計算する。空の日に「書く」と、その日付でバックデート記録される。
- **`GET /diary/calendar?year=&month=&tz=` はサーバ側の件数の正（整合性確認の補助）**で、描画には使わない。他端末で書いた分は既存の `/diary/sync` でローカルへ流し込み、`tz`（IANA）でサーバ側も同一の現地日グルーピングを行う（既定 UTC）。詳細は [apps/go/README.md](apps/go/README.md) の「Calendar design」。

### インサイト（16 ふりかえり）

「うつろい」= ダイアリーの mood と記録から週/月の傾向を映し返す静かな画面。**描画の正はローカル集計**で通信不要。

- **集計ロジックは `shared` の純ロジックが主役**。`InsightCalculator.calculate(period, entries, zone)` が `daysJournaled`／`moodDistribution`／`mostFrequentMood`／`longestStreak`／`moonOverlap`（月相バケット、`MoonPhaseCalculator` を再利用）を算出。純 `commonMain`（`kotlin.math` + `kotlinx-datetime`）なので Android/iOS で結果が一致し、`commonTest` がピン留め値で担保する。`InsightRepository.observeInsight` は既存の `DiaryRepository.observeEntries()` を畳むだけで新規永続化なし。
- **mood は静かな 5 カテゴリ**（`shared` の `DiaryMood`：しずか/おだやか/つかれ/のぞみ/おもさ、値 `calm/gentle/tired/hopeful/heavy`）。感情を数値化しない。「書く」画面の mood 選択がこの値を書き込み、集計が同じ値を読む（単一の真実）。**mood 未選択（過去データ含む）は日数・記録数には数えるが分布からは除外**し、「しるしのない夜」として静かに示す。
- **週起点は既定 日曜**（`InsightPeriods.DEFAULT_WEEK_START = DayOfWeek.SUNDAY`、カレンダーの 日〜土 に合わせる）。`InsightViewModel(weekStart = …)` で変更可能。期間は `[start, endExclusive)` の半開区間で、**タイムゾーンはユーザー現地日付**（`TimeZone.currentSystemDefault()`、日境界は現地 0:00）。うるう 2 月・月/年跨ぎ・TZ 境界は `InsightPeriodsTest`／`InsightCalculatorTest` が網羅。
- **要約はまずルールベースで `shared` に閉じる**。`SummaryComposer`（interface）越しに `RuleBasedSummaryComposer` がテンプレ＋条件分岐で静かな詩文を組み立てる（断定・診断・助言はしない）。**将来サーバ LLM 要約へ差し替える場合は `SummaryComposer` の別実装を `InsightRepository` の裏で注入するだけ**で、ViewModel と両 UI は無変更（`compose` は `suspend` なので通信実装も収まる）。
- **`GET /api/v1/insights?period=weekly|monthly&start=&tz=` は任意・最小のサーバ側集計**（`days_journaled`／`entry_count`／`unmooded_count`／`mood_distribution`）。カレンダー同様の**整合性確認の補助**で、描画には使わない（クライアントはローカル集計が正）。月相はクライアント専用のためサーバは返さない。詳細は [apps/go/api/openapi.yaml](apps/go/api/openapi.yaml)。

### 設定（テーマ切替 / アカウント・データ）

「認証済み前提」の設定機能。アプリ全体の外観テーマ切替と、プロフィール編集・データエクスポート・アカウント削除を持つ。通知設定・プライバシーロックは次の機能（下記「通知・プライバシーロック」）で実装済み。

**3 テーマ（全クライアント共通のトークン）**

外観は 7 つの意味的トークンで表し、テーマ選択でまとめて差し替える。各画面はトークンを参照し、色をハードコードしない（月の意匠 `MoonArt` は全テーマ共通の固定モチーフとして例外的に固定色を持つ）。選択は `shared` が持ち（`ThemeRepository` ＝ multiplatform-settings で永続化、`observeTheme(): StateFlow<AppTheme>`、起動時に適用）、色の値は各クライアントにネイティブ定義する（Android: `RunaColorScheme` + `LocalRunaColors`、iOS: `RunaTheme` + `@Environment(\.runaTheme)`）。下表を唯一の正典とし、3 クライアントで一致させる。

| トークン | 夜 dark（既定） | あさ light | ピンク pink |
| --- | --- | --- | --- |
| background | `#0E0E12` | `#FAF7F5` | `#141017` |
| surface | `#16161C` | `#FFFFFF` | `#1E1622` |
| heading | `#F5F3EF` | `#2A2620` | `#F6EEF2` |
| body | `#C8C6CE` | `#4E483F` | `#D6C4CE` |
| subtle | `#9A9AA5` | `#8C8579` | `#A08E99` |
| accent | `#F4A9C0` | `#E79CB6` | `#F4A9C0` |
| subAccent | `#E8E2D0` | `#C9B8A0` | `#E8B7C8` |

> 夜（ダーク）と pink の accent は確定値。light の背景（`#FAF7F5`）を除く light/pink の各色は確定デザイン画像がなく、テーマ選択画面のスウォッチと文章仕様から導出した暫定値（要サインオフ）。

**API（`/api/v1`、要 Bearer・本人のみ。詳細は [apps/go/api/openapi.yaml](apps/go/api/openapi.yaml)）**

| Method / Path | 内容 |
| --- | --- |
| `PATCH /me` | 表示名（`display_name`）を更新 |
| `GET /me/export` | 本人データを JSON で一括エクスポート |
| `DELETE /me` | アカウントを完全削除（`204`） |

- **エクスポート形式**: サーバは JSON 1 本を返す（`exported_at` / `schema_version` / `user` / `diaries[]`（tombstone 除く）/ `images[]`）。画像はメタデータ＋短命の署名付き GET URL（ストレージ未設定時は `url` を省略）。クライアントの「テキスト」形式はこの JSON から日記本文を整形して生成する（整形はクライアント責務、サーバ契約は JSON 単一）。
- **削除方針（ハードデリート）**: `DELETE FROM users` を 1 トランザクションで実行し、既存の `ON DELETE CASCADE` で `refresh_tokens` / `diary_entries` / `gallery_images` / `song_history` を連鎖削除する（ソフトデリートの `deleted_at` は用いない）。リフレッシュトークンは連鎖削除で失効。アクセストークンはステートレス JWT だが、ユーザ行の消失により以後の `GET /me` 等が `401` となり構造的に無効化される（残存窓は `ACCESS_TOKEN_TTL`、既定 15 分）。オブジェクトストレージ上の画像は DB cascade の対象外のため、削除前にキーを列挙し、行削除後に非同期・ベストエフォートで削除する。クライアントは削除成功時に認証状態を未認証へ落とし、ローカル DB とトークンを消去してサインイン画面へ戻る。

### 各 OS のネイティブ設定（実クレデンシャル）

ネイティブは IDトークンを取得して shared の `loginApple/loginGoogle` に渡すだけ。検証は BE が行う。**メール＋パスワードは設定なしで E2E 動作**する。

- **backend**: `APPLE_CLIENT_IDS` / `GOOGLE_CLIENT_IDS` に許容 audience を設定（[apps/go/README.md](apps/go/README.md)）。
- **Android**: Google は Credential Manager（Gradle property `RUNA_GOOGLE_SERVER_CLIENT_ID` に Google の**Web**クライアント ID）。Apple は Web フロー（`RUNA_APPLE_SERVICE_ID` / `RUNA_APPLE_REDIRECT_URI`）。詳細は [apps/kotlin/README.md](apps/kotlin/README.md)。
- **iOS**: Apple はネイティブ（`Runa/Runa.entitlements` の Sign in with Apple、App ID にケイパビリティ付与）。Google は `Info.plist` の `GIDClientID`（iOS クライアント ID）。詳細は [apps/swift/README.md](apps/swift/README.md)。

### 通知・プライバシーロック（夜のリマインダー / 生体認証）

「認証済み前提」。OS 固有機能が主役の回で、`shared` はインターフェイスと設定の保持・状態のみを持ち、通知スケジュールと生体認証の実処理は各ネイティブの actual が担う（04 通知許可 / 21 通知設定 / 22 プライバシー・ロック）。

- **夜のリマインダー（ローカル通知）**: 指定時刻に「静かに綴る時間」を知らせる日次ローカル通知。ON/OFF と時刻（プリセット 21:00 / 22:00 / 23:00＋自由指定、既定 22:00）を持つ。`shared` の `NotificationSettingsRepository` が multiplatform-settings に永続化し、`LocalNotificationScheduler`（expect 相当のインターフェイス、`platformModule()` で束縛）越しに各 OS のスケジューラを叩く。文面は世界観に沿う詩的コピー（`ReminderNotificationText`：「月が出ました」＋「今日を、そっと綴りませんか。」）を `shared` で共有。**プッシュ（サーバ起点）は必須にせず**、ローカル通知で完結。将来のサーバ起点告知用に BE の `PUT /api/v1/devices`（トークン登録口）だけ用意（下記）。
- **プライバシー・ロック（生体認証）**: ON にすると起動/復帰時に生体認証（Face ID / BiometricPrompt）でアプリをロックし、成功でのみ中身が見える。失敗時は端末パスコードにフォールバック（Android: `BIOMETRIC_STRONG or DEVICE_CREDENTIAL`／iOS: `LAPolicyDeviceOwnerAuthentication`）。**認証（サインイン）とは別レイヤーのロック**で、`AppLockViewModel` が `Unlocked / Locked / Authenticating / Unavailable` を持ち、ロック中はコンテンツを構築しない（内容が背後に漏れない）。端末セキュリティ未設定（`Unavailable`）は恒久ロックアウトを避け、注意表示のうえ内容を通す。設定は単一の `lockEnabled`（確定デザインの パスコード/Face ID/すぐにロック 3 コントロールは単一 ON/OFF に簡素化、合意済み）。
- **OS 別の権限・設定要件**:
  - **Android**: `POST_NOTIFICATIONS`（API 33+、導入フロー ④ で実行時要求）、`RECEIVE_BOOT_COMPLETED`（再起動後の再スケジュール）を `shared` の androidMain マニフェストで宣言。日次通知は `AlarmManager.setAndAllowWhileIdle`（不正確・許可不要）＋ `BroadcastReceiver` で翌日を再スケジュール。生体は `androidx.biometric`、`MainActivity` は `FragmentActivity`。詳細は [apps/kotlin/README.md](apps/kotlin/README.md)。
  - **iOS**: `Info.plist` に `NSFaceIDUsageDescription`、`project.yml` に `UserNotifications.framework` / `LocalAuthentication.framework`（静的 XCFramework のリンク要件）。日次通知は `UNCalendarNotificationTrigger`（repeats、OS 管理で再起動対応）。詳細は [apps/swift/README.md](apps/swift/README.md)。
- **BE（任意・最小）**: `PUT /api/v1/devices`（`push_token` / `platform` / `notify_time` / `enabled`）は将来のサーバ起点通知（FCM/APNs）用のトークン登録口。今回のリマインダーはローカル通知で完結するため必須ではなく、`shared` からの呼び出しも将来スライスで接続する。詳細は [apps/go/README.md](apps/go/README.md) / [apps/go/api/openapi.yaml](apps/go/api/openapi.yaml)。

### 状態画面（空 / オフライン / ローディング / エラー）

「認証済み前提」。各機能が個別に持っていた状態表示を、**共通の状態語彙（`shared`）＋共通コンポーネント（各 OS）** に統一する回。新規ビジネスロジックはほぼ無く、世界観（月モチーフ・静けさ）で一貫させるのが目的（確定デザイン 24 空 / 25 オフライン / 26 ローディング / 27 エラー）。

- **`UiState<T>`（`shared` の `core/state`）**: ページ全体の状態を全機能で統一した sealed 型。
  - `Loading` … 静かな全画面ローディング。
  - `Content(data, sync: SyncPhase)` … 本文＋背景同期フェーズ。**オフライン/同期は本文を隠さず `sync` の帯で示す**（2層オフラインモデル）。
  - `Empty` … まだ何も無い＝行動への招待。
  - `Failure(error: AppError)` … 本文を出せない致命状態（`error` で オフライン画面 か エラー画面 を出し分け）。
  - **2層オフライン（DoD#2）**: キャッシュ/月相など見せられるものがある時は `Content(sync = Offline)`（本文＋静かな帯）、見せるものが何も無い時だけ `Failure(AppError.Offline)`（全画面 25）。ローカルファースト機能（ダイアリー/カレンダー/インサイト/ギャラリー/ホーム）は前者が基本で、`Failure` は実質発生しない。
  - コンテンツ系 VM（Home/Diary/Calendar/Insight/Gallery/TodayMoon）は `UiState<T>` を返す。チーム/フォーム/アクション状態しか持たない VM（テーマ・通知・プレイヤー・保存状態・アカウント操作）は現状維持。ギャラリーの表示テーマ、インサイトの期間見出しは Content/Empty 双方に出す「クローム」なので `UiState` とは別の flow（`displayTheme` / `header`）で公開する。
- **`SyncPhase`（同期フェーズ・帯）**: `Idle / Syncing / Offline / Error`。従来の重複 enum（diary `SyncStatus`・`GallerySyncStatus`・`SyncBanner`・`CalendarBanner`・`InsightBanner`・`GalleryBanner`）を 1 型に集約。repository の `syncStatus` もこの型を公開。帯（`RunaSyncBanner`）は Offline/Error のみ表示し、Syncing は各画面の更新表示に委ねる。
- **`AppError`（エラー分類）**: `Offline`（到達不可）/ `Auth`（認証切れ→再認証）/ `Server`（4xx・5xx）/ `Unknown`。分類器 `Throwable.toAppError()` は既存の一行規則（`if (e is ApiException) Error else Offline`）を精緻化し、`ApiException` の **401 → `Auth`**、その他 `ApiException` → `Server`、非 `ApiException`（ネットワーク未応答）→ `Offline`。
- **認証切れ → 再認証（DoD#3）**: 401 はまず HTTP 層で透過 refresh され、refresh も失敗した時だけ `TokenStore.sessionExpired` が発火し**アプリ全体が自動でサインインに戻る**（既存の主導線）。加えて `AppError.Auth` のエラー画面は CTA「サインインし直す」で共通の再認証アクション（= セッションクリア）を呼ぶ。ナビゲーション route ではなく、Android は `LocalReauthenticate`、iOS は `\.runaReauthenticate` の環境値で app 全体に注入（サインアウトと同じ仕組み）。
- **共通コンポーネント（Android Compose / iOS SwiftUI）**: `RunaLoadingView` / `RunaEmptyView` / `RunaOfflineView` / `RunaErrorView` と、本文上の `RunaSyncBanner`。既存の月エンブレム（`GlowingMoon` 26 / `NewMoonEmblem` 24 / `CloudedMoon` 25 / `StumbleEmblem` 27、テーマ非連動の固定モチーフ）をラップし、周囲の文言/CTA はテーマトークンを読む。ローディングは月＋3点の静かな表示（スピナー禁止）で、**reduced motion 尊重**（Android=`ANIMATOR_DURATION_SCALE==0`、iOS=`accessibilityReduceMotion`）で静止に切り替わる。文面は LUNA の声（謝らず・何が起きたか・どうすればよいか）。空の文言は機能ごと（例: ダイアリー「まだ、なにもない夜。」）に差し込む。
- **画面の使い方**: Android は `RunaStateView(state) { data, sync -> … }` ディスパッチャで全画面を包む（本文冒頭に `RunaSyncBanner(sync)`）。iOS は各 View が `onEnum(of:)` で分岐し同じ共通コンポーネントへ渡す（コレクション payload の Diary/Gallery は observable でネイティブ enum に写して SKIE ジェネリクスを避ける）。両 OS で状態表現・モチーフ・文言が一致する（DoD#5）。
- **テスト（commonTest）**: `AppErrorTest`（401→Auth / その他 4xx・5xx→Server / 非 ApiException→Offline）、`DiaryListViewModelTest`（Loading→Empty / Content が `SyncPhase` を帯として保持 / オフライン→復帰で本文を落とさず帯だけ切替）。
- **ローカル検証の限界**: `shared` は `./gradlew :shared:testDebugUnitTest` で green を確認。JDK17/Xcode 不在のため Android/iOS UI は手レビュー。iOS の SKIE 記号（`UiState`/`AppError`/`SyncPhase` のブリッジ、ジェネリック `data` の型）は実 XCFramework ビルドに対して再整合が必要（[apps/swift/README.md](apps/swift/README.md) の方針に従う）。

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
