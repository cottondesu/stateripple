#!/usr/bin/env bash
# analyze-acceptance.sh <stateripple-binary>
#
# Runs the built binary's optional Git+Go adapter (`analyze`) against throwaway
# repositories. Needs Git 2.45+ and a Go toolchain on PATH. Everything happens in
# a temporary directory; the checkout under test is never touched.
#
# Checked: complete analysis (exit 0) and deterministic bytes; HEAD, index,
# object store and work tree unchanged by the analysis; text output; No-Clobber
# and output-path safety of --out; the Git version gate with a Git that is too
# old (exit 1, Go never started); a blob:none partial clone with missing base
# blobs (exit 2, incomplete, nothing fetched, nothing changed); invalid use
# (exit 1). It does not claim OS-level network isolation by itself; the CI job
# runs it inside an empty network namespace.
set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "usage: analyze-acceptance.sh <stateripple-binary>" >&2
  exit 2
fi
case $1 in /*) bin=$1 ;; *) bin=$PWD/$1 ;; esac
[ -x "$bin" ] || { echo "analyze-acceptance: $bin is not executable" >&2; exit 2; }

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
failures=0
checks=0

ok() { checks=$((checks + 1)); echo "ok   $*"; }
bad() { checks=$((checks + 1)); failures=$((failures + 1)); echo "FAIL $*" >&2; }
check() { # check <label> <command...>
  local label=$1
  shift
  if "$@"; then ok "$label"; else bad "$label"; fi
}

sha256_stdin() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum | cut -d' ' -f1; else shasum -a 256 | cut -d' ' -f1; fi
}
# Fingerprint of every file below a directory (names and contents).
tree_fingerprint() {
  (cd "$1" && find . -type f -print | LC_ALL=C sort | while IFS= read -r f; do
    printf '%s ' "$f"
    sha256_stdin <"$f"
  done) | sha256_stdin
}

# Fixture Git: no user or system configuration, fixed identity, local protocol
# only, lazy fetching forbidden, no prompts.
fxgit() {
  GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1 \
    GIT_AUTHOR_NAME=Fixture GIT_AUTHOR_EMAIL=fixture@example.invalid \
    GIT_COMMITTER_NAME=Fixture GIT_COMMITTER_EMAIL=fixture@example.invalid \
    GIT_AUTHOR_DATE='2026-01-01T00:00:00Z' GIT_COMMITTER_DATE='2026-01-01T00:00:00Z' \
    GIT_ALLOW_PROTOCOL=file GIT_NO_LAZY_FETCH=1 GIT_TERMINAL_PROMPT=0 \
    git "$@"
}
fxwrite() { # fxwrite <repo> <path> <content>
  mkdir -p "$(dirname "$1/$2")"
  printf '%s' "$3" >"$1/$2"
}
fxcommit() {
  fxgit -C "$1" add --all
  fxgit -C "$1" commit -q -m fixture
}
analyze() { # analyze <repo> <args...>; stdout/stderr go to $work/out and $work/err
  local repo=$1
  shift
  local status=0
  (cd "$repo" && "$bin" analyze "$@") >"$work/out" 2>"$work/err" || status=$?
  return "$status"
}
expect_exit() { # expect_exit <want> <label> <repo> <analyze args...>
  local want=$1 label=$2 repo=$3 got=0
  shift 3
  analyze "$repo" "$@" || got=$?
  if [ "$got" -eq "$want" ]; then ok "$label (exit $got)"; else
    bad "$label: want exit $want, got $got"
    sed 's/^/     stderr: /' "$work/err" | head -5 >&2
  fi
}

echo "analyze-acceptance: binary $bin"
git --version
go version

# ---- 1. complete analysis of a two-commit repository -------------------------
repo=$work/complete
mkdir -p "$repo"
fxgit -C "$repo" init -q
fxwrite "$repo" go.mod $'module example.invalid/fx\n\ngo 1.23\n'
fxwrite "$repo" a/a.go $'package a\n\nfunc A() int { return 1 }\n'
fxwrite "$repo" a/a_test.go $'package a\n\nimport "testing"\n\nfunc TestA(t *testing.T) {}\n'
fxwrite "$repo" b/b.go $'package b\n\nimport "example.invalid/fx/a"\n\nfunc B() int { return a.A() }\n'
fxcommit "$repo"
fxwrite "$repo" a/a.go $'package a\n\nfunc A() int { return 2 }\n'
fxcommit "$repo"

head_before=$(fxgit -C "$repo" rev-parse HEAD)
objects_before=$(tree_fingerprint "$repo/.git/objects")
index_before=$(sha256_stdin <"$repo/.git/index")

expect_exit 0 "complete analysis, JSON" "$repo" --base HEAD~1 --format json
cp "$work/out" "$work/complete.json"
check "status is complete" grep -q '"status":"complete"' "$work/complete.json"
check "changed package seeded at risk 3" grep -q '"id":"pkg:example.invalid/fx/a"' "$work/complete.json"
check "dependent package reached" grep -q '"id":"pkg:example.invalid/fx/b"' "$work/complete.json"
check "stdout is one canonical JSON document" test "$(wc -l <"$work/complete.json" | tr -d ' ')" -le 1
expect_exit 0 "second run" "$repo" --base HEAD~1 --format json
check "same repository, same bytes" cmp -s "$work/out" "$work/complete.json"
expect_exit 0 "text output" "$repo" --base HEAD~1 --format text
check "text report says complete" grep -q '^Status: complete' "$work/out"

check "HEAD unchanged by analysis" test "$(fxgit -C "$repo" rev-parse HEAD)" = "$head_before"
check "object store unchanged by analysis" test "$(tree_fingerprint "$repo/.git/objects")" = "$objects_before"
check "index unchanged by analysis" test "$(sha256_stdin <"$repo/.git/index")" = "$index_before"
check "work tree clean after analysis" test -z "$(fxgit -C "$repo" status --porcelain=v1 --untracked-files=all)"

# ---- 2. --out: new file only, private, never overwritten -----------------------
mkdir -p "$work/o"
out_file=$work/o/report.json
expect_exit 0 "--out creates a new file" "$repo" --base HEAD~1 --format json --out "$out_file"
check "--out bytes equal stdout bytes" cmp -s "$out_file" "$work/complete.json"
check "--out file is private (0600)" test "$(ls -l "$out_file" | cut -c1-10)" = "-rw-------"
out_sum=$(sha256_stdin <"$out_file")
expect_exit 1 "--out refuses an existing file (No-Clobber)" "$repo" --base HEAD~1 --format json --out "$out_file"
check "existing --out file unchanged" test "$(sha256_stdin <"$out_file")" = "$out_sum"
check "refused run wrote nothing to stdout" test ! -s "$work/out"
entries_before=$(ls -A "$work/o" | LC_ALL=C sort | tr '\n' ' ')
expect_exit 1 "--out refuses an existing directory" "$repo" --base HEAD~1 --out "$work/o"
expect_exit 1 "--out refuses a directory with trailing slash" "$repo" --base HEAD~1 --out "$work/o/"
expect_exit 1 "--out refuses a nonexistent path with trailing slash" "$repo" --base HEAD~1 --out "$work/o/new/"
expect_exit 1 "--out refuses '.'" "$repo" --base HEAD~1 --out .
ln -s "$out_file" "$work/o/link.json"
expect_exit 1 "--out refuses a symlink" "$repo" --base HEAD~1 --out "$work/o/link.json"
rm "$work/o/link.json"
check "no stray entries after refused --out" test "$(ls -A "$work/o" | LC_ALL=C sort | tr '\n' ' ')" = "$entries_before"
check "no output file appeared in the repository" test -z "$(fxgit -C "$repo" status --porcelain=v1 --untracked-files=all)"

# ---- 3. invalid use is exit 1 ---------------------------------------------------
mkdir -p "$work/notrepo"
expect_exit 1 "outside a Git repository" "$work/notrepo" --format json
check "no report on error" test ! -s "$work/out"
fxwrite "$repo" a/a.go $'package a\n\nfunc A() int { return 3 }\n'
expect_exit 1 "uncommitted tracked change" "$repo" --base HEAD~1 --format json
check "HEAD unchanged after refused analysis" test "$(fxgit -C "$repo" rev-parse HEAD)" = "$head_before"
fxgit -C "$repo" checkout -q -- a/a.go
expect_exit 1 "unknown base revision" "$repo" --base no-such-revision --format json
expect_exit 1 "unknown format" "$repo" --base HEAD~1 --format yaml

# ---- 4. Git version gate: a Git older than 2.45 is rejected before Go runs -----
shim=$work/shim
mkdir -p "$shim"
printf '#!/bin/sh\necho "git version 2.44.2"\n' >"$shim/git"
printf '#!/bin/sh\necho started >>"%s/go-started"\nexit 1\n' "$work" >"$shim/go"
chmod +x "$shim/git" "$shim/go"
got=0
(cd "$repo" && PATH="$shim:$PATH" "$bin" analyze --base HEAD~1 --format json) >"$work/out" 2>"$work/err" || got=$?
if [ "$got" -eq 1 ]; then ok "Git 2.44.2 rejected (exit 1)"; else bad "Git 2.44.2: want exit 1, got $got"; fi
check "diagnostic names the 2.45.0 requirement" grep -q 'requires Git 2.45.0' "$work/err"
check "no report from an unsupported Git" test ! -s "$work/out"
check "Go was never started for an unsupported Git" test ! -e "$work/go-started"

# ---- 5. partial clone with missing base blobs: incomplete (exit 2), no fetch ---
origin=$work/origin
mkdir -p "$origin"
fxgit -C "$origin" init -q
fxwrite "$origin" go.mod $'module example.invalid/fx\n\ngo 1.23\n'
fxwrite "$origin" a/a.go $'package a\n\nimport _ "embed"\n\n//go:embed old.txt\nvar Data string\n'
fxwrite "$origin" a/old.txt $'old embed\n'
fxwrite "$origin" b/b.go $'package b\n\nimport "example.invalid/fx/a"\n\nfunc B() string { return a.Data }\n'
fxcommit "$origin"
fxwrite "$origin" a/a.go $'package a\n\nvar Data = "now plain"\n'
fxwrite "$origin" b/b.go $'package b\n\nimport "example.invalid/fx/a"\n\nfunc B() string { return a.Data + "!" }\n'
fxcommit "$origin"
fxgit -C "$origin" config uploadpack.allowFilter true

partial=$work/partial
fxgit -C "$origin" clone -q --no-checkout --filter=blob:none "file://$origin" "$partial"
# The HEAD blobs come from origin as one explicit local pack; the checkout then
# runs with lazy fetching forbidden. Only the two base blobs stay absent.
fxgit -C "$partial" ls-tree -r --full-tree HEAD | while read -r _ type oid _; do
  [ "$type" = blob ] && echo "$oid"
done >"$work/head-blobs"
fxgit -C "$origin" pack-objects -q --stdout <"$work/head-blobs" | fxgit -C "$partial" unpack-objects -q
fxgit -C "$partial" checkout -q HEAD -- .
check "fixture: promisor remote" test "$(fxgit -C "$partial" config --get remote.origin.promisor)" = true
check "fixture: clean checkout" test -z "$(fxgit -C "$partial" status --porcelain=v1 --untracked-files=all)"
fxgit -C "$partial" --no-lazy-fetch rev-list --objects --missing=print --all | LC_ALL=C sort >"$work/missing-before"
check "fixture: exactly the two base blobs are absent" test "$(grep -c '^?' "$work/missing-before")" -eq 2

# The promisor remote's upload-pack is a canary: any implicit fetch records itself.
printf '#!/bin/sh\necho fetch >>"%s/fetch.log"\nexit 86\n' "$work" >"$work/upload-pack"
chmod +x "$work/upload-pack"
fxgit -C "$partial" config remote.origin.uploadpack "$work/upload-pack"
partial_head=$(fxgit -C "$partial" rev-parse HEAD)
partial_objects=$(tree_fingerprint "$partial/.git/objects")

expect_exit 2 "missing base blobs are incomplete" "$partial" --base HEAD~1 --format json
check "report status is incomplete" grep -q '"status":"incomplete"' "$work/out"
check "warning names the missing base source a/a.go" grep -q 'read base Go source \\"a/a.go\\"' "$work/out"
check "warning names the missing base source b/b.go" grep -q 'read base Go source \\"b/b.go\\"' "$work/out"
check "no implicit fetch (canary silent)" test ! -e "$work/fetch.log"
check "object store unchanged" test "$(tree_fingerprint "$partial/.git/objects")" = "$partial_objects"
fxgit -C "$partial" --no-lazy-fetch rev-list --objects --missing=print --all | LC_ALL=C sort >"$work/missing-after"
check "missing-object list unchanged" cmp -s "$work/missing-before" "$work/missing-after"
check "HEAD unchanged" test "$(fxgit -C "$partial" rev-parse HEAD)" = "$partial_head"
check "work tree clean" test -z "$(fxgit -C "$partial" status --porcelain=v1 --untracked-files=all)"

# Control: the canary is live. The same read with lazy fetching allowed asks the
# promisor remote, so a silent canary above is evidence and not an accident.
missing_oid=$(sed -n 's/^?//p' "$work/missing-before" | head -n 1)
control=0
(cd "$partial" && GIT_NO_LAZY_FETCH=0 GIT_TERMINAL_PROMPT=0 GIT_ALLOW_PROTOCOL=file git cat-file blob "$missing_oid") >/dev/null 2>&1 || control=$?
check "control: an allowed lazy fetch fails" test "$control" -ne 0
check "control: an allowed lazy fetch reaches the canary" test -s "$work/fetch.log"
fxgit -C "$partial" --no-lazy-fetch rev-list --objects --missing=print --all | LC_ALL=C sort >"$work/missing-control"
check "control: the canary transferred nothing" cmp -s "$work/missing-before" "$work/missing-control"

echo "analyze-acceptance: $checks checks, $failures failure(s)"
[ "$failures" -eq 0 ]
