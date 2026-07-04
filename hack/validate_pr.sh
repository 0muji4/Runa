#!/bin/sh
# Runa の PR 規約を検証する（詳細は CONTRIBUTING.md、カテゴリは hack/prefix.yaml）。
# CI は REPO と PR_NUMBER を env で渡す。ローカルは open な PR のあるブランチで実行する。
set -eu

dir=$(CDPATH= cd "$(dirname "$0")" && pwd)

error() { printf '\033[1;31mError: %s\033[0m\n' "$1" >&2; }

repo="${REPO:-$(gh repo view --json nameWithOwner -q .nameWithOwner)}"

pr_number="${PR_NUMBER:-}"
if [ -z "$pr_number" ]; then
  head="${HEAD:-$(git rev-parse --abbrev-ref HEAD)}"
  pr_number=$(gh pr list --repo "$repo" --head "$head" --state open --json number -q '.[0].number // empty')
fi
if [ -z "$pr_number" ]; then
  error "open な PR が見つからない（PR_NUMBER を渡すか、PR のあるブランチで実行する）。"
  exit 1
fi

pr=$(gh pr view "$pr_number" --repo "$repo" --json title,body,commits)
pr_title=$(printf '%s' "$pr" | jq -r '.title')
pr_body=$(printf '%s' "$pr" | jq -r '.body // ""')
commit_count=$(printf '%s' "$pr" | jq '.commits | length')
commit_subject=$(printf '%s' "$pr" | jq -r '.commits[0].messageHeadline // ""')

printf 'PR #%s: %s\n' "$pr_number" "$pr_title"

if [ "$commit_count" -ne 1 ]; then
  error "PR は 1 コミットにまとめる必要がある（現在 ${commit_count} 個）。squash すること。"
  exit 1
fi

rest=${commit_subject#"$pr_title"}
if [ "$rest" = "$commit_subject" ]; then
  error "コミット件名は PR タイトルで始める必要がある。
  PR タイトル  : ${pr_title}
  コミット件名 : ${commit_subject}"
  exit 1
fi
case "$commit_subject" in
  *.) : ;;
  *) error "コミット件名は末尾をピリオドにする: ${commit_subject}"; exit 1 ;;
esac

cats=$(grep -E '^[[:space:]]*-[[:space:]]+' "$dir/prefix.yaml" | sed -E 's/^[[:space:]]*-[[:space:]]+//' | paste -sd '|' -)
if ! printf '%s' "$pr_title" | grep -Eq "^\\[(${cats})(/(${cats}))*\\][[:space:]]"; then
  error "PR タイトルはカテゴリ接頭辞で始める。許可: [${cats}]（例: [Docs] 開発ガイドを追加）"
  exit 1
fi

if ! printf '%s' "$pr_body" | grep -Eiq '(close[sd]?|fix(e[sd])?|resolve[sd]?)[[:space:]]+#[0-9]+'; then
  error "PR 本文で Issue を閉じる参照が必要（例: 'Closes #12'）。"
  exit 1
fi

echo "PR validation passed."
