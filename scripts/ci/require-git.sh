#!/usr/bin/env bash
# Fails unless the Git on PATH is 2.45 or newer and understands --no-lazy-fetch.
# StateRipple's analyze adapter needs both; there is no fallback and no skip.
# Versions are compared as numbers, never as strings ("2.9" is older than "2.45").
set -euo pipefail

min_major=2
min_minor=45

banner=$(git --version) || {
  echo "require-git: git is not available on PATH" >&2
  exit 1
}
echo "$banner"

# "git version 2.52.0", "git version 2.39.5 (Apple Git-154)", "git version 2.43.0.windows.1"
if [[ ! $banner =~ ^git\ version\ ([0-9]+)\.([0-9]+) ]]; then
  echo "require-git: unrecognized version banner: $banner" >&2
  exit 1
fi
major=$((10#${BASH_REMATCH[1]}))
minor=$((10#${BASH_REMATCH[2]}))

if ((major < min_major || (major == min_major && minor < min_minor))); then
  echo "require-git: Git $min_major.$min_minor or later is required, found $major.$minor" >&2
  exit 1
fi

git --no-lazy-fetch version >/dev/null || {
  echo "require-git: this Git does not accept the global --no-lazy-fetch option" >&2
  exit 1
}
echo "require-git: OK (Git $major.$minor >= $min_major.$min_minor, --no-lazy-fetch accepted)"
