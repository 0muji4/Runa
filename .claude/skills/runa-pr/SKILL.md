---
name: runa-pr
description: >-
  Runa リポジトリで、今ローカルにある変更を規約準拠の Pull Request にまとめる。
  ユーザーが変更を push / ship / land / PR したいとき必ず使う。例:
  「今ローカルにある修正を push したい」「push して」「PR にして」「この変更まとめて」
  「ローカルの変更を PR にして」「make a PR for my changes」「これ ready なので出したい」。
  git status / diff を見て論点ごとに 1 PR に分割し、hack/prefix.yaml から [カテゴリ] 接頭辞を選び、
  規約・用語の不整合を点検し、実行コマンド列を生成してユーザーに渡す。
  Claude は git/gh の状態変更を実行せず、コマンドを渡すだけ。「push」「PR」だけの短い依頼でも発動する。
---

# Runa: ローカル変更を規約準拠 PR にする

## 大原則: あなたは「準備」、ユーザーが「実行」

Claude は変更を調査し、ドキュメント/コードの不整合を編集で直し、**正確なコマンド列を出力する**。
commit・push・branch 作成・PR/Issue 作成・merge といった **git/gh の状態変更はユーザー自身が実行する**。
これらを Claude が勝手に実行してはいけない（このリポジトリの確立された運用方針）。
`git status` / `diff` / `log` などの読み取り確認は行ってよい。

## 規約（出どころ）

詳細は [CONTRIBUTING.md](../../../CONTRIBUTING.md) と [hack/prefix.yaml](../../../hack/prefix.yaml)。CI が全 PR で強制する。生成するコマンドは必ず次を満たすこと:

- **直線履歴**: マージコミット禁止。マージは必ず `--rebase`。
- **1 PR = 1 commit**: 複数論点なら PR を分ける。1 PR の中は 1 コミット。
- **コミット件名 = PR タイトル + 末尾ピリオド**。`TITLE` 変数で完全一致させ、手入力のずれを防ぐ。
- **`[カテゴリ]` 接頭辞**: Issue・PR・コミットのタイトルに付ける。許可カテゴリは `hack/prefix.yaml` が唯一の出どころ。
- **`Closes #<番号>`**: PR 本文で対応 Issue を必ず参照する。
- **マージは `--rebase`（`--squash` 不可）**: squash は GitHub が件名を再生成して末尾ピリオドを落とし、規約と不整合になる。rebase なら検証済みの 1 コミットがそのまま main に載る。

## 手順

### 1. 現状を調べる（読み取り）

```sh
git rev-parse --abbrev-ref HEAD
git status --short
git diff --stat && git diff
git rev-list --left-right --count origin/main...main
```

未追跡ディレクトリは `git add -n <dir>` で「実際に何が追加されるか」を dry-run 確認する（`.gitignore` の除外を反映するため）。

### 2. 論点ごとに分割する（1 PR = 1 論点・単一カテゴリ）

変更が複数の無関係な関心事にまたがるなら、**PR を分ける**。関連する変更（例: 1 つの仕様変更が doc とコードにまたがる）は **1 PR にまとめる**——分けると 1 つの変更が複数 PR になり、かえって規約の趣旨に反する。
複数 PR になる場合は順序を明示し、各 PR をマージしてから次へ進む流れで出す。

### 3. カテゴリを選ぶ

`hack/prefix.yaml` を読んで現在の許可カテゴリから選ぶ（ハードコードしない。実態とずれるため）。
現状: `Android` / `iOS` / `Server` / `Infra` / `Docs` / `CI/CD` / `Chore`。Android と iOS の両方に関わる変更は複合 `[Android/iOS]`。
境界: `Android`/`iOS`=各 OS のクライアント、`Server`=Go バックエンド、`Infra`=クラウド/デプロイ/マネージド DB/スケジュール実行、`Chore`=ツール・設定・雑務。

### 4. 不整合を点検する

push 前に、変更が既存の規約・文書と矛盾しないか確認する。

- **用語のずれ**: PRD/DD で確定した用語の言い換えが新規ファイルに混入していないか。`grep -rn '<旧語>' <変更パス>` で確認。
- **CONTRIBUTING.md / README との矛盾**。

見つけたら **Claude がファイル編集で直す**（git 操作ではない）。直したファイルも同じ PR に含める。

### 5. 事前検証（コード変更のとき）

- `Server`（Go）を触る PR: `cd apps/backend && go vet ./... && go test ./...`
- `Android` / `iOS` を触る PR: 各クライアントのビルド・テストを実行する。

（コード用 CI は各コードが入る PR で追加する。詳細は CONTRIBUTING.md「コードの CI」。）

### 6. コマンド列を生成して渡す

論点ごとに次のテンプレートを埋めて出す。`TITLE` は接頭辞付き・ピリオドなし。Issue 番号は自動取得。`<files>` はその論点のファイルだけ（`git add -A` や `.` は使わない。`.gitignore` 除外も尊重）。

```sh
cd /Users/motoshi.suzuki/go/src/github.com/0muji4/Runa

TITLE='[カテゴリ] 変更の要約'
ISSUE_URL=$(gh issue create --title "$TITLE" --body '<Issue 本文>')
ISSUE_NUM=$(basename "$ISSUE_URL"); echo "issue=#$ISSUE_NUM"

git switch -c <category>/<slug>
git add <files>
git status --short

git commit -m "${TITLE}." -m '<本文・任意>'
git push -u origin <category>/<slug>

gh pr create --base main --head <category>/<slug> --title "$TITLE" --body "Closes #${ISSUE_NUM}"
gh pr merge --auto --rebase --delete-branch
gh pr checks --watch
git switch main && git pull
```

`gh pr merge --auto` で auto-merge を予約し、必須ステータスチェック（`validate-pr`）が緑になってから自動マージさせる。手動 `gh pr merge` は使わない（CI 完了前にマージされるのを防ぐ）。auto-merge はリポジトリの「Allow auto-merge」有効化と、`main` の必須チェック設定（branch protection / ruleset）が前提。

## ブートストラップの注意

`hack/prefix.yaml` に**新カテゴリを追加する**変更のとき、その Issue 作成時点では新カテゴリがまだ main に無く、issue タイトル検証 workflow（main を参照）が赤くなる。その PR は**既存カテゴリ**（例 `[CI/CD]` や `[Chore]`）で出し、**最初にマージ**する。以降の PR で新カテゴリが使える。

## ブランチ名

`<category-lowercase>/<slug>`（例 `docs/add-prd`、`server/auth-endpoint`）。複合カテゴリは `android-ios/<slug>` のようにつなぐ。Issue 番号を入れる運用なら `<category>/<issue#>-<slug>`。
CI 強制はしていないが、マージ済み PR 記録に残るので一貫させる。
