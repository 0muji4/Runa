#!/bin/sh
# 画面ヘッダーのトークンが、README の正典表と各クライアントの定義で一致するかを
# 検証する。値はジェネレータを持たず 4 ファイルに手動複製しているため、編集での
# 乖離（ドリフト）を防ぐ。
# 参照: README.md「画面ヘッダー（全画面共通の型）」を唯一の正典とする。
#
# あわせて、ヘッダーが共通コンポーネント 1 箇所に閉じているかも見る。画面が自前で
# タイトル体裁や上下余白を書き始めると、値が合っていても見た目は再びばらけるため。
set -eu

root=$(CDPATH= cd "$(dirname "$0")/.." && pwd)
cd "$root"

readme="README.md"
type_kt="apps/kotlin/androidApp/src/main/kotlin/com/runa/android/ui/theme/Type.kt"
dimens_kt="apps/kotlin/androidApp/src/main/kotlin/com/runa/android/ui/theme/RunaDimens.kt"
fonts_swift="apps/swift/Runa/Theme/RunaFonts.swift"
colors_swift="apps/swift/Runa/Theme/RunaColors.swift"
header_kt="apps/kotlin/androidApp/src/main/kotlin/com/runa/android/ui/components/RunaScreenHeader.kt"
header_swift="apps/swift/Runa/Theme/RunaScreenHeader.swift"

error() { printf '\033[1;31mError: %s\033[0m\n' "$1" >&2; }
ok()    { printf '\033[1;32m%s\033[0m\n' "$1"; }

for f in "$readme" "$type_kt" "$dimens_kt" "$fonts_swift" "$colors_swift" "$header_kt" "$header_swift"; do
  [ -f "$f" ] || { error "ファイルが見つからない: $f"; exit 1; }
done

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# ---- 1) 値の一致 -------------------------------------------------------------
#
# 各ソースを "token 値" の正規化行に落として突き合わせる。
# headerTitleLineHeight だけは Android 固有（SwiftUI はフォント本来の行送りに任せ、
# 明示指定を持たない）なので iOS 側の期待からは外す。

# README 正典表 → 7 行。
awk -F'|' '
  /^\|[ ]*header[A-Za-z]+[ ]*\|/ {
    tok=$2; gsub(/[^A-Za-z]/, "", tok)
    val=$3; gsub(/[^0-9]/, "", val)
    print tok " " val
  }
' "$readme" | sort > "$tmp/readme"

[ "$(wc -l < "$tmp/readme")" -eq 7 ] || {
  error "README の正典表から 7 トークンを読めなかった（$(wc -l < "$tmp/readme") 行）。表の形式を確認すること。"
  exit 1
}

# Android: Type.kt の headlineLarge / labelMedium と RunaDimens.kt の RunaHeader。
{
  awk '
    function num(s){ sub(/.*= */, "", s); sub(/\.sp.*/, "", s); return s }
    /headlineLarge[ ]*=[ ]*TextStyle/ { role="title"; next }
    /labelMedium[ ]*=[ ]*TextStyle/   { role="label"; next }
    /^[ ]*\),?[ ]*$/                  { role="" }
    role == "title" && /fontSize[ ]*=/   { print "headerTitleSize " num($0) }
    role == "title" && /lineHeight[ ]*=/ { print "headerTitleLineHeight " num($0) }
    role == "label" && /fontSize[ ]*=/   { print "headerLabelSize " num($0) }
  ' "$type_kt"
  awk '
    function num(s){ sub(/.*= */, "", s); sub(/\.dp.*/, "", s); return s }
    /val TopTab[ ]*=/    { print "headerTopTab " num($0) }
    /val TopPushed[ ]*=/ { print "headerTopPushed " num($0) }
    /val BackGap[ ]*=/   { print "headerBackGap " num($0) }
    /val Bottom[ ]*=/    { print "headerBottom " num($0) }
  ' "$dimens_kt"
} | sort > "$tmp/android"

# iOS: RunaFonts.swift の screenTitle / headerLabel と RunaColors.swift の RunaHeaderMetrics。
{
  awk '
    function arg(s){ sub(/.*\(/, "", s); sub(/[^0-9].*/, "", s); return s }
    /static let screenTitle[ ]*=/ { print "headerTitleSize " arg($0) }
    /static let headerLabel[ ]*=/ { print "headerLabelSize " arg($0) }
  ' "$fonts_swift"
  awk '
    function num(s){ sub(/.*= */, "", s); sub(/[^0-9].*/, "", s); return s }
    /static let topTab[ ]*:/    { print "headerTopTab " num($0) }
    /static let topPushed[ ]*:/ { print "headerTopPushed " num($0) }
    /static let backGap[ ]*:/   { print "headerBackGap " num($0) }
    /static let bottom[ ]*:/    { print "headerBottom " num($0) }
  ' "$colors_swift"
} | sort > "$tmp/ios"

# iOS は行高を持たないので、比較の左辺からも外す。
grep -v '^headerTitleLineHeight ' "$tmp/readme" > "$tmp/readme-ios"

status=0
if ! diff -u "$tmp/readme" "$tmp/android" > "$tmp/d-android" 2>&1; then
  error "README の正典と Android の定義が食い違っている（$type_kt / $dimens_kt）。"
  sed '1,2d' "$tmp/d-android" >&2
  status=1
fi
if ! diff -u "$tmp/readme-ios" "$tmp/ios" > "$tmp/d-ios" 2>&1; then
  error "README の正典と iOS の定義が食い違っている（$fonts_swift / $colors_swift）。"
  sed '1,2d' "$tmp/d-ios" >&2
  status=1
fi

# ---- 2) ヘッダーが共通コンポーネントに閉じているか ---------------------------
#
# タイトル体裁と余白トークンは、共通ヘッダー（と定義元、および見出しを持たない
# ホーム）以外から参照されてはいけない。

check_confined() {
  desc=$1; pattern=$2; root_dir=$3; shift 3
  # 許可ファイルを除いた参照を探す。
  found=$(grep -rl "$pattern" "$root_dir" || true)
  for f in $found; do
    allowed=no
    for a in "$@"; do
      [ "$f" = "$a" ] && allowed=yes
    done
    if [ "$allowed" = no ]; then
      error "$desc は共通ヘッダーの外から参照されている: $f"
      error "（画面は RunaScreenHeader を呼ぶこと。値の直参照はヘッダーの型を崩す）"
      status=1
    fi
  done
}

check_confined "Android の画面タイトル体裁 (headlineLarge)" "headlineLarge" \
  "apps/kotlin/androidApp/src/main" "$type_kt" "$header_kt"
check_confined "iOS の画面タイトル体裁 (RunaFonts.screenTitle)" "RunaFonts\.screenTitle" \
  "apps/swift/Runa" "$fonts_swift" "$header_swift"
check_confined "Android のヘッダー余白 (RunaHeader\.)" "RunaHeader\." \
  "apps/kotlin/androidApp/src/main" "$dimens_kt" "$header_kt" \
  "apps/kotlin/androidApp/src/main/kotlin/com/runa/android/ui/screens/HomeScreen.kt"
check_confined "iOS のヘッダー余白 (RunaHeaderMetrics)" "RunaHeaderMetrics" \
  "apps/swift/Runa" "$colors_swift" "$header_swift" \
  "apps/swift/Runa/Screens/HomeView.swift"

[ "$status" -eq 0 ] || exit 1
ok "Header tokens OK — README の正典と Android/iOS の定義は一致し、ヘッダーは共通コンポーネントに閉じている。"
