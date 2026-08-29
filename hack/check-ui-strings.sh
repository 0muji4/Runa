#!/bin/sh
# UI 文言の唯一の正典（Android strings.xml）と iOS の整合を検証する。
#
#  1) apps/swift/Runa/Strings.swift が strings.xml から再生成した内容と一致するか
#     （= 生成物の手編集・再生成漏れの検出）
#  2) iOS の Swift に生の非 ASCII リテラル（＝文言）が残っていないか
#     — 文言でないもの（日付フォーマット・記号・曜日/月相の配列）は
#       hack/ui-strings-allow.txt に明示的に列挙する
#
# 非 ASCII の判定は LC_ALL=C の grep（バイト単位）に任せ、awk では ASCII の
# 引用符とコロンしか扱わない。ロケール依存の多バイト正規表現を避けるため。
set -eu

root=$(CDPATH= cd "$(dirname "$0")/.." && pwd)
cd "$root"

gen="hack/gen-ios-strings.sh"
swift="apps/swift/Runa/Strings.swift"
swiftdir="apps/swift/Runa"
allow="hack/ui-strings-allow.txt"

error() { printf '\033[1;31mError: %s\033[0m\n' "$1" >&2; }
ok()    { printf '\033[1;32m%s\033[0m\n' "$1"; }

for f in "$gen" "$swift" "$allow"; do
  [ -f "$f" ] || { error "ファイルが見つからない: $f"; exit 1; }
done

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# ---- 1) 生成ドリフト -------------------------------------------------------

"$gen" "$tmp/Strings.swift" >/dev/null
if ! diff -u "$swift" "$tmp/Strings.swift" > "$tmp/diff"; then
  error "$swift が strings.xml と食い違っている。\`make ios-strings\` を実行して差分をコミットすること。"
  sed -n '1,60p' "$tmp/diff" >&2
  exit 1
fi

# ---- 2) iOS に残った生の文言 ----------------------------------------------

# 候補行: 非 ASCII を含む Swift 行。生成物とコメント行は除く。
LC_ALL=C grep -rn '[^ -~	]' "$swiftdir" --include='*.swift' 2>/dev/null \
  | grep -v "^$swift:" \
  | grep -Ev '^[^:]*:[0-9]+:[[:space:]]*(//|\*|/\*)' \
  > "$tmp/cand" || true

# 候補行を "path<TAB>リテラル" のレコードへ分解する（引用符とコロンは ASCII）。
awk '
  {
    p = index($0, ":"); path = substr($0, 1, p - 1)
    rest = substr($0, p + 1)
    q = index(rest, ":"); code = substr(rest, q + 1)
    n = split(code, part, "\"")
    for (i = 2; i <= n; i += 2) printf "%s\t%s\n", path, part[i]
  }
' "$tmp/cand" > "$tmp/records"

# 非 ASCII を含むリテラルだけを残し、allowlist を差し引く。
LC_ALL=C grep '[^ -~	]' "$tmp/records" > "$tmp/nonascii" || true
grep -v '^[[:space:]]*#' "$allow" | grep -v '^[[:space:]]*$' > "$tmp/allow" || true
sort -u "$tmp/nonascii" > "$tmp/nonascii.u"
grep -F -x -v -f "$tmp/allow" "$tmp/nonascii.u" > "$tmp/offenders" || true

if [ -s "$tmp/offenders" ]; then
  error "iOS に生の文言リテラルが残っている。strings.xml にキーを起こして L.xxx を参照すること。"
  error "（文言でないもの＝フォーマット・記号などは $allow に追加する）"
  sed 's/^/  /' "$tmp/offenders" >&2
  exit 1
fi

# allowlist の腐り止め: 実在しないエントリが残っていたら落とす。
if grep -F -x -v -f "$tmp/nonascii.u" "$tmp/allow" > "$tmp/stale" 2>/dev/null && [ -s "$tmp/stale" ]; then
  error "$allow に、もうコード上に存在しないエントリがある。削除すること。"
  sed 's/^/  /' "$tmp/stale" >&2
  exit 1
fi

ok "UI strings OK — strings.xml と Strings.swift は一致し、iOS に生の文言は残っていない。"
