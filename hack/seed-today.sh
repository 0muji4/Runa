#!/usr/bin/env bash
# Seed curated "today" content (daily quotes + songs) via the admin endpoints, so
# a fresh database shows a quote + song on the home screen. Dates are computed
# relative to the run date (today + the next few days), so it works on whatever
# calendar day you run it — GET /today matches an exact date.
#
# Songs are tracks from Apple's catalog, registered by iTunes track id; the
# server fetches title/artist/artwork/preview from the iTunes Search API. The
# default rotation below is 乃木坂46 — Runa takes its name and moon from 林瑠奈
# (4期生), so the days open with the night, then two 4期生 songs.
# Override with SONG_TRACK_IDS (comma-separated, one per day, in date order);
# find ids with hack/itunes-search.sh.
#
# Usage:
#   ADMIN_API_TOKEN=your-token ./hack/seed-today.sh [BASE_URL]
#   ADMIN_API_TOKEN=your-token SONG_TRACK_IDS=123,456,789 ./hack/seed-today.sh [BASE_URL]
#
#   BASE_URL defaults to http://localhost:8080 (host+port only; the script adds
#   /api/v1). The server must be started with the SAME ADMIN_API_TOKEN, otherwise
#   the admin endpoints answer 403 (they are disabled when the token is unset).
set -euo pipefail

BASE_URL="${1:-http://localhost:8080}"
API="${BASE_URL%/}/api/v1"
TOKEN="${ADMIN_API_TOKEN:-}"

if [[ -z "${TOKEN}" ]]; then
  echo "ADMIN_API_TOKEN is required (must match the server's ADMIN_API_TOKEN)." >&2
  exit 1
fi
if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required (brew install jq)." >&2
  exit 1
fi

# One quote per day starting today, each facing the same way as that day's
# song (no lyrics — original lines that lead back to writing the day down).
QUOTES=(
  "同じ日は、二度と来ない。"
  "弱音は、夜に置いていける。朝には少し軽くなっている。"
  "お茶が冷めるのを待つ。そんな時間も、今日のうち。"
  "書いてみると、見えてくるものがある。"
)
# iTunes track ids (country=jp), one per QUOTES entry:
#   1676590178  さざ波は戻らない              乃木坂46 / 人は夢を二度見る (2023)
#   1537503194  夜明けまで強がらなくてもいい  乃木坂46 / 24th single (2019)
#   1584800964  猫舌カモミールティー          乃木坂46 4期生楽曲 / ごめんねFingers crossed (2021)
#   1537782852  I see...                    乃木坂46 4期生楽曲 / しあわせの保護色 (2020)
DEFAULT_TRACK_IDS="1676590178,1537503194,1584800964,1537782852"
IFS=',' read -r -a TRACK_IDS <<< "${SONG_TRACK_IDS:-${DEFAULT_TRACK_IDS}}"
for id in "${TRACK_IDS[@]}"; do
  if [[ ! "${id}" =~ ^[0-9]+$ ]]; then
    echo "SONG_TRACK_IDS must be comma-separated iTunes track ids (got '${id}')." >&2
    exit 1
  fi
done

# date helper: GNU date uses -d, BSD/macOS date uses -v.
date_offset() {
  local n="$1"
  if date -v +1d +%Y-%m-%d >/dev/null 2>&1; then
    date -v +"${n}"d +%Y-%m-%d       # macOS/BSD
  else
    date -d "+${n} days" +%Y-%m-%d   # GNU/Linux
  fi
}

post() { # path json → response body (a 4xx/5xx prints the body and fails)
  local out
  if ! out="$(curl -sS -X POST "${API}$1" \
    -H "Content-Type: application/json" \
    -H "X-Admin-Token: ${TOKEN}" \
    -d "$2" --fail-with-body)"; then
    echo "POST $1 failed: ${out}" >&2
    exit 1
  fi
  printf '%s' "${out}"
}

for i in "${!QUOTES[@]}"; do
  d="$(date_offset "${i}")"
  post "/admin/quotes" "{\"date\":\"${d}\",\"body_text\":\"${QUOTES[$i]}\"}" >/dev/null
  if [[ -n "${TRACK_IDS[$i]:-}" ]]; then
    # The server rejects a track Apple does not know or cannot preview (422).
    song="$(post "/admin/songs" "{\"date\":\"${d}\",\"itunes_track_id\":${TRACK_IDS[$i]}}")"
    echo "seeded ${d}: quote + song $(printf '%s' "${song}" | jq -r '"\(.title) / \(.artist)"')"
  else
    echo "seeded ${d}: quote only (no track id for this day)"
  fi
done

echo "done. GET ${API}/today (Bearer) will now return today's quote + song."
