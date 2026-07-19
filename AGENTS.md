# Runa リポジトリ ガイド（エージェント向け）

## commit / push / PR / Issue を作るとき（最重要）

- IMPORTANT: ローカル変更を push / PR / Issue にするタスクでは、コマンドを自作せず**必ず `runa-pr` スキルを起動する**。「push して」「PR にして」「まとめて」等の短い依頼でも同じ。
- IMPORTANT: git / gh の**状態変更（branch 作成・commit・push・PR/Issue 作成・merge）は実行しない**。Codex は調査・不整合の修正・**正確なコマンド列の出力**までを担い、実行はユーザーが行う。`git status` / `diff` / `log` などの読み取りは可。
- 規約の唯一の出どころは [CONTRIBUTING.md](CONTRIBUTING.md) と [hack/prefix.yaml](hack/prefix.yaml)。ここに要点を再掲するが、齟齬があれば原典を優先する。

## 規約の要点（CI が全 PR で強制する）

- **直線履歴**: マージコミット禁止。マージは `--rebase`（`--squash` 不可 — 件名の末尾ピリオドが落ちて規約違反になる）。
- **1 Issue → 1 PR → 1 commit** を一対一で対応させ、タイトルを一致させる。1 PR が複数論点にまたがるなら PR を分ける。
- **タイトル接頭辞** `[カテゴリ]`: Issue・PR・コミットに付ける。許可カテゴリは `hack/prefix.yaml` が唯一の出どころ（現状: `Android` / `iOS` / `Server` / `Infra` / `Docs` / `CI/CD` / `Chore`）。ハードコードしない。
- **コミット件名 = PR タイトル + 末尾ピリオド**。`TITLE` 変数で完全一致させる。
- **`Closes #<番号>`**: PR 本文で対応 Issue を必ず参照する（無いとマージ不可）。

### カテゴリ境界

- `Android` / `iOS` … 各 OS のクライアント（両方にまたがるなら複合 `[Android/iOS]`）。
- `Server` … Go バックエンド（`apps/go`。API・認証・データ保存・DB スキーマなど）。
- `Infra` … クラウド・デプロイ・マネージド DB・スケジュール実行など、アプリロジックの外側の土台。
- `Docs` … PRD / DD / README などのドキュメント。
- `CI/CD` … GitHub Actions 等のパイプライン。
- `Chore` … ツール・設定・雑務（`.gitignore`・formatter・依存更新など）。

## 事前検証

- `Server`（Go）を触る PR: `cd apps/go && go vet ./... && go test ./...`
- `Android` / `iOS` を触る PR: 各クライアントのビルド・テストを実行する。
