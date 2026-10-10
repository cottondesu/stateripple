#!/usr/bin/env bash
# gotest.sh <label> <go test arguments...>
#
# Runs `go test -json` and turns the result into a gate that cannot be green by
# accident:
#   * any failing test or package, build error or timeout fails (go's own status);
#   * any skipped test fails: a skip is never an acceptable way to pass;
#   * zero passing tests fails (an -run pattern that matches nothing is green in
#     plain `go test`);
#   * every test named in REQUIRE_TESTS (space separated, top-level names) must
#     have passed; with GOTEST_ONLY_REQUIRED=1 only those tests are run.
# The full JSON stream is kept in $RUNNER_TEMP (or $TMPDIR) as gotest-<label>.jsonl.
set -euo pipefail

if [ "$#" -lt 2 ]; then
  echo "usage: gotest.sh <label> <go test arguments...>" >&2
  exit 2
fi
label=$1
shift

# GOTEST_ONLY_REQUIRED=1 runs exactly the REQUIRE_TESTS names, so the -run
# pattern and the list of required tests cannot drift apart.
if [ "${GOTEST_ONLY_REQUIRED:-}" = 1 ]; then
  if [ -z "${REQUIRE_TESTS// /}" ]; then
    echo "gotest[$label]: GOTEST_ONLY_REQUIRED needs REQUIRE_TESTS" >&2
    exit 2
  fi
  names=$(echo $REQUIRE_TESTS | tr -s ' \n' '||')
  set -- -run "^(${names%|})\$" "$@"
fi

log="${RUNNER_TEMP:-${TMPDIR:-/tmp}}/gotest-$label.jsonl"
started=$(date +%s)

status=0
go test -json "$@" >"$log" || status=$?
elapsed=$(($(date +%s) - started))

# Test-level events only: package-level "skip" is go's "[no test files]".
passed=$(grep -c '"Action":"pass","Package":"[^"]*","Test":"' "$log" || true)
failed=$(grep -c '"Action":"fail","Package":"[^"]*","Test":"' "$log" || true)
skipped=$(grep -c '"Action":"skip","Package":"[^"]*","Test":"' "$log" || true)

echo "gotest[$label]: go test exit=$status passed=$passed failed=$failed skipped=$skipped elapsed=${elapsed}s log=$log"

problems=0

if [ "$status" -ne 0 ]; then
  echo "gotest[$label]: go test failed (exit $status)" >&2
  # Show failing tests and their output; the raw stream stays in the log.
  if command -v jq >/dev/null 2>&1; then
    jq -r 'select(.Action=="fail") | "FAIL \(.Package) \(.Test // "(package)")"' "$log" | sort -u | head -50 >&2 || true
    jq -r 'select(.Action=="output" and (.Output | test("^(--- FAIL|FAIL|panic:|\\s+\\S+\\.go:[0-9]+:)"))) | .Output' "$log" | head -80 >&2 || true
  else
    tail -n 40 "$log" >&2
  fi
  problems=$((problems + 1))
fi

if [ "$skipped" -ne 0 ]; then
  echo "gotest[$label]: $skipped test(s) were skipped; skipping is not allowed in CI:" >&2
  grep '"Action":"skip","Package":"[^"]*","Test":"' "$log" | sed 's/.*"Package":"\([^"]*\)","Test":"\([^"]*\)".*/  \1 \2/' >&2
  problems=$((problems + 1))
fi

if [ "$passed" -eq 0 ]; then
  echo "gotest[$label]: no test passed; the package or -run pattern selected nothing" >&2
  problems=$((problems + 1))
fi

for name in ${REQUIRE_TESTS:-}; do
  if ! grep -q "\"Action\":\"pass\",\"Package\":\"[^\"]*\",\"Test\":\"$name\"" "$log"; then
    echo "gotest[$label]: required test $name did not pass (missing, failed or not run)" >&2
    problems=$((problems + 1))
  fi
done

if [ "$problems" -ne 0 ]; then
  exit 1
fi
echo "gotest[$label]: OK"
