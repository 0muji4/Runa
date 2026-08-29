#!/bin/sh
# Android の strings.xml（UI 文言の唯一の正典）から iOS 用の Strings.swift を生成する。
# iOS には文言リソースの仕組みが無く、放っておくと SwiftUI の literal が増えて
# Android と乖離するため、片側を生成物にして「1 ファイル編集で両 OS に効く」状態を作る。
#
#   ./hack/gen-ios-strings.sh            # 既定の出力先に書く（= make ios-strings）
#   ./hack/gen-ios-strings.sh /tmp/out   # 出力先を指定（ドリフト検査で使う）
#
# 生成物はコミットする（Xcode ビルド時にコード生成を走らせないため）。
# 整合は hack/check-ui-strings.sh が検証する。
set -eu

root=$(CDPATH= cd "$(dirname "$0")/.." && pwd)
xml="$root/apps/kotlin/androidApp/src/main/res/values/strings.xml"
out=${1:-"$root/apps/swift/Runa/Strings.swift"}

[ -f "$xml" ] || { printf '\033[1;31mError: %s が見つからない\033[0m\n' "$xml" >&2; exit 1; }

awk '
  # snake_case -> lowerCamelCase
  function camel(s,   n, p, i, r) {
    n = split(s, p, "_")
    r = p[1]
    for (i = 2; i <= n; i++) r = r toupper(substr(p[i], 1, 1)) substr(p[i], 2)
    return r
  }

  # XML エンティティと Android のエスケープを元に戻す。\n は Swift でもそのまま通る。
  function unesc(s) {
    gsub(/\\'"'"'/, "'"'"'", s)
    gsub(/&quot;/, "\"", s)
    gsub(/&apos;/, "'"'"'", s)
    gsub(/&lt;/,   "<",  s)
    gsub(/&gt;/,   ">",  s)
    gsub(/&amp;/,  "\\&", s)
    return s
  }

  # Swift の文字列リテラルに入れられる形にする。
  function swiftlit(s) {
    gsub(/"/, "\\\\\"", s)
    return s
  }

  # Android の位置指定書式 -> Swift の String(format:) 書式。 %1$s -> %1$@ / %1$d -> %1$ld
  function tofmt(s,   o, spec, ch) {
    o = ""
    while (match(s, /%[0-9]+\$[sd]/)) {
      spec = substr(s, RSTART, RLENGTH)
      ch = substr(spec, RLENGTH, 1)
      sub(/[sd]$/, (ch == "s" ? "@" : "ld"), spec)
      o = o substr(s, 1, RSTART - 1) spec
      s = substr(s, RSTART + RLENGTH)
    }
    return o s
  }

  BEGIN {
    print "// 生成ファイル — 手で編集しない。"
    print "//"
    print "// 正典: apps/kotlin/androidApp/src/main/res/values/strings.xml"
    print "// 再生成: make ios-strings   （整合の検証: ./hack/check-ui-strings.sh）"
    print "//"
    print "// 文言を直すときは strings.xml だけを編集する。1 ファイルの編集が"
    print "// Android と iOS の両方に効く。"
    print ""
    print "import Foundation"
    print ""
    print "enum L {"
  }

  /<string name="[^"]+">.*<\/string>/ {
    key = $0; sub(/^[^"]*"/, "", key); sub(/".*$/, "", key)
    val = $0; sub(/^[^>]*>/, "", val); sub(/<\/string>.*$/, "", val)
    val = unesc(val)

    # 位置指定引数の索引と型を集める。
    delete types
    maxidx = 0
    scan = val
    while (match(scan, /%[0-9]+\$[sd]/)) {
      spec = substr(scan, RSTART, RLENGTH)
      idx = spec; sub(/^%/, "", idx); sub(/\$.*$/, "", idx); idx += 0
      types[idx] = substr(spec, RLENGTH, 1)
      if (idx > maxidx) maxidx = idx
      scan = substr(scan, RSTART + RLENGTH)
    }

    name = camel(key)
    if (maxidx == 0) {
      printf "    static let %s = \"%s\"\n", name, swiftlit(val)
      next
    }

    sig = ""
    args = ""
    for (i = 1; i <= maxidx; i++) {
      t = (types[i] == "d") ? "Int" : "String"
      sig  = sig  (i > 1 ? ", " : "") "_ a" i ": " t
      args = args ", a" i
    }
    printf "    static func %s(%s) -> String {\n", name, sig
    printf "        String(format: \"%s\"%s)\n", swiftlit(tofmt(val)), args
    printf "    }\n"
  }

  END { print "}" }
' "$xml" > "$out"

printf '\033[1;32mgenerated: %s\033[0m\n' "$out"
