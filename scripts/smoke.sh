#!/usr/bin/env bash

# Simple smoke test for fr24 CLI commands.
# Usage: scripts/smoke.sh [path-to-binary]

set -u

BIN=${1:-}
if [[ -z "${BIN}" ]]; then
  if command -v fr24 >/dev/null 2>&1; then
    BIN=$(command -v fr24)
  else
    # fallback to first built artifact
    BIN=$(ls bin/fr24-* 2>/dev/null | head -n1 || true)
  fi
fi

if [[ -z "${BIN}" || ! -x "${BIN}" ]]; then
  echo "ERROR: fr24 binary not found. Pass path or build with 'make build'." >&2
  exit 1
fi

echo "Using binary: ${BIN}"

PASS=0; FAIL=0
ok() { echo "OK  - $1"; PASS=$((PASS+1)); }
ko() { echo "FAIL- $1"; FAIL=$((FAIL+1)); }

run() {
  local name="$1"; shift
  local timeout_s=${TIMEOUT:-25}
  if /usr/bin/timeout ${timeout_s}s ${BIN} "$@" > /tmp/fr24.$$.out 2>/tmp/fr24.$$.err; then
    ok "${name}"
  else
    ko "${name} (rc=$?, stderr: $(head -c 200 /tmp/fr24.$$.err))"
  fi
}

# Help checks
for sub in "" version login dirs flightlist airportlist find livefeed playbackfeed nearest livestatus topflights flightdetails playbackflight followflight; do
  if [[ -z "${sub}" ]]; then
    run "help (root)" -h
  else
    run "help (${sub})" ${sub} -h
  fi
done

# Functional checks (best-effort)
run "login" login

run "find" find -q A359
run "airportlist" airportlist -code HKG -mode arrivals
run "flightlist" flightlist -reg B-HPJ

run "livefeed" livefeed -south 22 -north 23 -west 113 -east 115
run "playbackfeed" playbackfeed -south 22 -north 23 -west 113 -east 115 -duration 5

# Derive a valid id+timestamp for playbackflight from playbackfeed output
PB_ID=""
PB_TS=""
if [[ -s /tmp/fr24.$$.out ]]; then
  cp /tmp/fr24.$$.out /tmp/fr24.playbackfeed.out
  PB_ID=$(grep -o '"flightid":[0-9]\+' /tmp/fr24.playbackfeed.out | head -n1 | tr -dc '0-9')
  PB_TS_MS=$(grep -o '"timestamp":[0-9]\+' /tmp/fr24.playbackfeed.out | head -n1 | tr -dc '0-9')
  if [[ -n "${PB_TS_MS}" ]]; then
    PB_TS=$(( PB_TS_MS / 1000 ))
  fi
fi

# nearest → capture an ID for later commands
if /usr/bin/timeout 25s ${BIN} nearest -lat 22.3 -lon 114.2 > /tmp/fr24.nearest.out 2>/tmp/fr24.nearest.err; then
  ok "nearest"
else
  ko "nearest (rc=$?, stderr: $(head -c 200 /tmp/fr24.nearest.err))"
fi

NEAR_ID=$(grep -o '"flightid":[0-9]\+' /tmp/fr24.nearest.out | head -n1 | tr -dc '0-9')
if [[ -n "${NEAR_ID}" ]]; then
  run "livestatus" livestatus -id ${NEAR_ID}
  if [[ -n "${PB_ID}" && -n "${PB_TS}" ]]; then
    run "playbackflight" playbackflight -id ${PB_ID} -ts ${PB_TS}
  else
    echo "WARN: could not derive playbackflight inputs; skipping playbackflight"
  fi
  # followflight is streaming; keep a short timeout
  TIMEOUT=8 run "followflight" followflight -id ${NEAR_ID} -timeout 6 -once
else
  echo "WARN: could not extract flightid from nearest output; skipping live-dependent checks"
fi

# topflights → pick one id for flightdetails
if /usr/bin/timeout 25s ${BIN} topflights -limit 3 > /tmp/fr24.top.out 2>/tmp/fr24.top.err; then
  ok "topflights"
  TOP_ID=$(grep -o '"flight_id":[0-9]\+' /tmp/fr24.top.out | head -n1 | tr -dc '0-9')
  if [[ -n "${TOP_ID}" ]]; then
    run "flightdetails" flightdetails -id ${TOP_ID}
  fi
else
  ko "topflights (rc=$?, stderr: $(head -c 200 /tmp/fr24.top.err))"
fi

echo
echo "Summary: PASS=${PASS} FAIL=${FAIL}"
exit ${FAIL}
