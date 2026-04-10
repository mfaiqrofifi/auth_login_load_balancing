#!/usr/bin/env sh

set -eu

TARGET_URL="${1:-http://localhost:8080/health}"
REQUESTS="${2:-10}"

echo "Testing load balancing against: ${TARGET_URL}"
echo "Requests: ${REQUESTS}"
echo

TMP_OUTPUT="$(mktemp)"
trap 'rm -f "$TMP_OUTPUT"' EXIT

i=1
while [ "$i" -le "$REQUESTS" ]; do
  BODY="$(curl -sS "$TARGET_URL")"
  INSTANCE="$(printf '%s' "$BODY" | sed -n 's/.*"instance_name":"\([^"]*\)".*/\1/p')"
  if [ -z "$INSTANCE" ]; then
    INSTANCE="unknown"
  fi
  echo "Request $i -> $INSTANCE"
  echo "$INSTANCE" >> "$TMP_OUTPUT"
  i=$((i + 1))
done

echo
echo "Summary:"
sort "$TMP_OUTPUT" | uniq -c
