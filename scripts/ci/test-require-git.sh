#!/usr/bin/env bash
# Self-test of require-git.sh with stand-in "git" executables. Proves that the
# gate compares numbers (not strings), rejects old, garbled and missing Git, and
# rejects a Git that does not accept --no-lazy-fetch. Uses no real Git.
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
gate="$here/require-git.sh"
bash=$(command -v bash)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

failures=0

# make_git <name> <banner> [reject-option]
make_git() {
  local dir="$work/$1"
  mkdir -p "$dir"
  {
    echo '#!/bin/sh'
    if [ "${3:-}" = reject-option ]; then
      echo 'case "$1" in --no-lazy-fetch) echo "unknown option: --no-lazy-fetch" >&2; exit 129;; esac'
    fi
    printf 'echo "%s"\n' "$2"
  } >"$dir/git"
  chmod +x "$dir/git"
}

# expect <exit-status 0|1> <label> <PATH-for-git>
expect() {
  local want=$1 label=$2 gitdir=$3 got=0
  # PATH holds only the stand-in directory, so a real Git can never be found;
  # the gate uses nothing but bash builtins and git.
  PATH="$gitdir" "$bash" "$gate" >"$work/out.txt" 2>&1 || got=$?
  if [ "$got" -eq "$want" ]; then
    echo "ok   $label (exit $got)"
  else
    echo "FAIL $label: want exit $want, got $got" >&2
    cat "$work/out.txt" >&2
    failures=$((failures + 1))
  fi
}

make_git v2_45_0 "git version 2.45.0"
make_git v2_52_0 "git version 2.52.0"
make_git v2_100_0 "git version 2.100.0"
make_git v3_0_0 "git version 3.0.0"
make_git v2_44_2 "git version 2.44.2"
make_git v2_9_9 "git version 2.9.9"
make_git v2_5_0 "git version 2.5.0"
make_git v1_99_0 "git version 1.99.0"
make_git apple "git version 2.39.5 (Apple Git-154)"
make_git windows "git version 2.47.1.windows.2"
make_git garbled "not a git banner"
make_git empty ""
make_git no_option "git version 2.50.0" reject-option

expect 0 "2.45.0 is the minimum" "$work/v2_45_0"
expect 0 "2.52.0" "$work/v2_52_0"
expect 0 "2.100.0 (numeric, not lexical: '2.100' < '2.45' as text)" "$work/v2_100_0"
expect 0 "3.0.0" "$work/v3_0_0"
expect 0 "2.47.1.windows.2 suffix" "$work/windows"
expect 1 "2.44.2 is too old" "$work/v2_44_2"
expect 1 "2.9.9 is too old ('2.9' > '2.45' as text)" "$work/v2_9_9"
expect 1 "2.5.0 is too old" "$work/v2_5_0"
expect 1 "1.99.0 is too old" "$work/v1_99_0"
expect 1 "Apple Git 2.39.5" "$work/apple"
expect 1 "garbled banner" "$work/garbled"
expect 1 "empty banner" "$work/empty"
expect 1 "Git without --no-lazy-fetch" "$work/no_option"
mkdir -p "$work/empty_path"
expect 1 "no git on PATH" "$work/empty_path"

if [ "$failures" -ne 0 ]; then
  echo "test-require-git: $failures failure(s)" >&2
  exit 1
fi
echo "test-require-git: all cases passed"
