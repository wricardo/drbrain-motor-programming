#!/usr/bin/env bash
# End-to-end smoke test: starts the server, drives a session to WON over
# GraphQL, checks /llms.txt, then shuts down. Usage: scripts/smoke.sh [port]
set -euo pipefail

PORT="${1:-9191}"
URL="http://127.0.0.1:${PORT}"
cd "$(dirname "$0")/.."

SESSIONS="$(mktemp -d)"
go build -o "${SESSIONS}/drbrain-motor-programming" .
"${SESSIONS}/drbrain-motor-programming" -port "${PORT}" -host 127.0.0.1 -sessions-dir "${SESSIONS}/sessions" >"${SESSIONS}/server.log" 2>&1 &
PID=$!
trap 'kill "${PID}" 2>/dev/null || true; wait "${PID}" 2>/dev/null || true; rm -rf "${SESSIONS}"' EXIT

for _ in $(seq 1 50); do
  curl -sf "${URL}/healthz" >/dev/null && break
  sleep 0.1
done

gql() { # gql '<query>'
  curl -sf -H 'Content-Type: application/json' -d "$(jq -nc --arg q "$1" '{query:$q}')" "${URL}/graphql"
}
fail() { echo "FAIL: $*" >&2; cat "${SESSIONS}/server.log" >&2; exit 1; }

echo "maps:"; gql '{ maps { id } }' | tee /dev/stderr | jq -e '.data.maps | length > 0' >/dev/null || fail "no maps"
echo

SID=$(gql 'mutation { createSession(mapID:"straight_line") { id } }' | jq -er '.data.createSession.id') || fail "createSession"
echo "session: ${SID}"

gql "mutation { setProgram(sessionID:\"${SID}\", program:{main:[MOVE_FORWARD,MOVE_FORWARD,MOVE_FORWARD,MOVE_FORWARD,MOVE_FORWARD,MOVE_FORWARD,MOVE_FORWARD,EMPTY], subs:[[EMPTY,EMPTY,EMPTY,EMPTY]]}) { id } }" \
  | jq -e '.errors == null' >/dev/null || fail "setProgram"

gql "mutation { run(sessionID:\"${SID}\", speedMs:50) { playing } }" | jq -e '.errors == null' >/dev/null || fail "run"
STATUS=""
for _ in $(seq 1 100); do
  STATUS=$(gql "{ session(id:\"${SID}\") { vm { status steps } } }" | jq -r '.data.session.vm.status')
  [ "${STATUS}" = "WON" ] && break
  sleep 0.1
done
[ "${STATUS}" = "WON" ] || fail "session did not reach WON (status=${STATUS})"
echo "session WON"

curl -sf "${URL}/llms.txt" | grep -q "Dr Brain" || fail "/llms.txt"
echo "llms.txt ok"
echo "SMOKE OK"
