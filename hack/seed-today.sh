#!/usr/bin/env bash
# Seed curated "today" content (daily quotes + songs) via the admin endpoints, so
# a fresh database shows a quote + song on the home screen. Dates are computed
# relative to the run date (today + the next few days), so it works on whatever
# calendar day you run it — GET /today matches an exact date.
#
# Usage:
#   ADMIN_API_TOKEN=your-token ./hack/seed-today.sh [BASE_URL]
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

# A small rotation of curated copy + tracks. Uses a public royalty-free sample
# stream so the player is actually playable in a demo; swap for real curation.
QUOTES=(
  "月あかりのはじまり。今日という夜を、そっと開く。"
  "満ちても欠けても、あなたはあなたのままで。"
  "静けさの中に、確かな光がある。"
  "眠る前のひと呼吸を、月に預けて。"
)
TITLES=("夜想曲" "薄明" "残響" "月の呼吸")
ARTISTS=("月詠" "灯" "凪" "しづく")
ARTWORK="https://upload.wikimedia.org/wikipedia/commons/e/e1/FullMoon2010.jpg"
AUDIO="https://download.samplelib.com/mp3/sample-9s.mp3"

# date helper: GNU date uses -d, BSD/macOS date uses -v.
date_offset() {
  local n="$1"
  if date -v +1d +%Y-%m-%d >/dev/null 2>&1; then
    date -v +"${n}"d +%Y-%m-%d       # macOS/BSD
  else
    date -d "+${n} days" +%Y-%m-%d   # GNU/Linux
  fi
}

post() { # path json
  curl -fsS -X POST "${API}$1" \
    -H "Content-Type: application/json" \
    -H "X-Admin-Token: ${TOKEN}" \
    -d "$2" >/dev/null
}

for i in "${!TITLES[@]}"; do
  d="$(date_offset "${i}")"
  post "/admin/quotes" "{\"date\":\"${d}\",\"body_text\":\"${QUOTES[$i]}\"}"
  post "/admin/songs" "{\"date\":\"${d}\",\"title\":\"${TITLES[$i]}\",\"artist\":\"${ARTISTS[$i]}\",\"artwork_url\":\"${ARTWORK}\",\"audio_url\":\"${AUDIO}\"}"
  echo "seeded ${d}: ${TITLES[$i]}"
done

echo "done. GET ${API}/today (Bearer) will now return today's quote + song."
