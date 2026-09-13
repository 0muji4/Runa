#!/usr/bin/env bash
# Seed curated "today" content (daily quotes + songs) via the admin endpoints, so
# a fresh database shows a quote + song on the home screen. Dates are computed
# relative to the run date (today + the next few days), so it works on whatever
# calendar day you run it — GET /today matches an exact date.
#
# Songs are tracks from Apple's catalog, registered by iTunes track id; the
# server fetches title/artist/artwork/preview from the iTunes Search API. Find
# ids with hack/itunes-search.sh and pass them comma-separated in
# SONG_TRACK_IDS (one per day, in date order). Without it only quotes are seeded.
#
# Usage:
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

# A small rotation of curated copy; one quote per day starting today.
QUOTES=(
  "月あかりのはじまり。今日という夜を、そっと開く。"
  "満ちても欠けても、あなたはあなたのままで。"
  "静けさの中に、確かな光がある。"
  "眠る前のひと呼吸を、月に預けて。"
)
IFS=',' read -r -a TRACK_IDS <<< "${SONG_TRACK_IDS:-}"
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

if [[ -z "${SONG_TRACK_IDS:-}" ]]; then
  echo "no songs seeded: set SONG_TRACK_IDS (find ids with ./hack/itunes-search.sh \"曲名 アーティスト\")."
fi
echo "done. GET ${API}/today (Bearer) will now return today's quote + song."
