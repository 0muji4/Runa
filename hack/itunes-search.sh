#!/usr/bin/env bash
# Find the iTunes track id of a song to register as today's song. Searches
# Apple's public (unauthenticated) iTunes Search API in the Japanese catalog and
# prints one candidate per line: track id, whether Apple offers a preview, title,
# artist. Pass the id to hack/seed-today.sh (SONG_TRACK_IDS) or straight to
# POST /admin/songs as itunes_track_id.
#
# Usage:
#   ./hack/itunes-search.sh "曲名 アーティスト名" [limit]
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <search term> [limit]" >&2
  exit 1
fi
if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required (brew install jq)." >&2
  exit 1
fi

QUERY="$1"
LIMIT="${2:-10}"

curl -fsS --get "https://itunes.apple.com/search" \
  --data-urlencode "term=${QUERY}" \
  --data-urlencode "country=jp" \
  --data-urlencode "entity=song" \
  --data-urlencode "limit=${LIMIT}" \
  | jq -r '
      .results[]
      | select(.kind == "song")
      | [ (.trackId | tostring),
          (if (.previewUrl // "") == "" then "no-preview" else "preview" end),
          .trackName,
          .artistName ]
      | @tsv'
