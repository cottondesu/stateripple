#!/usr/bin/env bash
# binary-acceptance.sh <stateripple-binary> <repository-root> [--expect-no-network]
#
# Black-box checks of the built binary's generic harness (simulate, help,
# version) with no Git and no Go available to it: every invocation uses
# `env -i` and a PATH that is an empty directory. The script itself still uses
# ordinary shell utilities. Checked: the three golden reports byte for byte and
# by pinned SHA-256 (report and normalized input), canonical JSON independent of
# input order and spacing, repeatable bytes, the exit-code contract (0 complete,
# 1 error, 2 incomplete), analyze failing closed without Git, and the --out
# No-Clobber and path-safety rules.
#
# --expect-no-network additionally asserts that this process has no usable
# network (only a loopback interface, and an outbound connection fails). It is
# for runs inside an empty network namespace or container; the script cannot
# create that isolation itself and does not claim to. Without the flag no
# statement about the network is made.
set -euo pipefail

if [ "$#" -lt 2 ] || [ "$#" -gt 3 ]; then
  echo "usage: binary-acceptance.sh <binary> <repository-root> [--expect-no-network]" >&2
  exit 2
fi
case $1 in /*) bin=$1 ;; *) bin=$PWD/$1 ;; esac
repo=$(cd "$2" && pwd)
expect_no_network=false
if [ "${3:-}" = "--expect-no-network" ]; then expect_no_network=true; elif [ -n "${3:-}" ]; then
  echo "binary-acceptance: unknown option $3" >&2
  exit 2
fi
[ -x "$bin" ] || { echo "binary-acceptance: $bin is not executable" >&2; exit 2; }
[ -d "$repo/testdata" ] || { echo "binary-acceptance: $repo has no testdata directory" >&2; exit 2; }

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
empty=$work/empty-path
mkdir -p "$empty" "$work/run"
failures=0
checks=0

ok() { checks=$((checks + 1)); echo "ok   $*"; }
bad() { checks=$((checks + 1)); failures=$((failures + 1)); echo "FAIL $*" >&2; }
check() {
  local label=$1
  shift
  if "$@"; then ok "$label"; else bad "$label"; fi
}
sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | cut -d' ' -f1; else shasum -a 256 "$1" | cut -d' ' -f1; fi
}

# The binary runs with nothing but an empty PATH: no Git, no Go, no inherited
# configuration. stdout and stderr go to $work/out and $work/err.
sr() {
  env -i PATH="$empty" HOME="$empty" "$bin" "$@" >"$work/out" 2>"$work/err"
}
expect_exit() { # expect_exit <want> <label> <args...>
  local want=$1 label=$2 got=0
  shift 2
  (cd "$work/run" && sr "$@") || got=$?
  if [ "$got" -eq "$want" ]; then ok "$label (exit $got)"; else
    bad "$label: want exit $want, got $got"
    sed 's/^/     stderr: /' "$work/err" | head -5 >&2
  fi
}

echo "binary-acceptance: binary $bin"
echo "binary-acceptance: sha256 $(sha256_file "$bin")"

# ---- environment controls -------------------------------------------------------
check "control: git is not reachable from the binary's environment" \
  sh -c '! env -i PATH="$1" git --version >/dev/null 2>&1' _ "$empty"
check "control: go is not reachable from the binary's environment" \
  sh -c '! env -i PATH="$1" go version >/dev/null 2>&1' _ "$empty"

if $expect_no_network; then
  # /proc/net follows this process's network namespace (/sys/class/net does not:
  # sysfs shows the namespace it was mounted in). Fallback tunnel devices such as
  # tunl0 or sit0 may exist in any namespace, so the interface list is printed
  # for information; what is asserted is that no route exists and that
  # connections fail.
  if [ -r /proc/net/dev ] && [ -r /proc/net/route ]; then
    echo "network control: interfaces: $(sed -n '3,$p' /proc/net/dev | cut -d: -f1 | tr -d ' ' | tr '\n' ' ')"
    check "network control: the IPv4 routing table is empty" test "$(sed -n '2,$p' /proc/net/route | wc -l | tr -d ' ')" -eq 0
  else
    bad "network control: /proc/net is unavailable, cannot inspect the namespace"
  fi
  check "network control: an outbound TCP connection fails" \
    bash -c '! (exec 3<>/dev/tcp/192.0.2.1/80) 2>/dev/null'
  check "network control: a loopback connection fails" \
    bash -c '! (exec 3<>/dev/tcp/127.0.0.1/9) 2>/dev/null'
fi

# ---- help and version ------------------------------------------------------------
expect_exit 0 "version" version
check "version text" grep -qx 'stateripple v0.1.0' "$work/out"
for form in help --help -h "simulate --help" "analyze --help"; do
  # shellcheck disable=SC2086
  expect_exit 0 "$form" $form
  check "$form prints usage on stdout" grep -q '^Usage:' "$work/out"
  check "$form prints nothing on stderr" test ! -s "$work/err"
done

# ---- goldens: bytes and pinned hashes -------------------------------------------
check_golden() { # <input> <golden> <input-sha256> <report-sha256>
  expect_exit 0 "simulate $1" simulate --input "$repo/testdata/$1" --format json
  check "$1: report equals testdata/$2 byte for byte" cmp -s "$work/out" "$repo/testdata/$2"
  check "$1: report SHA-256 is pinned" test "$(sha256_file "$work/out")" = "$4"
  check "$1: input_sha256 is pinned" grep -qF "\"input_sha256\":\"$3\"" "$work/out"
  check "$1: pinned golden file itself is unchanged" test "$(sha256_file "$repo/testdata/$2")" = "$4"
  cp "$work/out" "$work/first"
  for _ in 1 2 3 4 5; do
    (cd "$work/run" && sr simulate --input "$repo/testdata/$1" --format json)
    cmp -s "$work/out" "$work/first" || { bad "$1: repeated run differs"; return; }
  done
  ok "$1: five repeated runs are byte-identical"
}
check_golden example.json example.report.json \
  e0c96ca23a1e63430c810825ffd4161ecb4a26c4835398cad2e0b66b072acde8 \
  809eb4c476c5a45e7d5f764b443fd269aa5cdef0e9865d421d43aff554edf36a
check_golden cli-empty.json cli-empty.golden.json \
  a246bcb6c6b2cf0244b5c9d2ba58e6909a89c5bf6527ffb7fcd8626e642e6818 \
  c215f338ecb25787635a91d230a58eb3e56fc082541b2f3179bc5518eae1bda2
check_golden generic-supply-chain.json generic-supply-chain.report.json \
  c34ed81c120cb7e518ee6c407ef1f94c0a77f7cc53819cc76157ffc8db3093de \
  ebbb8c55a296b17e3b7e663f6730b58a7f11070b3f313eaa5993bd27b16a854a

expect_exit 0 "text format" simulate --input "$repo/testdata/generic-supply-chain.json" --format text
check "text report heading" grep -qx 'StateRipple v0.1.0' "$work/out"
check "generic text report has no Git or test wording" test -z "$(grep -iE 'git|test target|changed files' "$work/out" || true)"

# ---- canonical JSON: order and spacing of the input do not matter ----------------
cat >"$work/order-a.json" <<'EOF'
{"schema_version":1,"ruleset":"risk-propagation-v1","max_generations":8,
 "nodes":[{"id":"mine","kind":"supplier"},{"id":"smelter","kind":"facility"},{"id":"シップ","kind":"carrier"}],
 "edges":[{"from":"mine","to":"smelter","kind":"ships_to","attenuation":0},{"from":"smelter","to":"シップ","kind":"loads","attenuation":1}],
 "seeds":[{"id":"mine","risk":3,"reason":"flood"}]}
EOF
cat >"$work/order-b.json" <<'EOF'
{
  "seeds": [ {"reason": "flood", "risk": 3, "id": "mine"} ],
  "edges": [
    {"attenuation": 1, "kind": "loads", "to": "シップ", "from": "smelter"},
    {"attenuation": 0, "kind": "ships_to", "to": "smelter", "from": "mine"}
  ],
  "nodes": [ {"kind": "carrier", "id": "シップ"}, {"kind": "facility", "id": "smelter"}, {"kind": "supplier", "id": "mine"} ],
  "max_generations": 8, "ruleset": "risk-propagation-v1", "schema_version": 1
}
EOF
expect_exit 0 "order A" simulate --input "$work/order-a.json" --format json
cp "$work/out" "$work/order-a.out"
expect_exit 0 "order B (reordered, respaced, reordered keys)" simulate --input "$work/order-b.json" --format json
check "different input order gives identical report bytes" cmp -s "$work/out" "$work/order-a.out"
check "report is one line of canonical JSON" test "$(wc -l <"$work/out" | tr -d ' ')" -le 1

# ---- exit-code contract ----------------------------------------------------------
chain() { # chain <max_generations>: seed a -> b with attenuation 0
  printf '{"schema_version":1,"ruleset":"risk-propagation-v1","max_generations":%s,"nodes":[{"id":"a","kind":"x"},{"id":"b","kind":"x"}],"edges":[{"from":"a","to":"b","kind":"k","attenuation":0}],"seeds":[{"id":"a","risk":3,"reason":"r"}]}' "$1"
}
chain 2 >"$work/limit-2.json"
chain 1 >"$work/limit-1.json"
expect_exit 0 "complete result" simulate --input "$work/limit-2.json" --format json
check "status complete" grep -q '"status":"complete"' "$work/out"
expect_exit 2 "generation limit gives an incomplete result" simulate --input "$work/limit-1.json" --format json
check "status incomplete, stop reason generation_limit" grep -q '"stop_reason":"generation_limit"' "$work/out"
check "incomplete is never reported complete" test -z "$(grep '"status":"complete"' "$work/out" || true)"
expect_exit 2 "incomplete result in text format" simulate --input "$work/limit-1.json" --format text
check "text report says incomplete" grep -q '^Status: incomplete (generation_limit)' "$work/out"

printf '{not json' >"$work/invalid.json"
printf '{"schema_version":1,"ruleset":"risk-propagation-v1","nodes":[{"id":"a","kind":"x"},{"id":"b","kind":"x"}],"edges":[{"from":"a","to":"b","kind":"k","attenuation":4}],"seeds":[]}' >"$work/attenuation-4.json"
printf '{"schema_version":1,"ruleset":"risk-propagation-v1","nodes":[{"id":"a","kind":"x"},{"id":"b","kind":"x"}],"edges":[{"from":"a","to":"b","kind":"k"}],"seeds":[]}' >"$work/attenuation-missing.json"
printf '{"schema_version":1,"ruleset":"other","nodes":[],"edges":[],"seeds":[]}' >"$work/ruleset.json"
for tc in \
  "invalid JSON|simulate --input $work/invalid.json" \
  "attenuation 4|simulate --input $work/attenuation-4.json" \
  "omitted attenuation|simulate --input $work/attenuation-missing.json" \
  "unknown ruleset|simulate --input $work/ruleset.json" \
  "missing input file|simulate --input $work/does-not-exist.json" \
  "--input omitted|simulate" \
  "unknown format|simulate --input $work/limit-2.json --format yaml" \
  "unknown flag|simulate --input $work/limit-2.json --bogus" \
  "unknown command|frobnicate" \
  "no command|"; do
  label=${tc%%|*}
  args=${tc#*|}
  # shellcheck disable=SC2086
  expect_exit 1 "error: $label" $args
  check "error: $label writes no report" test ! -s "$work/out"
  check "error: $label explains itself on one stderr line family" test -s "$work/err"
done

# ---- analyze fails closed without Git --------------------------------------------
mkdir -p "$work/norepo"
got=0
(cd "$work/norepo" && sr analyze --format json) || got=$?
if [ "$got" -eq 1 ]; then ok "analyze without Git (exit 1)"; else bad "analyze without Git: want exit 1, got $got"; fi
check "analyze names the Git 2.45.0 requirement" grep -q 'requires Git 2.45.0' "$work/err"
check "analyze without Git writes no report" test ! -s "$work/out"

# ---- --out: No-Clobber and path safety ---------------------------------------------
outdir=$work/out-dir
mkdir -p "$outdir"
golden=$repo/testdata/example.report.json
input=$repo/testdata/example.json
expect_exit 0 "--out creates a new file" simulate --input "$input" --format json --out "$outdir/r.json"
check "--out bytes equal the golden" cmp -s "$outdir/r.json" "$golden"
check "--out file is private (0600)" test "$(ls -l "$outdir/r.json" | cut -c1-10)" = "-rw-------"
check "--out leaves stdout empty" test ! -s "$work/out"
before=$(sha256_file "$outdir/r.json")
expect_exit 1 "--out refuses an existing file" simulate --input "$input" --format json --out "$outdir/r.json"
check "existing file unchanged" test "$(sha256_file "$outdir/r.json")" = "$before"
ln "$outdir/r.json" "$outdir/hard.json"
expect_exit 1 "--out refuses a hard link" simulate --input "$input" --format json --out "$outdir/hard.json"
ln -s "$outdir/r.json" "$outdir/sym.json"
expect_exit 1 "--out refuses a symlink" simulate --input "$input" --format json --out "$outdir/sym.json"
ln -s "$outdir/missing-target.json" "$outdir/dangling.json"
expect_exit 1 "--out refuses a dangling symlink" simulate --input "$input" --format json --out "$outdir/dangling.json"
check "dangling symlink target was not created" test ! -e "$outdir/missing-target.json"
expect_exit 1 "--out refuses an existing directory" simulate --input "$input" --format json --out "$outdir"
expect_exit 1 "--out refuses a directory with trailing slash" simulate --input "$input" --format json --out "$outdir/"
expect_exit 1 "--out refuses a new path with trailing slash" simulate --input "$input" --format json --out "$outdir/new/"
expect_exit 1 "--out refuses a missing parent directory" simulate --input "$input" --format json --out "$outdir/missing/r.json"
cp "$input" "$work/alias-input.json"
alias_sum=$(sha256_file "$work/alias-input.json")
expect_exit 1 "--out refuses to overwrite the input" simulate --input "$work/alias-input.json" --format json --out "$work/alias-input.json"
check "input intact after refused alias" test "$(sha256_file "$work/alias-input.json")" = "$alias_sum"
ln "$work/alias-input.json" "$work/alias-hard.json"
expect_exit 1 "--out refuses a hard link of the input" simulate --input "$work/alias-input.json" --format json --out "$work/alias-hard.json"
check "input intact after refused hard-link alias" test "$(sha256_file "$work/alias-input.json")" = "$alias_sum"
check "refusals left only the expected entries" test "$(ls -A "$outdir" | LC_ALL=C sort | tr '\n' ' ')" = "dangling.json hard.json r.json sym.json "
expect_exit 0 "--out in text format" simulate --input "$input" --format text --out "$outdir/r.txt"
check "text --out file begins with the heading" test "$(head -n 1 "$outdir/r.txt")" = "StateRipple v0.1.0"

# Twelve processes race to create the same new file: exactly one may publish.
race_dir=$work/race
mkdir -p "$race_dir"
race_pids=""
for _ in 1 2 3 4 5 6 7 8 9 10 11 12; do
  (cd "$work/run" && exec env -i PATH="$empty" HOME="$empty" "$bin" simulate --input "$input" --format json --out "$race_dir/r.json" >/dev/null 2>&1) &
  race_pids="$race_pids $!"
done
race_ok=0
race_refused=0
for pid in $race_pids; do
  if wait "$pid"; then race_ok=$((race_ok + 1)); else race_refused=$((race_refused + 1)); fi
done
check "concurrent writers: exactly one succeeds ($race_ok ok, $race_refused refused)" test "$race_ok" -eq 1 -a "$race_refused" -eq 11
check "concurrent writers: the published file is the golden" cmp -s "$race_dir/r.json" "$golden"
check "concurrent writers: no temporary file remains" test "$(ls -A "$race_dir" | tr '\n' ' ')" = "r.json "

# ---- repository without HEAD: simulate is unaffected and leaves it untouched -------
if command -v git >/dev/null 2>&1; then
  nohead=$work/nohead
  mkdir -p "$nohead"
  git init -q "$nohead"
  if git -C "$nohead" rev-parse --verify -q HEAD >/dev/null; then
    bad "fixture: a fresh repository must have no HEAD"
  else
    ok "fixture: fresh repository has no HEAD"
  fi
  before=$(cd "$nohead" && find .git -type f | LC_ALL=C sort | tr '\n' ' ')
  got=0
  (cd "$nohead" && sr simulate --input "$input" --format json) || got=$?
  check "simulate inside a repository without HEAD (exit 0)" test "$got" -eq 0
  check "simulate inside a repository without HEAD: golden bytes" cmp -s "$work/out" "$golden"
  check "simulate did not change the repository" test "$(cd "$nohead" && find .git -type f | LC_ALL=C sort | tr '\n' ' ')" = "$before"
else
  echo "note: git is not on this script's PATH; the no-HEAD repository check was not run"
fi

echo "binary-acceptance: $checks checks, $failures failure(s)"
[ "$failures" -eq 0 ]
