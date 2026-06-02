#!/bin/sh

set -u

die() {
	printf '%s\n' "$1" >&2
	exit 1
}

if [ "$#" -gt 1 ]; then
	die "expected at most one release tag argument"
fi

if [ "$#" -eq 1 ]; then
	tag=$1
	tag_source="argument"
	if [ -z "$tag" ]; then
		die "release tag argument must not be empty; expected v2.Y.Z"
	fi
elif [ -n "${GORELEASER_CURRENT_TAG:-}" ]; then
	tag=$GORELEASER_CURRENT_TAG
	tag_source="GORELEASER_CURRENT_TAG"
else
	tag=$(git describe --tags --exact-match HEAD 2>/dev/null || true)
	tag_source="current git tag"
	if [ -z "$tag" ]; then
		exit 0
	fi
fi

if printf '%s\n' "$tag" | grep -Eq '^v2\.[0-9]+\.[0-9]+$'; then
	exit 0
fi

die "invalid release tag from $tag_source: $tag; expected v2.Y.Z, for example v2.1.0"
