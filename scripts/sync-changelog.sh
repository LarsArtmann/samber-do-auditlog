#!/bin/sh
# sync-changelog.sh — one-command CHANGELOG.md → website changelog.mdx sync.
#
# The docs-site changelog is editorially condensed, so this script does NOT
# rewrite existing sections. It only INSERTS the verbatim CHANGELOG.md section
# for every release the site changelog is missing (the "release cut, site
# changelog forgotten" failure mode that broke the Website workflow on the
# v0.11.0 release). A maintainer may condense the inserted section later.
#
# MDX safety: outside code spans, `<` and `{` are backslash-escaped (MDX would
# otherwise parse them as JSX/expressions — e.g. the literal `<meta>` tag in
# release notes). Text inside backticks is left untouched.
#
# Companion guard: scripts/check-changelog-sync.sh (fails CI on drift).
#
# Usage: scripts/sync-changelog.sh   (from repo root or anywhere)

set -eu

ROOT="${CHANGELOG_SYNC_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"

MAIN_CHANGELOG="$ROOT/CHANGELOG.md"
SITE_CHANGELOG="$ROOT/website/src/content/docs/changelog.mdx"

for f in "$MAIN_CHANGELOG" "$SITE_CHANGELOG"; do
	if [ ! -f "$f" ]; then
		echo "FAIL: required file missing: $f"
		exit 1
	fi
done

# escape_mdx <file>: backslash-escape < and { outside backtick code spans.
# Reads stdin, writes stdout. Longest-code-span-first is unnecessary: inline
# code in the changelog never nests backticks.
escape_mdx() {
	awk '
	{
		line = $0
		out = ""
		in_code = 0
		n = length(line)
		for (i = 1; i <= n; i++) {
			ch = substr(line, i, 1)
			if (ch == "`") {
				in_code = !in_code
				out = out ch
			} else if (!in_code && (ch == "<" || ch == "{")) {
				out = out "\\" ch
			} else {
				out = out ch
			}
		}
		print out
	}'
}

inserted=0

for version in $(grep -oE '^## \[[0-9]+\.[0-9]+\.[0-9]+\]' "$MAIN_CHANGELOG" | sed 's/^## \[//;s/\]$//' | sort -rV); do
	if grep -qF "## [$version]" "$SITE_CHANGELOG"; then
		continue
	fi

	# Extract the section body from CHANGELOG.md: from the `## [version]`
	# heading up to (excluding) the next `## [` heading.
	body="$(sed -n "/^## \[$version\]/,/^## \[/p" "$MAIN_CHANGELOG" | sed '$d')"

	# Match the site changelog heading style: `## [x.y.z] — date` (em dash).
	body="$(printf '%s\n' "$body" | sed "0,/^## \[$version\] - /s//## [$version] — /")"
	body="$(printf '%s\n' "$body" | escape_mdx)"

	# Insertion point: before the heading of the next-lower release present
	# in the site changelog (sections are ordered newest-first), or append
	# at the end when no older release exists yet.
	insert_before="$(grep -nE '^## \[[0-9]+\.[0-9]+\.[0-9]+\]' "$SITE_CHANGELOG" | cut -d: -f1- | while IFS=: read -r linenum heading; do
		other="$(printf '%s' "$heading" | sed 's/^## \[//;s/\].*//')"
		if [ "$(printf '%s\n%s\n' "$version" "$other" | sort -V | tail -n 1)" = "$version" ]; then
			echo "$linenum"
			break
		fi
	done)"

	if [ -n "$insert_before" ]; then
		tmp="$(mktemp)"
		head -n "$((insert_before - 1))" "$SITE_CHANGELOG" >"$tmp"
		printf '%s\n\n' "$body" >>"$tmp"
		tail -n "+$insert_before" "$SITE_CHANGELOG" >>"$tmp"
		mv "$tmp" "$SITE_CHANGELOG"
	else
		printf '\n%s\n' "$body" >>"$SITE_CHANGELOG"
	fi

	echo "inserted [$version] section into changelog.mdx"
	inserted=$((inserted + 1))
done

if [ "$inserted" -eq 0 ]; then
	echo "changelog.mdx already covers every CHANGELOG.md release (nothing to do)"
fi

# Delegate the final verdict to the guard so a successful sync is proven, not
# assumed.
exec "$ROOT/scripts/check-changelog-sync.sh"
