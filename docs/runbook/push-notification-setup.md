# 夜のリマインド（サーバー push）のセットアップ手順

設計は [DD](../dd/nightly-reminder-server-push.md)。ここは「値をどこで作り、どこに貼るか」だけを書く。所要時間はおよそ 40 分。Apple Developer Program と Google アカウントが必要。

最終的に Render の `runa-backend-dev` に入る環境変数は次の 12 個で、この手順書の各節がそれぞれの値を作る。

| 環境変数 | 作る節 | 値 |
|---|---|---|
| `APNS_TEAM_ID` | 1 | Apple Developer の Team ID（10 文字） |
| `APNS_KEY_ID` | 1 | APNs Auth Key の Key ID（10 文字） |
| `APNS_PRIVATE_KEY` | 1 | `AuthKey_<KEY_ID>.p8` を base64 にした 1 行 |
| `APNS_BUNDLE_ID` | 1 | `com.runa`（render.yaml の既定のまま） |
| `APNS_ENVIRONMENT` | 1 | `sandbox`（Xcode から入れた実機）/ `production`（TestFlight・App Store） |
| `GCP_PROJECT_ID` | 2 | Firebase / GCP のプロジェクト ID |
| `GCP_SERVICE_ACCOUNT_JSON` | 4 | サービスアカウント鍵 JSON を base64 にした 1 行 |
| `CLOUD_TASKS_LOCATION` | 3 | `asia-northeast1`（既定のまま） |
| `CLOUD_TASKS_QUEUE` | 3 | `runa-reminders`（既定のまま） |
| `PUSH_CALLBACK_BASE_URL` | 5 | `https://runa-backend-dev-xxxx.onrender.com`（末尾スラッシュ無し） |
| `PUSH_CALLBACK_TOKEN` | 5 | Render が生成（`generateValue: true`） |
| `PUSH_ENABLED` | 5 | `true`（既定のまま。止めたいときだけ `false`） |

Android アプリのビルドには別途 4 つの Gradle property（`RUNA_FIREBASE_PROJECT_ID` / `RUNA_FIREBASE_APP_ID` / `RUNA_FIREBASE_API_KEY` / `RUNA_FIREBASE_SENDER_ID`）が要る。2 節で作る。

## 1. Apple: App ID にケイパビリティを付け、APNs Auth Key を作る

1. https://developer.apple.com/account/resources/identifiers/list を開き、`com.runa` の App ID を選ぶ。
2. Capabilities の一覧で **Push Notifications** にチェックを入れ、右上 **Save**。（既に Sign in with Apple が付いている画面と同じ場所。）
3. https://developer.apple.com/account/resources/authkeys/list を開き、**+**（Create a key）。
4. Key Name に `Runa APNs`、**Apple Push Notifications service (APNs)** にチェック → **Continue** → **Register**。
5. **Download** で `AuthKey_XXXXXXXXXX.p8` を保存する。**この画面を離れると二度と取れない。** 画面の **Key ID**（`XXXXXXXXXX`）を控える → `APNS_KEY_ID`。
6. https://developer.apple.com/account の右上 **Membership details** → **Team ID**（10 文字）を控える → `APNS_TEAM_ID`。
7. 鍵を 1 行の base64 にする:

```bash
base64 -i ~/Downloads/AuthKey_XXXXXXXXXX.p8 | tr -d '\n'; echo
```

出力（`LS0tLS1CRUdJTi…` で始まる 1 行）→ `APNS_PRIVATE_KEY`。

8. `APNS_ENVIRONMENT` は、`make ios-install IOS_TEAM=… IOS_DEVICE=…` で入れた実機を使う間は `sandbox`。TestFlight / App Store で配ったら `production`。取り違えると全 iOS 端末が `invalid_token` になる（DD の障害シナリオ。直して端末を起動し直せば戻る）。

iOS 側のコードは `Runa/Runa.entitlements` に `aps-environment` を持つ（このリポジトリに入っている）。実機ビルドは `make ios-device-build IOS_TEAM=XXXXXXXXXX` で、App ID にケイパビリティが無いと署名で失敗する。

## 2. Firebase: プロジェクトと Android アプリを登録する

1. https://console.firebase.google.com/ → **プロジェクトを追加** → 名前 `runa-dev` → Google アナリティクスは **無効** → 作成。
2. 作成後、⚙ **プロジェクトの設定** → **全般** タブ → **プロジェクト ID**（例 `runa-dev-1a2b3`）を控える → `GCP_PROJECT_ID`。同じ画面の **プロジェクト番号** を控える → `RUNA_FIREBASE_SENDER_ID`。
3. 同じ **全般** タブの下部 **マイアプリ** → **アプリを追加** → Android → パッケージ名 `com.runa` → **アプリを登録**。`google-services.json` は**ダウンロードしない**（使わない）。**次へ** を押し切って終える。
4. 登録された Android アプリのカードで **アプリ ID**（`1:123456789012:android:abcdef…`）を控える → `RUNA_FIREBASE_APP_ID`。
5. **プロジェクトの設定** → **全般** → 上部の **ウェブ API キー**（無ければ Google Cloud Console → **APIs & Services** → **Credentials** → **Android key (auto created by Firebase)**）を控える → `RUNA_FIREBASE_API_KEY`。
6. **プロジェクトの設定** → **Cloud Messaging** タブ → **Firebase Cloud Messaging API (V1)** が **有効** であることを確認する（無効なら ⋮ → Google Cloud Console で有効化）。

Android のビルドに渡す:

```bash
cd apps/kotlin && ./gradlew :androidApp:assembleDebug \
  -PRUNA_FIREBASE_PROJECT_ID=runa-dev-1a2b3 \
  -PRUNA_FIREBASE_APP_ID='1:123456789012:android:abcdef0123456789' \
  -PRUNA_FIREBASE_API_KEY=AIza… \
  -PRUNA_FIREBASE_SENDER_ID=123456789012
```

4 つのうち 1 つでも空だと Firebase は初期化されず、その端末は登録されない（アプリは動く）。

## 3. Cloud Tasks: API を有効にしてキューを作る

`gcloud` が無ければ https://cloud.google.com/sdk/docs/install を先に。以下 `PROJECT` は 2 節のプロジェクト ID。

```bash
gcloud auth login
gcloud config set project PROJECT
gcloud services enable cloudtasks.googleapis.com
gcloud tasks queues create runa-reminders \
  --location=asia-northeast1 \
  --max-attempts=5 \
  --min-backoff=30s \
  --max-backoff=10m \
  --max-retry-duration=60m \
  --max-dispatches-per-second=10
```

`--max-retry-duration=60m` が「選んだ時刻から 1 時間以内に届くか、届かない」を決める値（DD Q2）。変えるなら `PushService` の `RetryWindow`（`DefaultPushConfig`）と揃える。

確認:

```bash
gcloud tasks queues describe runa-reminders --location=asia-northeast1
```

`state: RUNNING` と `retryConfig` が上の値であればよい。

## 4. サービスアカウント: FCM と Cloud Tasks の両方に使う鍵を作る

```bash
gcloud iam service-accounts create runa-push --display-name="Runa push"
gcloud projects add-iam-policy-binding PROJECT \
  --member="serviceAccount:runa-push@PROJECT.iam.gserviceaccount.com" \
  --role="roles/firebasecloudmessaging.admin"
gcloud projects add-iam-policy-binding PROJECT \
  --member="serviceAccount:runa-push@PROJECT.iam.gserviceaccount.com" \
  --role="roles/cloudtasks.enqueuer"
gcloud projects add-iam-policy-binding PROJECT \
  --member="serviceAccount:runa-push@PROJECT.iam.gserviceaccount.com" \
  --role="roles/cloudtasks.taskDeleter"
gcloud iam service-accounts keys create ~/runa-push-key.json \
  --iam-account=runa-push@PROJECT.iam.gserviceaccount.com
base64 -i ~/runa-push-key.json | tr -d '\n'; echo
```

最後の出力（`eyJ…` で始まる 1 行）→ `GCP_SERVICE_ACCOUNT_JSON`。貼ったら `~/runa-push-key.json` は消す。

`enqueuer` がタスクの作成、`taskDeleter` が設定変更・サインアウト時の取り消しに使う権限で、他は要らない（Cloud Tasks からサーバーへのコールバックは共有秘密（5 節）で守る）。

## 5. Render: 環境変数を貼る

https://dashboard.render.com/ → `runa-backend-dev` → **Environment**。`render.yaml` を Blueprint として同期しているなら `sync: false` の項目が空欄で並んでいるので、そこに貼る。

| キー | 貼る値 |
|---|---|
| `APNS_TEAM_ID` | 1-6 |
| `APNS_KEY_ID` | 1-5 |
| `APNS_PRIVATE_KEY` | 1-7 の 1 行 |
| `APNS_ENVIRONMENT` | `sandbox` または `production`（1-8） |
| `GCP_PROJECT_ID` | 2-2 |
| `GCP_SERVICE_ACCOUNT_JSON` | 4 の 1 行 |
| `PUSH_CALLBACK_BASE_URL` | このサービスの URL。ダッシュボード上部に出ている `https://runa-backend-dev-xxxx.onrender.com` を末尾スラッシュ無しで |
| `PUSH_CALLBACK_TOKEN` | Blueprint が生成済み。手で作るなら `openssl rand -hex 32` |

**Save Changes** で再デプロイされる。**Logs** に次の 3 行が**出ない**ことを確認する（出たらその部分が無効）:

- `push: APNs not configured; iOS reminders disabled`
- `push: FCM not configured; Android reminders disabled`
- `push: scheduler not configured; reminders will not be scheduled`

## 6. 動作確認

1. 実機（sandbox なら Xcode / `make ios-install` から入れた iPhone、Android は 2 節の property を付けた debug ビルド）でサインインし、19 設定 → 通知 → リマインダーを ON、時刻を**今から 3〜4 分後**に設定する。
2. Cloud Tasks コンソール https://console.cloud.google.com/cloudtasks/queue/asia-northeast1/runa-reminders/tasks?project=PROJECT に、名前が `r-<uuid>-<今日の日付>-<8桁>` のタスクが 1 件、Schedule time が設定時刻で現れる。現れなければ Render の Logs で `PUT /api/v1/devices` の status を見る（400 なら `time_zone` / `install_id`、500 なら Cloud Tasks の権限）。
3. 今日まだ書いていない状態で時刻を待つ → 通知「月が出ました」が届く。Render の Logs に `reminder delivered … outcome=sent`。
4. 通知をタップ → （ロックが ON なら解除の後）10 ダイアリー作成が開く。
5. 何か書いて保存し、時刻をまた 3 分後に設定 → 時刻が来ても届かない。Logs に `outcome=skipped_written`。コンソールのタスクは消費され、翌日分が 1 件残る。
6. サインアウト → コンソールのタスクが消える（残っていても、届いた時に `stale` で何もしない）。

届かないときの切り分け:

| Logs の outcome / 状態 | 原因 | 対処 |
|---|---|---|
| `invalid_token`（iOS 全端末） | `APNS_ENVIRONMENT` の取り違え、または鍵の失効 | 値を直して再デプロイ → 端末でアプリを起動し直す（再登録で `token_invalid_at` が消える） |
| `invalid_token`（1 端末） | アンインストール等でトークンが死んだ | 何もしない。再インストールで再登録される |
| `disabled` | その OS の鍵が未設定、または `PUSH_ENABLED=false` | 5 節の Logs 3 行を確認 |
| 503 が続き、コンソールで再試行中 | APNs / FCM / Neon が応答しない | 60 分で諦める。Logs の `reminder delivery deferred` の error を見る |
| タスクが作られない | `PUSH_CALLBACK_BASE_URL` / `PUSH_CALLBACK_TOKEN` / `GCP_SERVICE_ACCOUNT_JSON` のどれかが空 | 5 節 |
| タスクは消費されたが Logs に何も無い | `PUSH_CALLBACK_BASE_URL` が別のサービスを指している | URL を直す |
| コールバックが 403 | タスクに焼かれた `PUSH_CALLBACK_TOKEN` と現在の値が違う（値を変えた） | 待つ。端末の日次再登録が翌日分を新しいトークンで予約する（最長 1 日止まる。7 節） |

## 7. 止める・鍵を替える

- 全体を止める: Render で `PUSH_ENABLED=false` → 再デプロイ。予約は作られず、既存のタスクが届いても `disabled`。
- APNs 鍵を替える: 1-3〜1-7 をやり直し、`APNS_KEY_ID` / `APNS_PRIVATE_KEY` を貼り替える。古い鍵は Apple の画面で **Revoke**。端末側の操作は要らない。
- サービスアカウント鍵を替える: 4 の `keys create` をやり直して貼り替え、古い鍵は `gcloud iam service-accounts keys delete`。
- `PUSH_CALLBACK_TOKEN` を替える: 貼り替え後、既存のタスクは 403 になる。端末は次の起動（日次の再登録）で新しいトークンのタスクを作り直すので、最長 1 日リマインドが止まる。
