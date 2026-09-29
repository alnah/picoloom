#!/bin/sh

# Extract release notes for a tag from CHANGELOG.md, rewriting relative
# Markdown links to absolute URLs pinned to that tag.

set -u

die() {
	printf '%s\n' "$1" >&2
	exit 1
}

script_dir=$(cd -- "$(dirname -- "$0")" && pwd)
changelog="$script_dir/../CHANGELOG.md"
repo="alnah/picoloom"

if [ "$#" -gt 1 ]; then
	die "expected at most one release tag argument"
fi

if [ "$#" -eq 1 ]; then
	tag=$1
	tag_source="argument"
elif [ -n "${GORELEASER_CURRENT_TAG:-}" ]; then
	tag=$GORELEASER_CURRENT_TAG
	tag_source="GORELEASER_CURRENT_TAG"
else
	tag=$(git describe --tags --exact-match HEAD 2>/dev/null || true)
	tag_source="current git tag"
	if [ -z "$tag" ]; then
		die "no release tag given and HEAD is not tagged; expected v2.Y.Z"
	fi
fi

case "$tag" in
v[0-9]*.[0-9]*.[0-9]*) ;;
*) die "invalid release tag from $tag_source: $tag; expected v2.Y.Z, for example v2.2.0" ;;
esac

version=${tag#v}

[ -f "$changelog" ] || die "changelog not found: $changelog"

notes=$(
	awk -v version="$version" '
		BEGIN { heading = "^## \\[" version "\\]([[:space:]]+-[[:space:]]+[0-9]{4}-[0-9]{2}-[0-9]{2})?[[:space:]]*$" }
		$0 ~ heading { found = 1; next }
		found && /^## \[/ { exit }
		found { print }
	' "$changelog"
)

case "$notes" in
*[![:space:]]*) ;;
*) die "no changelog section found for $tag" ;;
esac

printf '%s\n' "$notes" | awk -v tag="$tag" -v repo="$repo" '
function rewrite(target,   fragment, path, route, head, hash) {
	if (target ~ /^[a-zA-Z][a-zA-Z0-9+.-]*:/ || target ~ /^#/ || target ~ /^\/\//) {
		return target
	}
	fragment = ""
	path = target
	hash = index(path, "#")
	if (hash > 0) {
		fragment = substr(path, hash)
		path = substr(path, 1, hash - 1)
	}
	route = "blob"
	if (path ~ /\/$/) {
		route = "tree"
	}
	return "https://github.com/" repo "/" route "/" tag "/" path fragment
}
{
	out = ""
	line = $0
	while ((start = index(line, "](")) > 0) {
		head = substr(line, 1, start + 1)
		rest = substr(line, start + 2)
		end = index(rest, ")")
		if (end == 0) {
			break
		}
		out = out head rewrite(substr(rest, 1, end - 1))
		line = substr(rest, end)
	}
	print out line
}'
