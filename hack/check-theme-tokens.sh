#!/bin/sh
# 3 テーマ（夜/あさ/ピンク）のトークン値が、README の正典表と各クライアントの
# 定義（Android Color.kt / iOS RunaColors.swift / Android colors.xml）で一致するかを検証する。
# 色の値はジェネレータを持たず 4 箇所に手動複製しているため、編集での乖離（ドリフト）を防ぐ。
# 参照: README.md「3 テーマ（全クライアント共通のトークン）」を唯一の正典とする。
set -eu

root=$(CDPATH= cd "$(dirname "$0")/.." && pwd)

readme="$root/README.md"
kt="$root/apps/kotlin/androidApp/src/main/kotlin/com/runa/android/ui/theme/Color.kt"
sw="$root/apps/swift/Runa/Theme/RunaColors.swift"
xml="$root/apps/kotlin/androidApp/src/main/res/values/colors.xml"

error() { printf '\033[1;31mError: %s\033[0m\n' "$1" >&2; }
ok()    { printf '\033[1;32m%s\033[0m\n' "$1"; }

for f in "$readme" "$kt" "$sw" "$xml"; do
  [ -f "$f" ] || { error "ファイルが見つからない: $f"; exit 1; }
done

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# 各ソースを "theme token HEX"（token 小文字 / HEX 大文字）の正規化行に落とし、突き合わせる。

# 1) README 正典表 → 7 トークン × 3 テーマ = 21 行。
awk -F'|' '
  function hex(s){ gsub(/[^0-9A-Fa-f]/, "", s); return toupper(s) }
  /^\|[ ]*(background|surface|heading|body|subtle|accent|subAccent)[ ]*\|/ {
    tok=$2; gsub(/[^A-Za-z]/, "", tok)
    print "dark "  tolower(tok) " " hex($3)
    print "light " tolower(tok) " " hex($4)
    print "pink "  tolower(tok) " " hex($5)
  }
' "$readme" | sort > "$tmp/readme"

# 2) Android Color.kt（RunaDarkColors / RunaLightColors / RunaPinkColors, 0xFFRRGGBB）。
awk '
  function hx(s){ sub(/.*0xFF/, "", s); sub(/[^0-9A-Fa-f].*/, "", s); return toupper(s) }
  /RunaDarkColors[ ]*=/  { theme="dark" }
  /RunaLightColors[ ]*=/ { theme="light" }
  /RunaPinkColors[ ]*=/  { theme="pink" }
  /=[ ]*Color\(0xFF[0-9A-Fa-f]/ {
    tok=$0; sub(/[ ]*=.*/, "", tok); gsub(/[^A-Za-z]/, "", tok)
    print theme " " tolower(tok) " " hx($0)
  }
' "$kt" | sort > "$tmp/kt"

# 3) iOS RunaColors.swift（static let dark/light/pink, Color(hex: 0xRRGGBB)）。
awk '
  function hx(s){ sub(/.*0x/, "", s); sub(/[^0-9A-Fa-f].*/, "", s); return toupper(s) }
  /static let dark[ ]*=/  { theme="dark" }
  /static let light[ ]*=/ { theme="light" }
  /static let pink[ ]*=/  { theme="pink" }
  /Color\(hex:[ ]*0x[0-9A-Fa-f]/ {
    tok=$0; sub(/:.*/, "", tok); gsub(/[^A-Za-z]/, "", tok)
    print theme " " tolower(tok) " " hx($0)
  }
' "$sw" | sort > "$tmp/sw"

# パーサ破損検知: 3 ソースとも 21 行を抽出できること（表/定義の形式変更を早期に検出）。
for name in readme kt sw; do
  n=$(awk 'END{print NR}' "$tmp/$name")
  if [ "$n" -ne 21 ]; then
    error "$name から 21 行を抽出できず（$n 行）。トークン表/定義の形式が変わった可能性がある。"
    exit 1
  fi
done

# 4) Android colors.xml（起動テーマ）は dark の subset。launcher_background は surface と同値。
awk '
  function hx(s){ sub(/.*>#/, "", s); sub(/<.*/, "", s); return toupper(s) }
  /name="runa_background"/          { print "background " hx($0) }
  /name="runa_surface"/             { print "surface " hx($0) }
  /name="runa_accent"/              { print "accent " hx($0) }
  /name="runa_launcher_background"/ { print "launcher " hx($0) }
' "$xml" | sort > "$tmp/xml"

awk '
  $1=="dark" && ($2=="background" || $2=="surface" || $2=="accent") { print $2, $3 }
  $1=="dark" && $2=="surface" { print "launcher", $3 }
' "$tmp/readme" | sort > "$tmp/xml_expected"

status=0
if ! diff -u "$tmp/readme" "$tmp/kt" > "$tmp/d_kt"; then
  error "Android Color.kt が README 正典表と不一致（< README / > Color.kt）:"; cat "$tmp/d_kt" >&2; status=1
fi
if ! diff -u "$tmp/readme" "$tmp/sw" > "$tmp/d_sw"; then
  error "iOS RunaColors.swift が README 正典表と不一致（< README / > RunaColors.swift）:"; cat "$tmp/d_sw" >&2; status=1
fi
if ! diff -u "$tmp/xml_expected" "$tmp/xml" > "$tmp/d_xml"; then
  error "Android colors.xml（起動テーマ）が README dark と不一致:"; cat "$tmp/d_xml" >&2; status=1
fi

if [ "$status" -eq 0 ]; then
  ok "theme tokens OK: README・Android・iOS で 3 テーマ × 7 トークンが一致（colors.xml 起動テーマも整合）"
fi
exit "$status"
