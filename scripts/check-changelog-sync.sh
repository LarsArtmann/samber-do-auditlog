#!/bin/sh
# check-changelog-sync.sh — fails when CHANGELOG.md and the docs-site
# changelog.mdx disagree on their release-version lists.
#
# The docs-site changelog drifted 2 releases behind before the 2026-09-01
# manual sync; this guard makes that drift a red check instead of a surprise.
#
# Usage: scripts/check-changelog-sync.sh   (from repo root or anywhere)

set -eu

ROOT="${CHANGELOG_SYNC_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"

extract_versions() {
	grep -oE '^## \[[0-9]+\.[0-9]+\.[0-9]+\]' "$1" | sed 's/^## \[//;s/\]$//' | sort
}

MAIN_CHANGELOG="$ROOT/CHANGELOG.md"
SITE_CHANGELOG="$ROOT/website/src/content/docs/changelog.mdx"

for f in "$MAIN_CHANGELOG" "$SITE_CHANGELOG"; do
	if [ ! -f "$f" ]; then
		echo "FAIL: required file missing: $f"
		exit 1
	fi
done

main_versions="$(extract_versions "$MAIN_CHANGELOG")"
site_versions="$(extract_versions "$SITE_CHANGELOG")"

if [ "$main_versions" != "$site_versions" ]; then
	missing_in_site="$(printf '%s\n' "$main_versions" | while IFS= read -r v; do
		[ -n "$v" ] || continue
		printf '%s\n' "$site_versions" | grep -qx "$v" || echo "$v"
	done)"
	missing_in_main="$(printf '%s\n' "$site_versions" | while IFS= read -r v; do
		[ -n "$v" ] || continue
		printf '%s\n' "$main_versions" | grep -qx "$v" || echo "$v"
	done)"

	echo "FAIL: CHANGELOG.md and website changelog.mdx disagree on releases."
	echo ""
	echo "Only in CHANGELOG.md: ${missing_in_site:-<none>}"
	echo "Only in changelog.mdx: ${missing_in_main:-<none>}"
	echo ""
	echo "Fix: sync the release sections between the two files."
	exit 1
fi

echo "changelog sync OK: $(echo "$main_versions" | wc -l | tr -d ' ') releases present in both files"
