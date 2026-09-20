# DD: 今日の記録が無い夜だけ、サーバーからリマインドを届ける

Status: Finalized
PRD: [今日の記録が無い夜だけ、サーバーからリマインドを届ける](../prd/nightly-reminder-server-push.md)

## 背景

PRD は、夜のリマインドを「その日の記録がまだ無いときだけ届く」サーバー起点の通知にすることを求めている。本 DD が答える問いは、「端末ごとに選ばれた時刻に、その利用者がその日を書いたかをサーバーが判定して、書いていない端末にだけ、一日一回、一時間以内に届けるには、サーバーとクライアントのどこを変えるか」である。

前提とする現状は次のとおりである。

- リマインドは端末のローカル通知である。`shared` の `NotificationSettingsRepository` が ON/OFF と時刻を multiplatform-settings に保存し、`LocalNotificationScheduler` 越しに Android は `AlarmManager`、iOS は `UNCalendarNotificationTrigger` を毎晩同じ時刻に鳴らす。文言は `shared` の `ReminderNotificationText` にある。
- サーバーには `PUT /api/v1/devices` があり、`devices` 表（`push_token` / `platform` / `notify_time` / `enabled`、一意キーは `(user_id, push_token)`）に登録できる。読み出す API も送信する処理も無く、クライアントもこれを呼んでいない。`shared` の `PushTokenProvider` は `expect class` の TODO スタブである。
- 日記は `diary_entries` に `created_at`（記録の日付を兼ねる）を持ち、`DiaryStore.CountByLocalDate` が任意のタイムゾーンで暦日ごとの件数を数えられる。
- サーバーは Render の無料プラン（アイドル時にスリープ）と Neon の無料プラン（アイドル時に自動停止）で動き、定期実行の基盤は無い。

## 概要

結論は、**「この端末に次のリマインドを送るか」を判定する時刻をサーバーが端末ごとに Google Cloud Tasks に予約し、その時刻にサーバー自身へ HTTP で呼び戻させて、記録の有無を見てから APNs / FCM に送る**である。端末は宛先と設定を登録し、届いた通知を表示・遷移するだけにする。

この方向になる理由は二つある。第一に、PRD の Step 2（書いたかで送り分ける）は、全端末の記録を知るサーバーにしか判定できない。第二に、N5（無料枠）の下では常駐プロセスも定期ポーリングも置けないため、「時刻になったら外部から起こしてもらう」形が唯一、時刻の精度（N2）と費用を両立する。

本 DD が設計判断を照らす要件は、PRD から次のとおり立てる。

- R1: 端末は宛先・設定・タイムゾーンをサーバーに自動で登録し、サインアウトと退会で外れる（Step 1）
- R2: 選んだ時刻に、その日の記録が無いときだけ届く（Step 2）
- R3: タップで新しい記録の画面が開き、ロックと認証の後に遷移する（Step 3）
- N1: 同じ端末に同じ日のリマインドは一回
- N2: 選んだ時刻から 1 時間以内に届くか、その夜は届かない
- N3: 本文と配送サービスに渡す内容に日記の内容を含めない
- N4: OFF・サインアウト・退会・無効トークンの端末に届けない
- N5: 無料枠内で運用する

論点は Q1 から Q7 で、順序は「判定をどこに置くか（Q1）」から「誰が起動するか（Q2）」「一日一回をどう守るか（Q3）」と配信の骨格を決め、次に端末の同定（Q4）と外部サービスの呼び方（Q5）、最後に端末側の振る舞い（Q6）と現在のローカル通知の扱い（Q7）を置く。

## 全体設計

変わるのは、サーバーの `devices` 表と登録経路、サーバーに増える配信経路、クライアントの登録と通知の受け口である。日記の同期・記録の API・21 通知設定の画面は変わらない。

```
[端末] DeviceRegistrar: 認証済み × push トークン × 設定 × タイムゾーン × オンライン
   │   の指紋が変わったとき（と日次で）
   │ PUT /api/v1/devices {install_id, push_token, platform, notify_time, time_zone, enabled}
   ▼
[API] DeviceService.Register ── devices を upsert ── 次の notify_time の絶対時刻を計算
   │                             └─ Scheduler.Schedule(name=r-{device}-{date}-{hash(time,tz)}, runAt, body)
   ▼
[Cloud Tasks queue]  ── runAt に ──▶  POST /api/v1/hooks/push/reminder (X-Push-Callback-Token)
                                       body = {device_id, local_date, notify_time, time_zone}
   ▼
[API] PushService.Deliver
   1. 端末行を読む。無い / OFF / 設定が body と違う / トークン無効 / 45 日超 → 200 (stale)
   2. claim(device, local_date)   ← 一日一回の冪等ロック
   3. CountByLocalDate(user, local_date) > 0 → 200 (skipped_written)
   4. Sender.Send → 200 (sent) / トークン無効を記録 / 一時失敗は 503 → Cloud Tasks が再試行
   5. 翌日の task を予約（自己継続）
   ▼
internal/push.Sender ── apns: HTTP/2 + ES256 JWT, aps.alert + kind/date/title/body
                     └─ fcm : HTTP v1 + OAuth2, データ通知 / HIGH / ttl / collapse_key
   ▼
[端末] Android: RunaMessagingService が通知を組み立てる（未ログイン・当日記録ありなら表示しない）
       iOS: OS が表示
       タップ → PendingRoute（sticky）→ ロック解除 + 認証の後に 10 ダイアリー作成へ
```

現在との差分は次の四点である。

1. `devices` 表の主キーが `(user_id, push_token)` から `(user_id, install_id)` になり、タイムゾーン・トークン無効化・予約と当日の配信状態の列が加わる。
2. サーバーに、配送サービスを呼ぶ `internal/push`（`apns` / `fcm`）と、予約コールバックを扱う `internal/schedule`（`cloudtasks`）が増え、`PushService` がこれらと `devices` / `diary_entries` を束ねる。
3. `POST /api/v1/hooks/push/reminder` が増え、`POST /auth/logout` が `install_id` を受け取る。
4. クライアントから `LocalNotificationScheduler` とその actual が消え、`DeviceRegistrar`・`PushTokenStore`・`PendingRoute` と、Android の `RunaMessagingService`・iOS の `AppDelegate` が増える。

## 詳細設計

### Q1: 「書いたか」の判定をどこに置くか（R2）

**アプローチ**: 判定に必要な「その利用者のその日の記録」を、通知の時点で誰が知っているかで決める。

**設計**: 判定はサーバーの `PushService.Deliver` が、既存の `DiaryStore.CountByLocalDate(userID, dayStart, dayStart+24h, loc)` で行う。`loc` は端末が登録したタイムゾーン、`dayStart` はコールバックの body が指す暦日の 0 時である。件数が 1 以上なら送らない。

Android には二重の判定を置く。FCM からのデータ通知を受けた `RunaMessagingService` が、端末の SQLDelight に body の `date` と同じ暦日の記録があれば表示しない。オフラインで書いた記録は同期されるまでサーバーに無いので、この分を端末で補う。iOS は表示前にアプリのコードを挟めない（通知の抑止には Apple の個別承認が要る entitlement が必要）ため、サーバーの判定だけに従う。

**根拠**: 利用者は複数の端末を持ちうるので、他端末の記録を知るのはサーバーだけである（R2）。Android の端末側判定は、サーバーの判定を置き換えるものではなく、同期の遅れを埋める補助である。

**棄却案**:

- 端末のローカル通知のまま、発火時に端末の記録を見る。Android は `BroadcastReceiver` で可能だが、iOS の `UNCalendarNotificationTrigger` は発火時にアプリのコードが走らず、他端末の記録も知れない。R2 を iOS で満たせない。
- Android だけローカル判定、iOS だけサーバー判定。通知の経路が OS ごとに二つになり、文言・時刻・設定の同期を二経路で保つことになる。

### Q2: 誰が、いつ、配信処理を起動するか（N2, N5）

**アプローチ**: サーバーは Render の無料プランでアイドル時にスリープし、Neon も自動停止する。この制約の下で「端末ごとに選ばれた時刻」に処理を起こす手段を選ぶ。

**設計**: 起動役は Google Cloud Tasks の HTTP タスクである。`PUT /api/v1/devices` が処理されるたび、および配信が終わるたびに、`PushService.ScheduleNext` が「次に `notify_time` が来る絶対時刻」（`nextFireAt`: 端末のタイムゾーンで今日のその時刻がまだ先ならそれ、過ぎていれば明日）を計算し、その時刻に `POST /api/v1/hooks/push/reminder` を叩くタスクを一件予約する。タスクの body は `{device_id, local_date, notify_time, time_zone}` で、ヘッダに共有秘密 `X-Push-Callback-Token` を持つ。

予約は自己継続する。コールバックを処理するたび（端末行が消えた・設定が変わった `stale` と、無効トークンのときを除く）に翌日分を予約するので、端末が何もしなくても毎晩続く。翌日分は今夜のタスクを取り消す前に作り、今夜のタスク自身は取り消さない。予約に失敗したら 503 を返し、再試行が claim の結果（`ClaimDone`）を見て予約だけをやり直す。連鎖が切れた場合（認証情報の失効やサーバーの長時間停止）は、端末の `DeviceRegistrar` が日次で行う再登録が繋ぎ直す。

一時失敗（配送サービスや DB が応答しない）はコールバックが 503 を返し、Cloud Tasks の再試行に任せる。キューの再試行は「最大 5 回・30 秒から 10 分の指数バックオフ・**最長 60 分**」に設定し、これが N2 の「1 時間以内に届くか、届かない」の実体になる。サーバー側にも同じ 60 分（`RetryWindow`）の判定を置き、選んだ時刻から 60 分を過ぎて届いたコールバックは `expired` として何もしない。キューの設定が変わっても深夜に届かないようにするためである。

Cloud Tasks は FCM と同じ GCP プロジェクトに置き、同じサービスアカウント鍵で REST API を直接呼ぶ（`internal/schedule/cloudtasks`）。無料枠は月 100 万操作で、端末あたり一日 1〜2 操作である。

**根拠**: 予約した時刻に外部から呼ばれる形なので、サーバーは配信時刻にだけ起き、それ以外は眠っていてよい（N5）。予約は秒単位の絶対時刻で、Cloud Tasks はそれより前に呼ばない（N2）。コールドスタート（1 分前後）は再試行が吸収する。

**棄却案**:

- サーバー内のタイマー。Render の無料プランはアイドル時にスリープするので発火しない。
- GitHub Actions の cron で 10 分ごとに「期限が来た端末」をまとめて配信する。無料だが、GitHub の cron は数分から数十分遅れることが常態で、0 時をまたぐ判定・猶予窓・`has_more` のループが要る。さらに 10 分ごとに叩くと Render の web が常時起動相当（月 744 時間、無料枠 750 時間）になり、Neon も起きたままになる。
- Postgres 上のジョブキュー（River など）。常駐 worker が必要で、その worker のポーリングが Neon の無料枠（月 100 CU 時間）を使い切る。有料化が前提になる。

### Q3: 一日一回と再試行をどう両立するか（N1, N2）

**アプローチ**: 「送り直し（再試行）はしたいが、二回は届けたくない」を、予約・コールバック・DB の三段で守る。

**設計**: 第一段はタスク名である。名前は `r-{device_id}-{local_date}-{sha256(notify_time|time_zone|updated_at)の先頭 8 桁}` で、Cloud Tasks は同名のタスク作成を 409 で拒む（`Schedule` はこれを成功として扱う）。自己継続が同じ日を二重に予約しても一件になる。登録時刻 `updated_at` をハッシュに含めるのは、Cloud Tasks が実行済み・削除済みの名前を数時間は再利用させないためで、時刻を変えた・OFF にしてすぐ ON にした直後でも、再登録のたびに新しい名前になる。日次の再登録が今夜の分を別名で作ったときは、旧名を取り消し、取り消せなくても両方が届いて第三段の claim が一方を落とす。名前の先頭が乱数（端末 id の UUID）なので、Cloud Tasks が名前付きタスクに求める分散も満たす。

第二段は DB との突き合わせである。コールバックは body の `notify_time` / `time_zone` と現在の端末行を比べ、違えば `stale` として何もしない。古い設定で予約されたタスクは、取り消しに失敗しても無害である。したがって取り消し（`Cancel`）はベストエフォートでよく、正は常に DB の行である。

第三段は `devices` 行の claim 列（`reminder_date` / `reminder_status` / `reminder_attempts` / `reminder_claimed_at` / `reminder_error`）である。`ClaimReminder` は一文の条件付き `UPDATE` で、「その日の claim がまだ無い」「`failed` で試行回数が 3 未満」「`pending` のまま 5 分過ぎた（送信中に落ちた）」のいずれかのときだけ行を取る（`ClaimWon`）。取れなかったときは、生きている `pending` が居るなら `ClaimHeld`、その日が済んでいるなら `ClaimDone` を返す。送信後は `sent` / `skipped` / `failed` を書く。二つのコールバックが同時に来ても、`UPDATE` は一方にしか当たらない。

再試行は次の順で働く。一時失敗はまず `failed` を書いて 503 を返す。Cloud Tasks が再送すると claim が「`failed` で 3 未満」に当たり、再び送る。`ClaimHeld` にも 503 を返す。送信中に落ちた `pending` を 5 分経つ前の再送が「済み」と誤認して 200 を返すと再試行が止まり、その夜が失われるためである。送信は成功したが `sent` を書く前に落ちた場合は `pending` が残り、5 分後の再送で claim が通って二回目が送られうる。この重複は `apns-collapse-id` / FCM の `collapse_key`（いずれも `diary_reminder`）で端末のトレイ上一件に畳まれる。送信後の DB 書き込み失敗という稀な事象のために送信前に `sent` を書く（at-most-once）と、逆に「送っていないのに送ったことになる」が起きるので、at-least-once と畳み込みの組を選ぶ。

**根拠**: 名前で予約の重複を、行の突き合わせで陳腐化を、claim 列で同日の重複をそれぞれ止めるので、どの段の障害でも二回目が届かない（N1）。再試行は Cloud Tasks に任せ、サーバーは 200 / 503 の区別だけを返せばよい（N2）。

**棄却案**:

- 配信履歴の別表 `push_deliveries`。通知の種類が一つで、端末ごと日ごとに一件しか無い今は、`devices` の列で足りる。表を増やすと store・fake・契約テストがもう一組増える。種類が増えたら抽出する（将来の拡張）。
- 予約時に「送る」と決めておき、コールバックでは送るだけにする。記録は予約から配信までの間に書かれるので、判定は配信時刻でなければならない。

### Q4: 端末をどう同定し、誰の端末かをどう決めるか（R1, N4）

**アプローチ**: プッシュ通知のトークンは OS が任意に更新し、同じ端末で別の利用者がサインインしうる。「端末」と「利用者」と「トークン」の対応を、更新と入れ替わりに耐える形で決める。

**設計**: 端末の同一性は `install_id` で表す。クライアントがインストール時に一度生成する UUID で、multiplatform-settings に保存し、トークンが変わっても変わらない。`devices` の一意キーは `(user_id, install_id)` で、再登録は同じ行を更新する。

`push_token` は表全体で一意にする。登録時、同じトークンを別の `(user_id, install_id)` が持っていればその行を同じトランザクションで削除する。同じ端末で別の利用者がサインインすると、トークンは新しい利用者に移り、前の利用者の行は消える。前の利用者のリマインドが次の利用者の端末に届くことはない。

`time_zone` は必須で、`PUT` が `time.LoadLocation` で検証し、`""` と `"Local"` を拒む。どちらも読み込めてしまうが、意味は UTC とサーバーのゾーンで、深夜に鳴る誤りになる。端末はフォアグラウンド復帰のたびにタイムゾーンを読み直す。

配送サービスがトークンを無効と答えた（`Unregistered` / `BadDeviceToken` / `UNREGISTERED`）ときは、行を消さず `token_invalid_at` を立てる。予約は止まり、次の `PUT` で解除される。行を消すと、APNs の環境（sandbox / production）を取り違えただけで全 iOS 利用者の設定が失われ、原因を示す痕跡も消えるためである。

サインアウトは `POST /auth/logout` の body に `install_id` を足して行う。この経路は認証を要さないので、`refresh_token` のハッシュから利用者を引け、そのトークンが失効も期限切れもしていないときだけ、その利用者のその `install_id` の行を削除する。退会は `AccountService.DeleteAccount` が利用者の全端末を削除してから利用者を消す（DB の cascade だけでは予約の取り消しに必要なタスク名を読めない）。利用者の操作を伴わない強制サインアウト（401 による失効）は端末側で解除できないので、`updated_at` が 45 日より古い行は配信しない。端末は日次で再登録するため、生きている端末は常に新しい。

**根拠**: `install_id` を主キーにするとトークン更新で行が増えず、`push_token` の全体一意で入れ替わりを閉じる（N4）。無効トークンを消さないのは、誤設定からの復旧を一つの `UPDATE` にとどめるためである。

**棄却案**:

- `DELETE /api/v1/devices/{install_id}` を足す。サインアウト時にアクセストークンが期限切れなら呼べず、`logout` 本文で同時に済むので、口を増やす理由が無い。
- `time_zone` に既定値 `Asia/Tokyo` を置く。クライアントの送り忘れが「間違った時刻に鳴る」として現れ、検証で落とす方が早く気づける。
- 無効トークンの行を削除する。上記のとおり誤設定で設定が全滅する。

### Q5: APNs・FCM・Cloud Tasks をどう呼ぶか（N3, N5）

**アプローチ**: 三つの外部サービスに対して、依存を最小に、テストを fake で回せる形に揃える。

**設計**: `internal/push` に `Sender { Send(ctx, token, Notification) }` を置き、`apns` と `fcm` が実装する。`internal/schedule` に `Scheduler { Schedule(ctx, Task); Cancel(ctx, name) }` を置き、`cloudtasks` が実装する。いずれも REST を `net/http` で直接呼び、`memobject` と同じ型の記録用 fake（`mempush` / `memschedule`）を持つ。

- APNs: `POST /3/device/{token}` を HTTP/2 で呼ぶ（Go の `http.Client` は TLS で自動的に h2 を交渉する）。認証は Apple の Auth Key（`.p8`）で署名した ES256 の JWT で、`golang-jwt` を使い、50 分キャッシュし、`ExpiredProviderToken` の 403 には一度だけ作り直して再送する（Apple は 20 分より短い間隔の更新も拒む）。ヘッダは `apns-topic`（Bundle ID）、`apns-push-type: alert`、`apns-priority: 10`、`apns-collapse-id`、`apns-expiration`。body は `aps.alert{title, body}` と `aps.sound: default`、それに `kind` / `date` / `title` / `body`。
- FCM: HTTP v1 の `projects/{id}/messages:send` を呼ぶ。認証はサービスアカウント鍵による OAuth2 で、`golang.org/x/oauth2/jwt` の `TokenSource` を使う（唯一の新規依存。`x/oauth2/google` は `cloud.google.com/go` を引き込むので使わない）。メッセージは `notification` ブロックを持たない**データ通知**で、`android.priority: HIGH`、`ttl`、`collapse_key` を付ける。Android アプリが通知を組み立てるので、Q1 の端末側判定と Android 13 以降の通知権限の確認を挟める。
- Cloud Tasks: `POST /v2/projects/{p}/locations/{l}/queues/{q}/tasks` で `httpRequest{url, headers, body(base64)}` と `scheduleTime` を持つタスクを作る。409 は成功、`DELETE` の 404 も成功として扱う（Q3）。FCM と同じサービスアカウント鍵を scope だけ変えて使う。

サーバーはコールバックを `X-Push-Callback-Token` の定数時間比較で守る（既存の `RequireAdmin` と同型）。トークンは Render に生成させ、Cloud Tasks のタスクにヘッダとして焼かれる。

設定はすべて環境変数で、鍵は base64 で渡す（Render の環境変数は一行のため）。APNs の鍵が無ければ iOS 向け `Sender` が無く、GCP の鍵が無ければ FCM と予約が無い。無い部分だけが無効になり、他は動く（`ObjectStore` の nil パターン）。`PUSH_ENABLED=false` は全体のキルスイッチである。

通知の本文は `PushService` の定数（`月が出ました` / `今日を、そっと綴りませんか。`）で、日記の内容は payload に含めない（N3）。

**根拠**: 依存を `x/oauth2` 一つに抑え、fake で router まで通すテストが書ける（既存の `appleFake` と同じ型）。データ通知にすることで、Android の表示判断を端末に残せる。

**棄却案**:

- iOS も Firebase 経由で送る（FCM の APNs 連携）。iOS アプリに Firebase SDK が入り、Google への依存が iOS にも広がる。APNs 直結は Go の標準ライブラリと既存の `golang-jwt` で書ける。
- `google-cloud-go` / `firebase-admin-go` を使う。gRPC を含む依存の広がりが大きく、REST 三本のために持ち込む量ではない。
- FCM の `notification` ブロックで OS に表示させる。バックグラウンドでは `onMessageReceived` を通らず、端末側の判定（Q1）を挟めない。
- コールバックを Cloud Tasks の OIDC トークンで検証する。既存の Google ID トークン検証（`internal/auth/oidc.go`）を再利用できるが、サービスエージェントへの IAM 付与が増える。共有秘密で足りるので、強化案として将来に置く。

### Q6: 端末側で何を判断するか（R1, R3, N4）

**アプローチ**: 端末の責務を「登録」「表示の可否」「タップ後の遷移」の三つに絞り、それぞれの置き場所を決める。

**設計**: 登録は `shared` の `DeviceRegistrar`（Koin の `single(createdAtStart = true)`）が担う。`authState`・`PushTokenStore.token`・`reminderEnabled`・`reminderTime`・`NetworkMonitor.isOnline` を `combine` し、認証済み・トークンあり・オンラインのとき `Fingerprint(userId, token, enabled, time, tz)` を作る。保存済みの指紋と違うか、保存から 24 時間過ぎていれば `PUT /devices` を送り、成功時だけ指紋を保存する。トークンが認証より先に届いても後に届いても同じ結果になる。指紋に `userId` を含めるのは、同じ端末で利用者が替わったときに再登録するためである。`Unauthenticated` になったら指紋を消す。

トークンの取得は OS 固有で、共通のインターフェイスは作らない。`shared` は sink（`PushTokenStore.set`）だけを持ち、Android は `FirebaseMessaging.token` と `onNewToken` から、iOS は `AppDelegate` の `didRegisterForRemoteNotificationsWithDeviceToken` から書き込む。Android の Firebase は `google-services` プラグインを使わず、既存の `-P` → `BuildConfig` の型で `FirebaseApp.initializeApp` に値を渡す。値が空なら初期化せず、CI は秘密なしで通る。

表示の可否は Android だけが判断する（Q1）。`RunaMessagingService` は `kind == diary_reminder` 以外を無視し、セッションが無ければ表示せず、当日の記録があれば表示せず、それ以外で通知チャネル `runa.reminder.nightly` に投稿する。文言は payload の `title` / `body` を使い、欠けていれば `ReminderNotificationText` に戻る。

タップ後の遷移は `PendingRoute`（`StateFlow<PendingRouteKind?>`、sticky）で運ぶ。Android は `MainActivity` が `runa://diary/editor` の intent（`onCreate` と `onNewIntent` の両方）から、iOS は `AppDelegate` の `didReceive` から `set` する。消費するのは、Android は `RunaAuthenticatedApp`（`AppLockGate` の内側）、iOS は `DiaryListView`（`LockGateView` と `RootView` の内側）で、ロック解除と認証を抜けるまで自然に待つ。`Unauthenticated` になったら消費して捨て、後のサインインで発火しない。iOS では `UNUserNotificationCenter.delegate` を `didFinishLaunching` の中で設定する。設定が遅れると、コールドスタート時のタップが失われる。

**根拠**: 登録を指紋の差分にすると、起動のたびに無条件で `PUT` を送らずに済み、変化があった時だけ通信する（R1）。`PendingRoute` を sticky にするのは、通知のタップが「ロック中」「認証復元中」に届き、その時点では遷移できないからである（R3）。

**棄却案**:

- `PushTokenProvider` を `expect class` のまま実装する。`docs/adding-a-feature.md` のとおり `expect class` はコンストラクタ注入も fake も効かず、iOS はそもそも「取りに行く」API ではなくコールバックで受け取る。
- 通知のタップを一回限りのイベントで流す。ロック中は購読者がいないので落ちる。
- `google-services.json` とプラグインで Firebase を初期化する。秘密ファイルをビルドに要し、CI にも配る必要が生じる。

### Q7: ローカル通知を残すか（N1）

**アプローチ**: サーバー起点に切り替えた後、端末のローカル通知を併存させるかを決める。

**設計**: 残さない。`LocalNotificationScheduler` と Android の `AlarmManager` / `ReminderReceiver` / `BootReceiver`、iOS の `UNCalendarNotificationTrigger` の予約は削除する。`DefaultNotificationSettingsRepository` は設定の保存だけになる（キーは変えず、既存の設定は引き継ぐ）。Android の通知の組み立て（チャネル・アイコン・投稿）は `RunaMessagingService` に移す。

**根拠**: 二つの経路が並ぶと、同じ夜に二回鳴らさないための解除漏れの管理が要り、N1 を守る対象が増える。サーバーが送れない状況でリマインドが鳴らない後退は、PRD がリスクとして受け入れている。

**棄却案**:

- トークンが未取得の間だけローカル通知を張る。40 行ほどで書けるが、トークン取得後の解除が漏れると二重に鳴る。Play サービスの無い端末は PRD が責務の外に置いている。

## マイグレーション

0009 として次を行う。

1. `devices` の全行を削除する。クライアントはこの表に一度も登録していない（`shared` に `/devices` の呼び出しが無かった）ので、失うものは無い。
2. `install_id UUID NOT NULL`、`time_zone TEXT NOT NULL`、`token_invalid_at`、`next_task_name` / `next_fire_at`、`reminder_date` / `reminder_status` / `reminder_attempts` / `reminder_claimed_at` / `reminder_error` を足す。
3. 一意インデックスを `(user_id, push_token)` から `(user_id, install_id)` と `(push_token)` に替える。

claim 列と `next_*` は `updated_at` を動かさない。`updated_at` は端末の生存判定（Q4 の 45 日）に使うので、サーバー自身の書き込みで若返ってはならない。

OpenAPI は 0.9.0 になる（`RegisterDeviceRequest` / `Device` に `install_id` と `time_zone`、`LogoutRequest.install_id`、`/api/v1/hooks/push/reminder` と `ReminderTask` / `ReminderOutcome`）。

## 障害シナリオとエッジケース

| 場面 | 振る舞い |
|---|---|
| コールバック時に Render がコールドスタート中 | Cloud Tasks が 30 秒〜10 分のバックオフで最長 60 分再試行する。起きたら通常どおり判定して送る |
| 選んだ時刻から 60 分過ぎてコールバックが届く | `expired` として送らず、翌日分だけ予約する（深夜に鳴らない） |
| 送信後に DB へ `sent` を書けなかった | 5 分以内の再送は 503（`ClaimHeld`）で待たされ、5 分後に `pending` が回収されて再送されうる。`collapse-id` でトレイ上は一件 |
| 翌日分の予約に失敗した | 503 で再試行。今夜のタスクは取り消していないので再試行が来て、claim は済み（`ClaimDone`）なので予約だけをやり直す |
| リマインドを OFF にしてすぐ ON にした | 再登録で `updated_at` が変わり別名で予約される（Cloud Tasks の名前再利用制限に当たらない） |
| APNs / FCM が 5xx・429 | `failed` を書いて 503。再試行は 3 回まで、60 分以内 |
| APNs 鍵の失効・環境の取り違え | 全 iOS 端末が `invalid_token` になり `token_invalid_at` が立つ。行と設定は残り、鍵を直して端末が再登録すれば戻る。Cloud Tasks コンソールに失敗として残る |
| 端末が圏外で配信時刻を迎えた | 配送サービスが `ttl`（選んだ時刻 + 60 分）まで保持し、それを過ぎたら捨てる |
| 0 時をまたぐ（23:50 設定、23:58 に予約、0:05 に配信） | 予約は絶対時刻なので当日分として届く。判定日は body の `local_date`（前日）で行う |
| 夏時間の切り替え日 | 予約時刻は `time.Date` の正規化に従う。存在しない時刻は隣接する時刻になり、重複する時刻は一方になる。判定日は変わらない |
| 端末がタイムゾーンを変えた | フォアグラウンド復帰で `DeviceRegistrar.refresh()` が指紋の差分を検出し再登録する。旧タスクは `stale` |
| 時刻を変えた直後に旧タスクが届く | body の `notify_time` が行と違うので `stale` |
| 同じ端末で別の利用者がサインイン | トークンが新しい利用者に移り、前の利用者の行は消える。前の利用者のタスクは `stale` |
| 強制サインアウト（401 失効） | 端末は解除できない。再登録が止まって 45 日で対象外。Android はセッションが無ければ表示しない。iOS には最長 45 日、無害な文言が届きうる |
| オフラインで書いた直後に配信時刻 | サーバーは記録を知らない。Android は端末の記録を見て表示しない。iOS には届く（PRD の前提条件） |
| 複数の端末を持つ利用者 | 端末ごとに判定し、記録があれば全端末が `skipped_written`。無ければ全端末に届く |
| 通知権限を拒否した端末 | トークンは発行されるので登録は行われ、サーバーは送るが OS が表示しない |
| `PUSH_ENABLED=false`、またはその OS の鍵が未設定 | 登録は保存だけ行い予約しない。コールバックは `disabled`。鍵を設定した後は端末の日次再登録で予約が始まる |
| 自己継続の連鎖が切れた | 端末の日次再登録（指紋の 24 時間）が翌日までに繋ぎ直す |

## 将来の拡張

- 通知の種類が増えたら（今日の言葉・きょうの一曲の公開、うつろいの手紙）、`devices` の claim 列を `push_deliveries(device_id, kind, local_date)` に抽出し、`ReminderTask` に `kind` を持たせる。`Sender` / `Scheduler` はそのまま使える。
- コールバックの検証を共有秘密から Cloud Tasks の OIDC トークン（`internal/auth/oidc.go` の Google 検証器を再利用）に替える。
- タップの計測（`POST /push/opened`）。`PendingRoute` を消費する箇所から送れる。
- 設定を利用者単位にする場合は `users` 側に設定を持ち、`devices` は宛先だけにする。判定と claim は端末ごとのまま使える。
- iOS で未同期の記録による誤配信を止めるには、Notification Service Extension と `com.apple.developer.usernotifications.filtering` entitlement（Apple の承認が要る）が必要になる。
