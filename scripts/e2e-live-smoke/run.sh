#!/bin/sh
# e2e-live-smoke.sh — real-browser E2E smoke of the live dashboard.
#
# Boots the live/demo server, drives it with headless Chromium via
# puppeteer-core, and asserts: zero console/page errors, signals initialize,
# every dashboard fragment renders, all 5 tabs activate via clicks (which
# exercises datastar expression evaluation under the dashboard CSP), and the
# NDJSON export endpoint serves a valid stream. Exit 0 = all checks pass.
#
# Prerequisites (NixOS: use a store chromium; the HyperFrames playwright
# download lacks glibc deps):
#   node >= 20 and a chromium binary, e.g.
#   CHROME="$(ls -d /nix/store/*-chromium-*/bin/chromium | head -1)"
#
# Usage:
#   CHROME=/path/to/chromium scripts/e2e-live-smoke/run.sh
#   (installs scripts/e2e-live-smoke/node_modules on first run via npm)
#
# The browser test was first run 2026-10-26 against the live/demo fixture
# (Chromium 152): 12/12 checks green, zero console errors. The pre-2026-10-26
# CSP work had been verified at string/header/test level only.
set -eu

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
CHROME="${CHROME:?set CHROME to a chromium binary path}"
PORT="${PORT:-7777}"
DASH_URL="http://localhost:${PORT}/debug/di/"

export GOEXPERIMENT=jsonv2
cd "$ROOT/scripts/e2e-live-smoke"
if [ ! -d node_modules ]; then
	npm install --no-fund --no-audit --silent
fi

DEMO_BIN="$(mktemp /tmp/do-live-demo.XXXXXX)"
trap 'kill "${DEMO_PID:-0}" 2>/dev/null || true; rm -f "$DEMO_BIN"' EXIT

cd "$ROOT"
go build -o "$DEMO_BIN" ./live/demo

"$DEMO_BIN" > /tmp/do-live-demo-e2e.log 2>&1 &
DEMO_PID=$!

# The demo lifecycle takes ~10s (services build with delays); wait it out so
# assertions see the final state (health checks, completion status).
for i in $(seq 1 40); do
	if timeout 1 sh -c "exec 3<>/dev/tcp/127.0.0.1/${PORT}" 2>/dev/null; then
		break
	fi
	sleep 1
done
sleep 12

DASH_URL="$DASH_URL" CHROME="$CHROME" node "$(dirname "$0")/smoke.mjs"
