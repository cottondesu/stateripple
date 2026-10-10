# Validation evidence — StateRipple v0.1.0

Records below the first section were written while the project was named RiskLife (module `github.com/cottondesu/risklife`). They are kept as history; their binary names, module path and Mandala wording refer to that name. Before publication, local absolute paths in every record (home, workspace, backup and uniquely named temporary directories) were replaced by placeholders such as `<backup-directory>` and `<temporary-directory>`; commands, conditions and results were not changed. Text in angle brackets is a placeholder, not a literal or runnable path. Generic operating-system paths (`/private/tmp`, `/var` → `/private/var` on macOS) are kept where a result depends on them. The rename did not change any computation, canonical JSON or hash (see the first section).

## Current record: GitHub Actions hosted runs (2026-10-10)

The workflow `.github/workflows/ci.yml` has run four times on GitHub-hosted `ubuntu-24.04` runners (Linux 6.17.0-1022-azure x86_64; the version gate printed Git 2.55.0 in every job of every run). This closes the hosted items of the "Not verified" list in the CI-preparation record below; that record and its limits are unchanged history. No run published anything (`contents: read`, no upload step). Evidence:

| Run | Event | Commit | Result | Wall |
|---|---|---|---|---|
| [38026796484](https://github.com/cottondesu/stateripple/actions/runs/38026796484) | `push` to `main` | 31cf6e7 | every job green; `long-shuffle` skipped by design | 91 s |
| [38027059887](https://github.com/cottondesu/stateripple/actions/runs/38027059887) | `workflow_dispatch` (`long_shuffle` on) | 31cf6e7 | every job green; long shuffle ×10 green | 2 min |
| [38027821273](https://github.com/cottondesu/stateripple/actions/runs/38027821273) | `pull_request` (PR #1, `docs/verify-pr-ci-trigger` → `main`) | a8b4810 | every job green; `long-shuffle` skipped by design | 80 s |
| [38028313271](https://github.com/cottondesu/stateripple/actions/runs/38028313271) | `pull_request` (same PR, next commit) | a88ff5b | every job green; `long-shuffle` skipped by design | 93 s |

What the runs showed:

- Go matrix (`compat` 1.23 / 1.26 / 1.27: gofmt, `go vet`, CLI build, full suite): success in all four runs, job times 37–46 s.
- Enhanced validation (strict environment, `-race`, `-shuffle=on -count=2`, the five named groups): success in all four runs, job times 70–83 s.
- `linux-binary` (static build; Git-free/offline acceptance and a real `analyze`, each inside an empty network namespace — so `sudo unshare --net` and `setpriv` work on the hosted runner): success in all four runs, job times 21–41 s.
- `ci-ok`: success in all four runs.
- Long shuffle (`-shuffle=on -count=10`): green in run 38027059887 (job 116 s); skipped by its `if` condition on `push` and `pull_request`, as designed.
- PR trigger: runs 38027821273 and 38028313271 both have event `pull_request`, head branch `docs/verify-pr-ci-trigger`, base `main`; every required job green in both, no unexpected skip or failure. The two runs executed sequentially — the first completed before the second began.

Still not verified: concurrency cancellation (a newer run of the same pull request cancelling an in-progress one — PR #1's two runs executed sequentially, so no run overlapped or was cancelled), Windows, macOS Intel, FreeBSD/OpenBSD/NetBSD and illumos execution, and any hosted arm64 runner.

## Previous record: GitHub CI preparation (2026-10-10 JST)

Verdict: the CI configuration is written and validated locally; **it has not run on GitHub Actions**, so no Actions result is claimed anywhere. This record is not an independent audit or release approval. No commit, push, tag, release, remote or branch change was made; HEAD is still absent. `docs/CI.md` describes the workflow; this section records the evidence. The directory `<audit-directory>` below holds logs, toolchains, scratch copies and scripts outside the repository.

### Scope

Added `.github/workflows/ci.yml`, `scripts/ci/` (six shell scripts), `docs/CI.md`; edited `README.md`, `README.ja.md`, `CHANGELOG.md` and this file. A SHA-256 manifest of the 92 candidate files was taken before the first edit (outside the repository). Afterwards every one of the 89 other files, including all `.go` files, `go.mod`, the six `testdata` files, `.gitignore`, `LICENSE`, SCHEMA, ARCHITECTURE and the design record, is byte-identical; the four edited documents are the only changes. RiskPropagationV1, the snapshot schema, goldens and the Git and Go adapters are untouched, and no golden was regenerated. `git add -A --dry-run` with a throwaway `GIT_INDEX_FILE` lists 100 files (92 + workflow + six scripts + `docs/CI.md`); the real index stays empty.

### Action versions

Resolved with the GitHub API on 2026-10-10, read-only: `actions/checkout` latest release `v7.0.1` → `3d3c42e5aac5ba805825da76410c181273ba90b1`; `actions/setup-go` latest release `v7.0.0` → `b7ad1dad31e06c5925ef5d2fc7ad053ef454303e`. The tag refs are commits, not annotated tags, and the moving `v7` tags resolve to the same commits. Both actions run on `node24`. `setup-go` v7 enables its cache by default; the workflow sets `cache: false`. `persist-credentials` defaults to true in checkout; the workflow sets false. The repository on GitHub exists, is public and empty; its default workflow permission is `read`, Actions may not approve pull requests, and "require pinned SHAs" is off.

### Go compatibility, run locally with real toolchains

Official archives from go.dev, SHA-256 checked against the release list before use (a first download without `-L` was an HTML redirect page and was rejected by the checksum). Strict environment (`GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local GOWORK=off GIT_NO_LAZY_FETCH=1 GIT_OPTIONAL_LOCKS=0 GIT_TERMINAL_PROMPT=0`, `GOFLAGS` unset), Homebrew Git 2.52.0.

| Platform | Go | gofmt | `go vet ./...` | CLI build | `go test -count=1 -timeout=5m ./...` |
|---|---|---|---|---|---|
| macOS arm64 | 1.23.12 | exit 0 | exit 0 | exit 0 | exit 0, 504 passed, 0 failed, 0 skipped, 60 s |
| macOS arm64 | 1.26.9 | exit 0 | exit 0 | exit 0 | exit 0, 504 passed, 0 skipped, 71 s |
| macOS arm64 | 1.27.2 | exit 0 | exit 0 | exit 0 | exit 0, 504 passed, 0 skipped, 66 s |
| Linux arm64 container (Debian 13, Git 2.47.3, `--network none`, uid 1000) | 1.23.12 | exit 0 | exit 0 | exit 0 | exit 0, 501 passed, 0 skipped, 28 s |
| same | 1.26.9 | exit 0 | exit 0 | exit 0 | exit 0, 501 passed, 0 skipped, 32 s |
| same | 1.27.2 | exit 0 | exit 0 | exit 0 | exit 0, 501 passed, 0 skipped, 33 s |

Go 1.23.12 therefore builds and passes the full suite on macOS arm64 and Linux arm64: the earlier "Go 1.23 NOT VERIFIED" is closed for those two platforms (not on a GitHub-hosted runner). The 3-test difference between macOS and Linux is `TestExistingOutputRetainsDarwinACLAndXattr` and its two subtests (macOS only). Go 1.27.2 is the version `setup-go` will resolve for `1.27`; the earlier records used 1.27.1.

Strict-environment enhanced checks, run with the workflow's own wrapper `scripts/ci/gotest.sh`:

| Check | macOS arm64, Go 1.27.1 | Linux arm64 container, Go 1.27.2 |
|---|---|---|
| `-race -count=1 -timeout=5m` | 504 passed, 0 skipped, 96 s | 501 passed, 0 skipped, 68 s |
| `-shuffle=on -count=2 -timeout=10m` | 1,008 passed, 0 skipped, 147 s | 1,002 passed, 0 skipped, 69 s |
| five named groups (generic graph 43, snapshot golden 6, Git-free 14, offline security 30, output safety 59 results) | all required tests passed, 0 skipped | same counts, all passed, 0 skipped |

### Git below 2.45, executed for real

An early run used the machine's Apple Git 2.30.1 (`/usr/bin/git`, ahead of Homebrew on `PATH` by mistake): all three toolchains gave 406 passed, 98 failed, 0 skipped, each failure being `analyze requires Git 2.45.0 or later with --no-lazy-fetch … unknown option: --no-lazy-fetch`. Too-old Git fails; it does not skip. This replaces the earlier statement that a real Git below 2.45 had never been executed. The same real Git is rejected by `scripts/ci/require-git.sh` (exit 1, "found 2.30").

### Helper scripts and how they were checked

| Item | Check | Result |
|---|---|---|
| `require-git.sh` | `test-require-git.sh` runs it against 14 stand-in Git executables (2.45.0, 2.52.0, 2.100.0, 3.0.0, `windows` suffix pass; 2.44.2, 2.9.9, 2.5.0, 1.99.0, Apple Git 2.39.5, garbled and empty banner, option rejected, no Git fail) and against the real Git | 14/14; a mutant that compares as strings fails 3 cases (2.100.0, 2.9.9, 2.5.0) |
| `gotest.sh` | real packages (pass), `-run` matching nothing, a required test present and one missing, a skipped test, a failing test, a build error | each case gives the intended status; skip, zero tests and missing required test exit 1 |
| `ci.yml` structure | a Ruby/Psych checker (outside the repository) with 39 assertions: triggers exactly `push`/`pull_request` on `main` plus `workflow_dispatch`; no `pull_request_target`, `workflow_run`, schedule; permissions exactly `contents: read`; only the two official actions, pinned to the 40-hex SHAs; `persist-credentials: false`; `cache: false`; matrix `["1.23","1.26","1.27"]`; `GOTOOLCHAIN=local`; strict variables; every job has `timeout-minutes` and `ubuntu-24.04`; no `|| true`, `set +e`, `continue-on-error`, secret, publishing or release command; only the static matrix value inside `run:`; every referenced script exists and is executable | 39/39; 22 workflow mutants (unpinned action, write permission, persisted credentials, cache on, matrix change, extra action, `|| true`, `pull_request_target`, secret, `git push`, `gh release`, untrusted expression, …) are all detected |
| `binary-acceptance.sh` | built binary, `env -i`, empty `PATH`: help/version, three goldens byte-for-byte and by pinned report and `input_sha256`, canonical bytes independent of input order, five repeats, exit codes 0/1/2 (10 error cases, generation limit), `--out` No-Clobber, hard-link/symlink/dangling/directory/trailing-slash/alias refusals, 12 concurrent writers (exactly one wins), repository without HEAD | 112 checks on macOS, 115 inside a network namespace, 0 failures |
| `analyze-acceptance.sh` | real `analyze`: complete run, deterministic bytes, HEAD/index/object store/work tree unchanged, `--out` rules, outside a repository, dirty tree, unknown base, Git 2.44.2 stand-in rejected with Go never started, `blob:none` partial clone with two missing base blobs (exit 2, canary upload-pack silent, object store and missing list unchanged, control shows the canary reacts when lazy fetch is allowed) | 51 checks on macOS and in the Linux container, 0 failures |
| acceptance scripts, mutation | six product mutants in a copy outside the repository: lazy fetch allowed, Git gate bypassed, incomplete exits 0, No-Clobber publish replaced by rename, attenuation off by one, output mode 0644 | 6/6 detected. The first run missed the rename mutant (a sequential run is stopped by the pre-check); the concurrent-writer check was added and detects it |
| `in-netns.sh` | non-root uid 1000 in a container with a setuid `sudo` stand-in: new network namespace (inode differs), uid/gid kept, environment passed through, no-argument call exits 2; both acceptance scripts inside it | pass. `/sys/class/net` showed the host namespace's devices inside the new namespace and fallback tunnel devices exist in every namespace on that kernel, so the check uses the empty routing table and failing connections instead; found by running it, not by reading it |

What this does not show: `sudo unshare --net` and `setpriv` on a hosted runner (the stand-in is a compiled setuid program in a container), x86-64 execution, and the behavior of the workflow engine.

### Public file hygiene

`git add -A --dry-run` (throwaway index): 100 files, about 600 KB, largest `docs/design/2026-10-09-generic-harness-spec.ja.md` (86 KB). Nothing under `.omx/`, `.mandala/`, `bin/`; no `risklife`, `stateripple` or `.exe` binary, no NUL-containing file, no symlink; the only executables are the six scripts. A scan found no credential pattern (AWS, GitHub, Slack, OpenAI/Anthropic-style keys, private-key headers, quoted password/token assignments), no user name, home or workspace path, and only `*@example.invalid` addresses. The 25 internal references (`refs/t3/*`, `refs/codex/*`) are unchanged and are not sent by `git push origin main`; `git push --mirror`, `--all` and `--tags` must not be used.

### Not verified

Everything that requires a GitHub-hosted run: the workflow as parsed by GitHub, hosted-runner Git version and speed, `-timeout=5m` margins there, `sudo unshare --net`/`setpriv` there, Linux x86-64 execution. Windows, macOS Intel and the BSDs were not executed. Cross-compilation was not repeated.

## Previous record: StateRipple rename and generic graph harness (2026-10-10 JST)

Verdict: implementation and validation complete in the measured scope; ready for an independent audit. This record is not an independent audit or release approval.

Environment: Go 1.27.1 darwin/arm64 (Apple M1, `go.mod` directive 1.23), Git 2.52.0 (Homebrew). Every Go command ran with `GOTOOLCHAIN=local GOPROXY=off`. No network access, dependency or toolchain download, image pull, commit, push, tag or release. HEAD is absent; every source file is untracked, so `git diff` cannot show changes. A pre-change SHA-256 manifest of all 95 files and a full source copy were stored outside the repository (`<backup-directory>`, a local directory beside the checkout, together with baseline binaries, corpora, scripts and logs).

### Scope

| Change | Files |
|---|---|
| Rename to StateRipple (module, imports, `cmd/stateripple`, help/version/diagnostic prefix, Text heading, test strings, docs) | `go.mod`, 59 import lines, `cmd/stateripple/main.go`, `internal/cli/cli.go`, `internal/report/text.go`, tests, docs, `.gitignore` |
| Generic node/edge kinds and attenuation 0..3 in the core | `internal/graph/graph.go`, `internal/snapshot/snapshot.go` |
| Go vocabulary moved to adapter profile validation | `internal/adapters/profile.go`, `internal/adapters/analyze.go` |
| Mode-dependent Text and generic help | `internal/report/text.go`, `internal/cli/cli.go` |
| GOFLAGS/GOEXPERIMENT/GOWORK test isolation | `internal/adapters/main_test.go`, `internal/cli/cli_test.go` (`childEnv`) |
| New tests and fixture | `internal/graph/graph_test.go`, `internal/rules/risk/generic_test.go`, `internal/adapters/profile_test.go`, `internal/cli/gitfree_test.go`, additions to snapshot/report/adapters tests, `testdata/generic-supply-chain{,.report}.json` |

### Compatibility (pre-change golden baseline)

Before any edit, the original source was built (`risklife-before`) and run on: `testdata/example.json`, `testdata/cli-empty.json`, 300 generated legal snapshots in the former vocabulary (random graphs with Unicode/HTML/control-character IDs, exact duplicates, shuffled order, ASCII/UTF-8 encodings, indentation, limits 1..1024; 281 complete, 19 incomplete) and 27 invalid inputs. JSON bytes, Text, stderr and exit codes were recorded.

| Comparison | Result |
|---|---|
| Rename-only binary vs baseline | 300/300 JSON byte-identical, exit codes identical, Text identical except the heading line; 27/27 invalid inputs still exit 1 with identical messages apart from the `stateripple:` prefix |
| Final binary vs baseline | 300/300 JSON byte-identical (hash, final risk, generation count, waves, causes, test targets, stop reason), exit codes identical. 27/27 invalid inputs exit 1, no stdout. One message changed: an edge pair with kinds `prod_import` and `other` is now rejected as "conflicting edge kinds for endpoints" instead of "invalid edge kind". Text changed in all 300 only by removing the Git "Changed files" section and, without test targets, the test section and footer; a script confirmed 0 other differences |
| Pinned goldens | `example.json` → `input_sha256` `e0c96ca2…acde8`, report SHA-256 `809eb4c4…f36a`; `cli-empty.json` → `a246bcb6…6818`, report `c215f338…bda2`. Both unchanged and pinned by `TestGitFreeAcceptance` and existing golden tests. Existing golden files were not regenerated |
| Large former-vocabulary graph (20,000 nodes, 200,000 edges, 1,000 seeds) | Final and pre-change binaries produce identical report bytes |
| `analyze`, pre-change vs final binary, 8 fixture repositories (source, test-only, embed, README, unknown file, go.mod, deleted test, new package) | 8/8 JSON bytes and exit codes identical (6 complete/exit 0, 2 incomplete/exit 2) |
| Independent hash | A Python implementation of normalization and canonical encoding reproduces `input_sha256` for `example.json`, `cli-empty.json` and `generic-supply-chain.json` |

The only expected-result change in an existing test is `TestRejectWhenDomainInputInvalid`: the six former Go-vocabulary cases (node_kind, edge_kind, test_out, prod_to_test, test_to_prod, attenuation 0 on `prod_import`) are now valid generic input. They moved to `TestNormalizeAcceptsGenericVocabulary` (accepted by the core) and `TestGoProfileRejectsFormerCoreCases` (still rejected by the Go adapter, plus five more attenuation/target cases). New core rejection cases cover empty/oversized/invalid-UTF-8 kinds and attenuation −1, 4, 5, MaxInt and MinInt.

### Generic graph and RiskPropagationV1

The engine formula and code are unchanged. `TestGenericGraphScenarios` covers empty, single node, linear chain, branching with attenuation 0/1/2/3, zero-attenuation cycle, self-loops, disconnected components, multiple seeds, duplicate seeds, risk-0 seed, attenuation 0..3 (risk-3 contributions 3, 2, 1, 0) and custom kinds; `TestEqualRiskSourcesListEveryCause`, `TestZeroAttenuationCycleCauses`, `TestNonSoftwareSupplyChain`, `TestGenericLargeSparseGraph` (20,000 nodes, 40,000 edges) and `TestGenericRejectsInvalidAttenuation` complete the matrix. Every case is checked against two independent computations inside the test: a map-based synchronous reference written from the formula (final state, generation count, each wave's changes, status, stop reason, test targets) and, for converged runs, a Dijkstra minimum-attenuation path model of the least fixed point. Every wave cause is verified to satisfy `R_prev(u) − a = new value`. 1,000 random generic graphs (`TestGenericMatchesReferenceWhenRandomGraphs`) also satisfy `generation_count ≤ max(1, |V|)` when complete.

Outside the test suite, a standalone Python reference (no shared code) was run against the final binary on 1,000 random generic snapshots with Unicode/HTML IDs and kinds, attenuation 0..3, duplicates and limits 1..1024: 1,000/1,000 matched final state, generation count, waves, wave-0 seeds, status, stop reason, exit code and `input_sha256` (942 complete, 58 incomplete).

Zero attenuation: an n-node chain needs n updates. With limit 1024, n = 1024 completes (exit 0, 1024 updates); n = 1025 stops at `generation_limit`, `incomplete`, exit 2, although its last node already reached 3 (no confirming update); n = 1026 and 3,000 leave the tail at 0 and are incomplete. No unconverged run reported `complete`. The limit 1..1024 is unchanged.

Finding during implementation (documented, not changed): in an `incomplete` run stopped by the generation limit, a positive cell can have empty final `causes` when its supporting neighbor increased afterwards (`TestIncompleteFinalCausesMayBeEmpty`). It occurred in none of the 19 incomplete baseline reports. Changing the final-cause computation could alter existing incomplete report bytes, so the behavior is documented in SCHEMA/ARCHITECTURE. Waves still record the cause and the result is never complete.

Correction (post-audit SR-03, 2026-10-10): this paragraph previously said the case "cannot occur with the former uniform attenuation". That was wrong. The former vocabulary is not uniformly attenuated: `own_test` has attenuation 0. The pre-change binary reproduces the case with `pkg:A` (seed 3) → `pkg:B` (seed 1) by `prod_import`, `pkg:B` → `test:B` by `own_test` and `max_generations` 1: `test:B` ends at risk 1 with no final cause, and the pre-change and current report bytes are identical. It is therefore existing behavior of unconverged runs, not something the generic vocabulary introduced. `analyze` does not produce it, because it always emits limit 16 and a Go profile that converges within 4 updates (see the post-audit record below).

### Git-free acceptance (`TestGitFreeAcceptance`, built binary, `PATH` = empty directory, minimal environment)

| ID | Result |
|---|---|
| GF-01 no `.git` | PASS: three goldens byte-identical, report and input hashes pinned |
| GF-02 `git init` without HEAD | PASS: HEAD absence asserted; goldens identical with empty PATH and with Git on PATH |
| GF-03 no Git executable | PASS |
| GF-04 no Go executable (prebuilt binary) | PASS |
| GF-05 help/version without Git/Go | PASS: `help`, `--help`, `-h`, `version`, `simulate --help`, `analyze --help` exit 0 on stdout |
| GF-06 custom node kind | PASS (`sensor`, `アクチュエータ`) |
| GF-07 custom edge kind | PASS (`actuates` used as cause reason) |
| GF-08 attenuation 0 | PASS: risk 3 → 3 |
| GF-09 attenuation 3 | PASS: risk 3 → 0, one update, no activation |
| GF-10 attenuation 4/5 | PASS: 4, 5, −1, `2.0`, `"1"`, 99999999999999999999 → exit 1, no stdout, diagnostic names attenuation |
| GF-11 long zero-attenuation chain | PASS: 1024 nodes complete/exit 0; 1025 and 1500 incomplete/`generation_limit`/exit 2 |
| GF-12 analyze without Git | PASS: exit 1, "analyze requires Git 2.45.0 or later" |
| GF-13/GF-14 same snapshot | PASS: 25 repeated executions byte-identical, same `input_sha256` |
| GF-15 non-software graph | PASS: supply-chain golden; Text contains no Git/test wording |

### GOFLAGS isolation

Cause: `Analyze` intentionally inspects its own process environment and reports nonempty `GOFLAGS`/`GOEXPERIMENT` (and explicit `GOWORK`) as incomplete, while its go children always receive a clean policy. Tests running `Analyze` in-process therefore depended on the developer's environment. Fix: `internal/adapters/main_test.go` clears only these three variables once in `TestMain`, before any test runs (no adapters or cli test uses `t.Parallel`); tests that exercise them set them with `t.Setenv`. CLI tests give child processes explicit `exec.Cmd.Env` (`childEnv`) and never modify the test process environment; the test build uses `GOFLAGS= GOWORK=off GOTOOLCHAIN=local GOPROXY=off`. Production behavior and the offline child policy are unchanged; no global Go configuration was written.

| Parent environment for `go test ./...` | Before | After |
|---|---|---|
| unset / empty | PASS | PASS |
| `GOFLAGS=-mod=readonly` | 15 FAIL (adapters 14, cli 1) | PASS |
| `GOFLAGS=-mod=mod`, `-tags=custom`, `-mod=vendor`, `-count=1` | not measured | PASS |
| `GOFLAGS=-definitely-not-a-go-flag` | `go` itself refuses to run (`parsing $GOFLAGS`) | same; compiled adapters and cli test binaries run with this value: PASS |
| `GOFLAGS=not-a-flag`, `GOEXPERIMENT=nosuchexperiment`, `GOWORK=/nonexistent/go.work` inherited by compiled test binaries | not measured | PASS |

`TestAnalyzeReportsInheritedGoPolicy` checks that Analyze still reports incomplete for valid, invalid and analysis-changing GOFLAGS and GOEXPERIMENT without applying them; `TestInheritedGoPolicyClearedBeforeTests` checks the isolation.

### Full suite

| Command (`GOTOOLCHAIN=local GOPROXY=off GOFLAGS=`) | Result |
|---|---|
| `gofmt -l .` | empty |
| `go vet ./...` | PASS |
| `go test -count=1 ./...` | PASS (all 8 packages with tests) |
| `go test -race -count=1 ./...` | PASS |
| `go test -shuffle=on -count=10 -timeout 30m ./...` | PASS (adapters 620 s). A first run without `-timeout` hit Go's default 10-minute limit in adapters while fuzzing ran concurrently; it was a timeout, not a test failure |
| Fuzz, 60 s each: `FuzzRead`, `FuzzPropagation`, `FuzzGenericPropagation` | PASS: 6.4 M / 2.2 M / 1.5 M executions, no failing input |
| `go build ./cmd/stateripple` | PASS |

Previous findings (RL-001–RL-010, NF-01, NF-02, NEW-01–NEW-04) are covered by the unchanged regression tests listed in the earlier records (e.g. `TestAnalyzeRejectsHiddenHeadMismatch`, `TestAnalyzeGitlinkCannotBeHidden`, `TestAnalyzeNativeSourceChanges`, `TestSimulateRejectsOutputThatAliasesInput`, `TestTextEscapesEveryUntrustedString`, `TestExecutableRejectsFIFOInputWithoutBlocking`, `TestAnalyzeCachedVersionedReplacement`, `TestExistingOutputRetainsDarwinACLAndXattr`, `TestGoCommandIgnoresSavedEnvironmentAndInjectedFlags`, `TestReplaceCompletenessDependsOnReachablePackages`, `TestGitMissingBlobNeverFetches`, `TestExecutableRejectsDirectoryOutputSpellings`, `TestAnalyzeRejectsUnsupportedGitBeforeGoCollection`, `TestAnalyzeTraceDoesNotChangeCompleteness`, `TestExecutableResolvesOutputParentBeforeDotDot`); all pass in the normal, race and shuffle runs. The Go profile move is checked by `TestGoProfileRejectsFormerCoreCases`, `TestAnalyzeCollectsGoProfileGraph` and the analyze before/after comparison above.

### Performance (Apple M1, best of 5 in-process; CLI best of 3)

| Measurement | Former vocabulary (attenuation 1) | Generic (attenuation 0..3) |
|---|---|---|
| Input | 20,000 nodes, 200,000 edges, 1,000 seeds, 16.2 MB | same sizes, 15.7 MB |
| `graph.Normalize` | 64 ms | 62 ms |
| `snapshot.Read` (bounded decode, validation, normalization) | 498 ms | 511 ms |
| `snapshot.Hash` (normalization, canonical JSON, SHA-256) | 51 ms | 50 ms |
| `risk.Run` (normalization, hash, engine, trace) | 97 ms (3 updates) | 108 ms (12 updates) |
| `report.JSON` | 26 ms (10.3 MB) | 44 ms (16.8 MB) |
| `stateripple simulate … --out` end to end | 0.68 s, max RSS 186 MB (pre-change binary 0.73 s, identical bytes) | 0.73 s, max RSS 190 MB |

`analyze` end to end on generated single-module repositories (10 packages, one changed file, measured now, not copied from earlier reports): 100 Go files 2.06–2.41 s; 1,000 files 18.5 s; 10,000 files 177 s (one run). Base embed analysis still starts one `git cat-file blob` per base Go file; a streamed `cat-file --batch` reader remains a v0.1.1 candidate and was not implemented.

### Platforms and toolchains

| Target | Status |
|---|---|
| macOS arm64 (Darwin) | Runtime: full suite, race, shuffle, fuzz |
| Linux arm64 | Runtime: static binary in the locally present `postgres:15` image (Debian 13), `--pull=never --network=none`, no git/go on PATH: version/help exit 0; three goldens byte-identical by SHA-256; analyze exit 1 with the Git requirement message; `--out` creates and then refuses to overwrite. The Go test suite was not run on Linux |
| Linux amd64, linux/386, windows amd64/arm64, darwin amd64, freebsd amd64 | Cross-compiled binary and `go vet` including tests: PASS. Not executed |
| illumos amd64 | Binary cross-compiles. `go vet` of tests fails: `internal/cli/cli_io_unix_test.go` (unchanged since before this work) uses `syscall.Mkfifo`, which illumos lacks. Test-only portability issue, not fixed here |
| Go 1.23 toolchain | NOT VERIFIED: no Go 1.23 toolchain is installed and downloads are prohibited |
| Git below 2.45 | `analyze` rejects it (existing tests); `simulate`, `help` and `version` need no Git |

### Post-audit remediation: SR-01, SR-02, SR-03, SR-07 (2026-10-10 JST)

Scope: the four findings of the independent audit that were selected for remediation before publication. No product Go source, `go.mod`, testdata, golden, schema or calculation changed: all 27 non-test `.go` files, `go.mod` and the six testdata files are byte-identical to their pre-remediation SHA-256, and a `-trimpath` build of a copy without `.git` is byte-identical to the audit-time binary (`434edfd1…a37880`). This record is not an independent audit or release approval. No commit, push, tag or release was made; HEAD is still absent.

| Finding | Change | Check |
|---|---|---|
| SR-01 local state could be committed | `.gitignore` gains `/.omx/` and `/.mandala/`; `/risklife` stays ignored | `git check-ignore -v` names the new rules for `.omx`, `.mandala` and their files and reports nothing for sources, docs and testdata. `git add -A --dry-run` lists 89 files (`.gitignore`, 4 root documents, `go.mod`, `cmd` 1, `docs` 5, `internal` 71, `testdata` 6) and none under `.omx/`, `.mandala/` or `risklife`; no index was created. A scan of those 89 files found no credential pattern, no binary and only `*@example.invalid` fixture addresses |
| SR-02 omitted required fields were not isolated by tests | `TestReadRejectsOmittedRequiredField`, `TestReadDistinguishesZeroNullAndOmission` (`internal/snapshot`) and `TestSimulateRejectsOmittedOrNullField` (`internal/cli`). Each case starts from a snapshot that is accepted as written, removes or replaces exactly one field (the replaced text must occur exactly once) and requires the specific diagnostic; the CLI test also requires exit 1, empty stdout and the full stderr line | 28 new test results, all PASS. Mutation check on an isolated copy, running only the new tests: optional `attenuation`, optional `risk`, optional `kind`, optional `id`/`from`/`to`/`reason`/root fields, no required-field check, null accepted and explicit attenuation 0 rejected are each detected (7 of 7). Before this change the first two survived the snapshot, graph, rule, CLI, report and engine tests, because the only omission cases used an empty graph and were rejected for another reason anyway |
| SR-03 wrong statement about empty final causes | Corrected above and in the design record (§0.1, §6.3 P5); SCHEMA, ARCHITECTURE and both READMEs now state that a `generation_limit` result is a lower bound whose test targets and final causes can be incomplete | Pre-change binary and current binary give byte-identical reports for the Go-vocabulary reproduction (`max_generations` 1, exit 2, `test:B` risk 1 without final cause). 798 unconverged random runs: no cell above its converged value, 1,483 below, 83 test targets missing and 64 under-rated. 4,000 random Go-profile snapshots at limit 16: largest `generation_count` 4, none incomplete, no positive cell without a final cause, all reports byte-identical to the independent reference |
| SR-07 kind `test` is interpreted by the core | Behavior unchanged. README (both languages), SCHEMA and ARCHITECTURE no longer say the core assigns no meaning to kinds; they describe the exact-match `test` projection into `test_targets`, its effect on Text and that it also applies to `simulate`. The separation is recorded as a v0.2.0 item (README roadmap, design record §0.1 and §27) | Built binary: kind `test` with positive risk is listed and Text uses test wording; `test` at risk 0, `Test`, `tests` and `lab` are not listed. The only kind comparisons outside the Go profile are edge ordering/deduplication in `internal/graph` and `n.Kind == "test"` in `internal/rules/risk/run.go` |

| Command (`GOTOOLCHAIN=local GOPROXY=off GOWORK=off`, `GOFLAGS` unset, run in the repository) | Exit | Result |
|---|---|---|
| `gofmt -l .` | 0 | empty |
| `go vet ./...` | 0 | no findings |
| `go test -count=1 ./...` | 0 | 430 results, 0 failed, 0 skipped (402 before this change) |
| `go test -race -count=1 ./...` | 0 | PASS |
| `go test -shuffle=on -count=10 -timeout 30m ./...` | 0 | PASS; adapters 604.6 s. The timeout is explicit because adapters took 634 s in the audit run, which exceeds Go's default 10 minutes |
| Goldens with the built binary | 0 | `example.report.json`, `cli-empty.golden.json`, `generic-supply-chain.report.json` byte-identical; `input_sha256` `e0c96ca2…acde8`, `a246bcb6…6818`, `c34ed81c…93de`; the pre-rename goldens in the backup are also reproduced |
| Git-free `simulate` | 0 | empty `PATH` and environment outside any repository, and a sandbox that denies fork, network and file writes: golden byte-identical; `analyze` without Git exits 1 |

Residual: Go 1.23, Linux `analyze`/test suite and Windows/BSD/illumos execution remain not verified. SR-04, SR-05, SR-06 and SR-08 of the audit are not addressed here. The repository-internal checkpoint references (`refs/t3/*`, `refs/codex/*`) contain snapshots of `.omx/`; they were inspected only, are not sent by an ordinary branch push, and must not be published with `git push --mirror`. The comment in `internal/graph/graph.go` ("The core assigns them no meaning") was left unchanged because product source was out of scope.

### Final cross-model audit remediation: CA-01–CA-05 and the SR-07 comment (2026-10-10 JST)

Scope: the five findings of the final cross-model audit (Codex) and the remaining source comment of SR-07. This is remediation evidence for a targeted re-audit, not an audit or a release approval. RiskPropagationV1, the snapshot schema, canonical JSON, report shape, the Git and Go adapters, `go.mod` and all six testdata files are unchanged (testdata SHA-256 identical to the pre-change manifest). No commit, push, tag, release, remote change, toolchain or dependency download, or network access. HEAD is still absent, so changes were tracked with a SHA-256 manifest of the 91 files taken before the first edit: 3 files added, 12 changed (one of them the CLI-owned Mandala state), 79 unchanged, none removed. Fixtures, mutants and binaries were built in `<temporary-directory>/stateripple-remediation`, outside the repository.

Each finding was reproduced on the unchanged source before it was edited:

| Finding | Reproduction on the pre-change source |
|---|---|
| CA-01 | Pre-change binary, missing input and existing output whose names contain LF and ESC: exit 1 and files intact, but stderr holds the raw bytes `0a` and `1b`; an unknown flag name is printed raw by the `flag` package as well |
| CA-02 | `docs/VALIDATION.md` and the design record contained the local user name and absolute home, workspace, backup and session-scratch paths |
| CA-03 | `GIT_NO_LAZY_FETCH=1 go test -run TestGitMissingBlobNeverFetches ./internal/adapters`: exit 1, both subtests stop in `git clone --filter=blob:none` (checkout exits 128, "could not fetch … from promisor remote") before `Analyze` is called. Without the variable the same test passes. A fixture defect, not a product fetch |
| CA-04 | `docs/SCHEMA.md`, `docs/ARCHITECTURE.md` and the design record §0.1 said that a limit-stopped final state "is not a fixed point"; the 1025-node zero-attenuation chain with limit 1024 ends with every value final and is still, correctly, `incomplete` |
| CA-05 | Pre-change binary with stdout opened read-only: `help`, `--help`, `simulate --help`, `analyze -h` exit 0 with no diagnostic; `version` exits 1 |

| Finding | Change | Files |
|---|---|---|
| CA-01 | Every stderr diagnostic passes through one function that escapes the complete formatted message: runes rejected by `strconv.IsGraphic` (C0/C1 controls including LF, CR, ESC and NUL, DEL, Unicode line and paragraph separators, bidirectional and other format controls) and invalid UTF-8 bytes become the Go escapes the text report uses. Escaping the final string, not each call site, covers wrapped `*os.PathError`/`*os.LinkError`, joined errors and Git/Go tool output, and leaves no second route for a raw path. Backslashes and quotes are not escaped, so ordinary and Windows paths and `%q`-quoted values are unchanged and nothing is escaped twice. The `flag` package no longer writes to stderr; the CLI prints its escaped error followed by the unchanged usage text. The report's own `escaped` helper was considered and not reused, because it also escapes `"` and `\`, which would change every diagnostic that quotes a value | `internal/cli/diagnostic.go` (new), `internal/cli/cli.go` |
| CA-05 | Help, usage and version text are written through one checked writer: an error or a short write gives `stateripple: write output: …` and exit 1 | same files |
| CA-03 | Test only. The fixture is built in explicit local steps: `clone --no-checkout --filter=blob:none` from a `file://` origin (commits and trees), the HEAD blobs copied as one local pack (`pack-objects` → `unpack-objects`), then a checkout with `GIT_NO_LAZY_FETCH=1`. Fixture commands run with a pinned environment (lazy fetch forbidden, no prompt, no trace, `GIT_ALLOW_PROTOCOL=file`), require the Git version that `analyze` requires, and report every failure as `fixture setup:`. The fixture asserts a clean checkout, a promisor remote and that the absent objects are exactly the two base blobs. The offline policy of the adapter (`internal/adapters/command.go`) is unchanged | `internal/adapters/git_offline_unix_test.go` |
| CA-04 | The three statements are replaced by: reaching the fixed point is not confirmed; the final state can already be the fixed point; it is `incomplete` because that is unconfirmed. ARCHITECTURE separates "is a fixed point", "the engine confirmed it", "stopped at the limit before confirming" and "`complete`". The design record keeps the original sentence's meaning with a dated correction; its §6.2 was already correct. No calculation or exit code changed | `docs/SCHEMA.md`, `docs/ARCHITECTURE.md`, design record |
| CA-02 | Local absolute paths replaced by `<backup-directory>`, `<temporary-directory>`, `<workspace>`, `<project-root>` and `<evidence-dir>`, each introduced as a placeholder that is not a runnable path; commands, conditions and results kept. Kept on purpose: `/tmp/…` and `/path/to/stateripple` in README examples (generic and runnable), `~/go/bin/mandala` (generic), `/private/tmp` and `/var` → `/private/var` (macOS facts that explain recorded results), and the public account name in the module path and license | `docs/VALIDATION.md`, design record |
| SR-07 | The comment "The core assigns them no meaning" now names the exception: the risk rule projects kind `test` into `test_targets`. No behavior change | `internal/graph/graph.go` |

Normal behavior of the changed product code. The pre-change and final binaries (`go build -trimpath`, SHA-256 `5e77537c…575b` and `ae8fa5bc…b6d4`) were run on 55 invocations without control characters (help forms, version, usage errors, invalid and missing inputs, the three goldens in JSON and Text, a limit-stopped snapshot, every output refusal, `analyze` outside a repository): stdout, stderr and exit code are byte-identical in all 55 (19 exit 0, 31 exit 1, 2 exit 2, plus 3 `--out` files with identical bytes and mode 0600). 240 seeded random lawful snapshots (230 complete, 10 incomplete) give identical JSON, Text, stderr and exit code from both binaries, and permuted, duplicated input gives identical report bytes. Intended differences: diagnostics containing nongraphic characters, including line breaks of joined errors and of multi-line Git/Go messages, which are now `\n` inside one line; and the exit code of a failed help, usage or version write.

Tests added. `internal/cli/diagnostic_test.go`: identity on graphic text (Japanese, emoji, Windows path, `%q` text); every Unicode code point and every invalid byte against the text report's escape; idempotence; wrapped, twice-wrapped, link and joined errors; hostile command, flag name, flag syntax, input path (with NUL and an invalid byte) and snapshot value through `Run`; exact help and usage bytes, stream and exit code for 13 invocations; write failure at the first byte, midway and as a short write for 8 invocations, and usage on a failing stderr. `internal/cli/diagnostic_unix_test.go`: 12 real file-system cases with a hostile name (missing, directory and dangling-symlink input; existing, symlink, directory, same-file and hard-link output; missing, dangling-symlink and unwritable output directory; trailing separator), each requiring exit 1, empty stdout, one line of graphic runes with the escaped name, and an unchanged directory tree. `TestGitMissingBlobNeverFetches` now runs seven inherited environments (none; `GIT_NO_LAZY_FETCH` 1, 0 and empty; `GIT_TERMINAL_PROMPT=0`; trace variables; all hostile values together) on a fixture with two missing base sources and one present source with an embed directive, and checks: incomplete result, both missing sources named, the present one not named, no upload-pack call, missing-object list and every file below `.git/objects` unchanged; then, as a control, that the same read with lazy fetching allowed does reach the canary and transfers nothing. `TestGitChildrenRejectPromptAndOverrideInheritedEnvironment` additionally requires the `--no-lazy-fetch` option.

| Command | Environment | Exit | Result |
|---|---|---:|---|
| `gofmt -l .` | strict | 0 | empty |
| `go vet ./...` | strict | 0 | no findings |
| `go test -count=1 -timeout=5m ./...` | strict | 0 | 71 s; adapters 70.7 s |
| `go test -race -count=1 -timeout=5m ./...` | strict | 0 | 78 s; no race report |
| `go test -shuffle=on -count=2 -timeout=10m ./...` | strict | 0 | 140 s; adapters 138.9 s |
| `go test -count=1 -timeout=5m -json ./...` | ordinary | 0 | 504 results (145 top-level tests), 0 failed, 0 skipped; 430 before this change |
| `go test -count=1 -timeout=5m ./internal/adapters ./internal/cli` | hostile | 0 | 72 s |

Strict is the environment in which the audit saw CA-03: `GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local GOWORK=off GIT_NO_LAZY_FETCH=1 GIT_OPTIONAL_LOCKS=0 GIT_TERMINAL_PROMPT=0`, `GOFLAGS` and `GOEXPERIMENT` unset. Ordinary has no Git policy variable. Hostile inherits `GIT_NO_LAZY_FETCH=0 GIT_TERMINAL_PROMPT=1 GIT_TRACE=1 GIT_TRACE2=1 GIT_OPTIONAL_LOCKS=1`. Go 1.27.1 darwin/arm64, Git 2.52.0. No run timed out; there was no fixture-setup failure and no assertion failure.

Mutation checks (copies outside the repository; a mutant counts as killed only when the targeted test fails by assertion, not by build error): 30 mutants, 28 killed and 2 expected survivors. Product mutants of the offline boundary: lazy fetch allowed by removing both the option and the environment pin (killed by the canary, "implicit fetch attempted"), a missing base blob no longer marking the result incomplete, an unreadable base source skipped, only the first missing source reported (all killed). Fixture mutants: full clone, base blobs copied with the HEAD blobs, HEAD blobs not prepared (all stop as `fixture setup`), canary not installed (stops as `fixture control`). Diagnostic mutants: escape disabled, LF, ESC, a bidirectional override, U+2028, a C1 control or an invalid byte left raw, backslash escaped too, one diagnostic call printed raw (three sites), `flag` writing to stderr again (all killed). Help mutants: write error ignored (four sites), short write accepted, defaults dropped from usage, usage sent to the wrong stream (all killed). The two survivors remove only the option or only the environment pin; each protection alone still prevents the fetch in this test. At package level both are killed by `TestGitChildrenRejectPromptAndOverrideInheritedEnvironment` after its change above; before it, removing the option alone survived the whole package.

Independent fixtures (a script separate from the Go tests, real processes, other hostile strings including an OSC sequence, backspace, U+061C and a tag character): 70 checks, 0 failed. Nine file-system cases give one escaped line and exit 1 with the directory tree unchanged, while the pre-change binary prints the raw bytes for the same nine; a hostile command name, flag name and `--base` revision each give one line of graphic characters, and a Git shim with a three-line hostile message yields one line. Help, usage and version on an unwritable stdout exit 1 (pre-change: 0 for the five help forms), with unchanged bytes on success. The three goldens are byte-identical with empty `PATH` and environment (`809eb4c4…f36a`, `c215f338…bda2`, `ebbb8c55…854a`); new output is 0600; a second write, the same path, a hard link and a symlink alias are refused with the input intact. Zero-attenuation chains with limit 1024: 1024 nodes complete, 1025 incomplete with tail 3, 1026 incomplete with tail 0; attenuation 0..3 from a risk-3 seed gives 3, 2, 1, 0. A partial clone built the same explicit way, run under seven inherited Git environments, with `PATH` wrappers recording any Git transport subcommand or network client, and inside a macOS sandbox that denies network access (a control connection inside the sandbox fails): exit 2, `incomplete`, both missing sources named, canary and wrapper logs absent, object store and missing-object list unchanged, no trace file, and identical report bytes in all seven. Controls: an allowed lazy fetch reaches the canary and transfers nothing; a full clone of the same history is `complete` with no warning and identical output from the pre-change binary; a rename appears as one deletion and one addition, identical in both binaries. The first run of this script reported 10 failures caused by its own expected strings (escape sequences stored as literal characters); the product output was already correct, the script was fixed, and the figures above are from the rerun.

Publication check: `git add -A --dry-run` with a throwaway index lists 92 files (89 before, plus the three new CLI files) and nothing under `.omx/`, `.mandala/` or `bin/` and no binary; `git check-ignore` still names the rules for local state and binaries. A scan of the 92 files for the local user name, absolute home paths, uniquely named temporary and backup directories, session identifiers, e-mail addresses and eight credential patterns has no match; a pattern scan cannot prove the absence of every secret. No repository index was created. Publication needs only an ordinary branch push; `git push --mirror` would also send the internal checkpoint references and must not be used.

Residual: Go 1.23, Linux `go test` and `analyze`, and Windows execution remain not verified. SR-04, SR-06, SR-08 and NF-03–NF-07 are unchanged and out of scope; diagnostics are escaped but still not length-limited (escaping expands a byte to at most 4 bytes and a rune to at most 10). SR-05: the adapters package now takes about 71 s per run (64.65 s for the whole suite in the audit, before five environment cases were added), so `-shuffle=on -count=10` is expected to exceed Go's default 10-minute package timeout; it was not rerun here, and CI must pass an explicit `-timeout` to every suite (5 minutes was sufficient for one run, 10 minutes for shuffle ×2, 30 minutes was used for ×10 before). The network sandbox and the unwritable-directory case are macOS observations; the unwritable-directory test skips only when run as the superuser.

## Previous record: NEW-01–NEW-04 / NF-04 remediation (2026-10-09 JST, Claude Code continuation, as RiskLife)

Verdict: implementation and validation complete in the measured scope. READY FOR FINAL TARGETED RE-AUDIT; this record is not an independent audit or release approval.

Codex implemented NEW-01–NEW-04 code, regressions and README/CHANGELOG/SCHEMA/ARCHITECTURE updates, then stopped at a token limit before validation records and Mandala updates. Claude Code kept that work, changed no Codex design decision, and completed the items below. Environment: Go 1.27.1 darwin/arm64, Apple M1, Git 2.52.0 (Homebrew). No network access, Docker image pull, dependency or toolchain installation, commit, push, tag or release. HEAD is still absent. Evidence root: `<temporary-directory>/rl-claude/evidence` (temporary): fixture scripts, logs, `risklife-after` binary. "Before" is Codex's post-NF-01, pre-NEW-01 binary `<temporary-directory>/risklife-final/risklife-after`.

### Changes in this continuation

| Change | Reason |
|---|---|
| embed.go: base-tree symlink ending in `.go` is an unsupported base source (incomplete) | Go compiles the symlink target, whose bytes are not the entry's blob. Skipping it let a former `//go:embed` owner disappear: before binary returned complete/0 with the formerly embedded `x/guide.md` as `excluded_document`; now incomplete/2 with `unknown`. Unit test `TestBaseSourcesRejectsSymbolicGoSource` failed first, then passed. |
| git_offline_unix_test.go: partial-clone test asserts a useful incomplete result (err nil, Incomplete, warning naming `a/a.go`) | The earlier assertion accepted any error or incomplete result. Strengthened, not weakened. |
| head_test.go, fileio_fault_test.go: resolve `t.TempDir()` physically | With macOS default TMPDIR (`/var` -> `/private/var`) these failed: the helper-level test sees a symlinked path, and NEW-04 reports the physically resolved published path. Both passed with physical TMPDIR; assertions are unchanged. |
| README/README.ja/CHANGELOG/SCHEMA/ARCHITECTURE | Document the symlinked base source rule and NEW-01 process cost. |

### Finding status and independent fixtures

New fixture scripts differ from the unit tests (different packages, patterns, payload kinds and environments).

| Finding | Status | Independent evidence |
|---|---|---|
| NEW-01 P1 | FIXED | `new01/verify_new01_doc.sh`: base has `x/x.go` (`//go:embed assets/*.txt guide.md`) and `y/y.go` (`//go:embed y.md`); HEAD drops x's embed and edits `x/guide.md` and `docs/notes.md`. `--filter=blob:none` local clone, fetch-blocking upload-pack. Before: partial complete/0, `x/guide.md` excluded_document, no warning (false complete; y's match hid x's missing blob). After: incomplete/2, `x/guide.md` and `docs/notes.md` unknown, warning names `x/x.go` and OID. Hostile `GIT_NO_LAZY_FETCH=0 GIT_TERMINAL_PROMPT=1 GIT_TRACE=1 GIT_TRACE2=1`: same. Full clone: complete/0, `former_embed`, notes excluded. upload-pack calls 0; missing objects 3 -> 3, list and object-store fingerprint unchanged. `verify_new01.sh` (.txt payload) also passes. Symlinked base `.go` (`verify_symlink_base.sh`): before complete/0, after incomplete/2. |
| NEW-02 P2 | FIXED | `verify_new02.sh` with PATH shims: 2.44.2, Apple 2.39.5, 1.9.0, option-rejecting Git, garbage version and absent Git all exit 1 with a 2.45.0 requirement diagnostic and 0 Go starts; real Git 2.52.0 through the shim exits 0. `simulate` without Git exits 0 with Golden bytes. Minimum confirmed from installed `RelNotes/2.45.0.adoc` (`git --no-lazy-fetch cmd`). A real Git below 2.45 was not executed (none installed). |
| NEW-03 P3 | FIXED | `verify_new03.sh` (repository without embeds, grep no-match path): before GIT_TRACE=1, GIT_TRACE2=1 and combined TRACE/TRACE2_EVENT/PACKET/SETUP gave incomplete/2; after all complete/0. `GIT_TRACE=<file>` creates no trace file after the fix. Repository config `trace2.normalTarget=2`/`eventTarget=2` complete/0. Real read failures with trace enabled remain incomplete (NEW-01 hostile case). |
| NEW-04 | FIXED | `verify_new04.sh`: `link/../r1.json` (link -> ../real/deep) is written to physical `real/`, Golden bytes, 0600; before wrote lexical `work/r1.json`. Existing physical target: exit 1, bytes kept. Lexical-only existing file: physical file created, lexical bytes kept. `missing/../r4.json`: exit 1, nothing created. Text output, trailing slash rejection, plain `..` and zero temporary remnants checked. Codex reported a separate 22/22 fixture pass; it was not rerun here. |
| NF-04 | KNOWN LIMITATION (deferred P3; P2 operational risk) | Re-measured with current binary: ignored `p/fifo.go` FIFO keeps analyze running beyond 3 s; SIGTERM ends it in 0.232 s with exit 1, and its Go child is gone. Documentation in README/README.ja/ARCHITECTURE is unchanged and matches behavior. |

### Regression: RL-001–RL-010, NF-01, NF-02

All existing tests (284 including the new one) pass under normal, `-race -shuffle=on` and `-shuffle=on -count=5`; these cover RL-001–RL-010 as listed in the historical record. Codex's independent NF-01 and NF-02 scripts were copied with only binary/output paths changed and rerun against the new binary: NF-01 fixed partial-clone runs incomplete/2, upload-pack 0, missing 3 -> 3, fingerprint equal (before binary still shows 2 local upload-pack calls and 3 -> 1); auth and old-Git shims fail closed. NF-02: summary failures=0 (JSON/Text rejection cases, valid Golden 0600 output, 24 processes with exactly 1 success, inode/ACL/xattr preservation).

### Validation commands

| Command | Result |
|---|---|
| gofmt -l . | empty |
| go vet ./...; go vet -stdversion=true ./... | exit 0, no diagnostics |
| go test -count=1 ./... (default macOS TMPDIR) | PASS, 284 tests |
| go test -count=1 -race -shuffle=on ./... | PASS, exit 0, adapters 68.6 s |
| go test -count=5 -shuffle=on ./... | PASS, exit 0, adapters 305.0 s |
| FuzzRead 30 s, parallel 4 | PASS, 3,257,793 execs |
| FuzzPropagation 30 s, parallel 4 | PASS, 750,981 execs |
| per-file `git diff --no-index --check` | 80 untracked project files, 0 issues |
| BenchmarkBaseSourceReading, 1 iteration | 100 files 2.18 s, 1,000 files 20.8 s, 10,000 files 181.6 s |

Determinism: example `simulate --format json` 25 runs, one SHA-256 `809eb4c4…f36a` equal to `testdata/example.report.json`. Codex's seven oracle graphs (chain, diamond, tied100, cycle, components, duplicates, canonical_unicode): baseline, shuffled and repeated inputs each reproduce the expected report SHA-256, and report `input_sha256` equals the normalized Snapshot SHA-256 (chain `0f37c28a…ba60`, report `4dd2c629…9b14`). Analyze on the NEW-01 full and partial clones: 10 runs each, one output hash per repository. Engine, RiskPropagationV1, snapshot/hash, canonical encoder, graph and report sources were not edited.

### Residual risks

- NEW-01 cost: one Git process per base `.go` file (about 18 ms each here). Correct but slow for very large repositories; documented in README. A streamed `cat-file --batch` reader needs its own framing/size/missing-object audit and is a v0.1.1 candidate.
- NEW-02: version parsing trusts the `git version X.Y` banner plus the real option probe; real Git < 2.45 and Linux Git were not executed this session (shims only; Linux 2.47.3 evidence is in the historical record).
- NF-03–NF-07 remain open as documented. NF-04 is not fixed.
- Go 1.23 toolchain and Windows runtime remain NOT VERIFIED.

### Mandala

Goal `Implement and validate RiskLife v0.1.0` preserved. Initial show (this continuation): 21 cells; required gaps adapters.git-compatibility, adapters.nf-01, oss, validation.determinism, validation.fuzz, validation.regression. Mapping: NEW-01/NEW-03 -> adapters.nf-01 (Git read/offline boundary), NEW-02 -> adapters.git-compatibility; added cli.new-04 and cli.nf-04 because the adapters group has no free child slot. Final state is recorded in the user report; `gaps --required --json` is rerun immediately before it.

## Historical record: NF-01/NF-02 final release blockers (2026-10-09 JST)

Verdict (historical): PASS for NF-01/NF-02 remediation in the measured scope. READY FOR FINAL DIFFERENTIAL AUDIT, not release approval. The independent `RiskLife-v0.1.0-final-audit.md` was read from the scratch directory of another local session (`<temporary-directory>/audit/`); its CONDITIONAL PASS was treated as evidence of unresolved findings, not reused as a passing test result.

Environment: Go 1.27.1 darwin/arm64, Apple M1, Git 2.52.0. Linux aarch64 containers used existing local images with `--network none`; Git 2.47.3 was also exercised there. No network access, dependency addition, toolchain installation, commit, push, tag or release occurred. HEAD is still absent and the project files are untracked. Fixture commits exist only in temporary repositories. Mandala state was changed only through installed CLI v0.3.0; hook-owned .omx state was preserved.

Evidence root: `<temporary-directory>/risklife-final`. It contains original source copies, before/after binaries, full command logs, `core/probe-results.json`, and independent NF fixtures. These are temporary and may be removed by OS cleanup. Test commands used `TMPDIR=/private/tmp`, a writable GOCACHE under that evidence root, `GOTOOLCHAIN=local`, `GOPROXY=off`, and `GOSUMDB=off`. The default user build cache was sandbox read-only; the first build attempt failed on cache write and was rerun successfully with the temporary cache.

### Implementation and independent reproduction

Production edits are confined to adapters/command.go, adapters/embed.go, and cli/fileio.go. New tests are git_offline_unix_test.go and output_path_test.go. Engine, RiskPropagationV1, snapshot/hash, canonical encoder, graph and report sources are unchanged. Each modified Go file has at most 143 nonblank/noncomment lines. Review checked shared callers, error unwrapping, child-only environment precedence, original-path validation, and exclusive publication; no new dependency or enum/typing escape hatch was introduced. A separate focused read-only reviewer approved all five Go files with no findings; its targeted tests, race checks and vet also passed. This does not replace the requested final independent differential audit.

| Finding | Regression and separate CLI fixture |
|---|---|
| NF-01 FIXED | New local partial-clone test failed before the fix with upload-pack attempts for both ordinary and inherited `GIT_NO_LAZY_FETCH=0` cases; prompt shim also exposed unsafe inheritance. After fix, tests pass. Independent fixture deletes an embedded source and retired payload between commits: before binary returned complete/0 after 2 local upload-pack calls, missing blobs 3 -> 1, object store changed. Fixed binary returns incomplete/2, `base embed analysis incomplete`, upload-pack 0, missing blobs 3 -> 3, exact missing list and object-store fingerprint unchanged. Hostile parent no-lazy=0/prompt=1 gives identical safe output. Complete ordinary repo returns complete/0 with former_embed classification and no warning. Evidence: `nf01-independent/run.fbtqMo`, script `nf01-independent/verify_nf01.sh`. |
| NF-02 FIXED | New executable test failed before the fix for JSON/Text directory slash/multiple slash and directory-symlink slash. Shared writer now rejects separators before normalization and Lstats the original argument. Independent before binary created existing-dir/existing-dir with exit 0; fixed binary rejects every tested directory/dot/symlink/nonexistent slash case with exit 1 and unchanged tree. New reports/result.json is Golden-identical and 0600. Evidence: `nf02-independent/results.tsv` and `verify_nf02.sh`; portable fixture also ran on macOS and Linux. |

### Git and Go offline boundary

All production Git starts use the common runner: rev-parse, status, diff, ls-tree, ls-files, grep and show. No production cat-file was added. A real delegating shim observed 11 direct Git calls with `--no-lazy-fetch`, plus 7 Go/buildvcs Git calls (including cat-file, status/log/config/for-each-ref). All 18 inherited GIT_NO_LAZY_FETCH=1, GIT_TERMINAL_PROMPT=0 and GIT_OPTIONAL_LOCKS=0; conflicting parent keys were removed, not permanently changed. Optional locks are disabled to avoid unnecessary index refresh writes.

On real Git 2.52.0, environment-only show/grep/cat-file reads of missing blobs failed with exits 128/1/1 and zero upload-pack calls. Git grep can emit a missing-blob diagnostic with exit 1; it is no longer accepted as an ordinary no-match result. Linux Git 2.47.3 environment-only show/grep also failed (128/1), upload-pack 0 and unchanged missing-object list (`linux-git.log`). A simulated Git without `--no-lazy-fetch` support errors at the first direct call, exit 1, 0 Go starts. No claim is made that every older Git supports the environment variable; direct commands require the option and fail closed if it is absent. Auth shim confirms prompt=0 and immediate error/1 within a 5-second guard.

Go acquisition controls from RL-009 are retained and re-executed: disabled proxy/direct VCS/sumdb/toolchain acquisition, saved GOENV/injected flags/workspace suppression, and read-only module listing. Cached dependencies and cached remote replacements remain usable; missing reachable dependencies stay incomplete/error. Go-launched Git inherits the Git environment too. This is acquisition policy, not OS network isolation: installed tools, repository-configured helpers, credential helpers, wrappers and external cache helpers remain trusted. No external endpoint was contacted; partial-clone preparation and measured vulnerable fetches used only local file:// origins.

### Output safety and platform fixtures

JSON/Text each reject existing directory, directory with slash/multiple slashes, directory symlink, dangling symlink, `.`, `./`, `..`, `../`, and nonexistent trailing-slash path. The regression also covers a directory symlink with a trailing separator. No unexpected entry or changed sentinel bytes remain. Valid new-file output equals example.report.json (SHA-256 `809eb4c476c5a45e7d5f764b443fd269aa5cdef0e9865d421d43aff554edf36a`) and has POSIX mode 0600. Existing output/alias/hardlink protection, exclusive os.Link publication, partial-write fault handling and published-cleanup diagnostics remain intact.

macOS independent fixture: 24 actual processes, exactly 1 success and 23 errors/1, full Golden bytes, zero temporary remnants. Existing ACL+xattr file retains inode, mode, bytes, ACL and xattr with error/1. The existing 12-process regression and Darwin deny-read ACL/xattr tests also pass without skips.

Linux aarch64 CLI was cross-built then actually executed in existing network-disabled containers: 20 JSON/Text rejection cases, Golden/0600 new output, 24-process race (1 success, 23 errors), existing inode/bytes/mode and user xattr preservation pass on native tmpfs and overlay. A separate libacl fixture on overlay verifies POSIX ACL and xattr together with inode/bytes unchanged and error/1. Logs: `linux-native-output.log`, `linux-overlay-output.log` (first line is the ACL result). A preliminary host-shared filesystem lacked user xattr support, and tmpfs lacked POSIX ACL support (errno 95 during fixture setup); those are environment limitations, not passed ACL tests or product failures. No packages were installed to resolve them. Unsupported publication filesystems still error without a replacement fallback.

### RL-001 through RL-010 regression

All existing tests were re-executed with current source in normal, race and ten shuffled runs. Earlier PASS records were not substituted for these executions.

| Findings | Re-executed behavior |
|---|---|
| RL-001 | Hidden assume-unchanged/skip-worktree mismatch, executable mode and transformed bytes rejected; index flags retained. |
| RL-002 | Gitlink changes retained despite local/env suppression configuration, including documentation-like names. |
| RL-003 | Active native source ownership; ignored native inputs incomplete; inactive/unknown inputs fail closed. |
| RL-004, RL-008 | Input/output aliases, symlinks, existing entries, exclusive publication, concurrent output, failure stages, ACL/xattr/inode protection. |
| RL-005 | Text control characters escaped; canonical JSON and caller values preserved. |
| RL-006 | Snapshot FIFO/FIFO-symlink rejected without blocking, including executable and preserved-output fixtures. This does not fix analyze's ignored worktree FIFO (NF-04). |
| RL-007, RL-010 | Cached remote replace complete; reachable local/missing replacements incomplete; unused replacements do not alone affect completeness. |
| RL-009 | Hostile private/proxy/VCS settings and saved Go configuration cannot cause dependency acquisition; target/cache settings retained. |

### Determinism and Golden output

Existing tests pass for 200 node/edge/seed shuffles, 200 report metadata/cause/cell/target/wave shuffles, and 25 repetitions. The inspected standalone oracle was rerun against the new CLI with fresh outputs: seven graphs (chain, diamond, 100 ties, cycle, components, duplicates, Unicode), each 200 shuffles and 25 repeats: 1,400 shuffles, 175 repeats. Independent normalized-input SHA-256, canonical JSON bytes, complete Trace, Generation and Test Targets agree; 31 additional invalid/status probes also pass. `core/probe-results.json` records expected and actual values, and `core-oracle.log` records totals. Existing Golden tests and the executable example Golden comparison pass.

Chain Snapshot SHA-256: `0f37c28a7d4f57d88b88d741493228f696ae57f4acc9e297ed5269838ea6ba60`. Chain report SHA-256: `4dd2c6298b608dee59a5a5c5b045a5b2aea2965c1579b332d0b9ed9979659b14`. Chain/diamond/ties/cycle use 4 generations; the other oracle graphs use 3. RiskPropagationV1's formula was not edited.

### Full validation

| Command | Actual result |
|---|---|
| gofmt -l . | PASS, empty output |
| go test ./... | PASS, exit 0; adapters 62.160 s, CLI 5.054 s |
| go test -race ./... | PASS, exit 0; adapters 76.453 s, CLI 9.066 s; no races |
| go test -shuffle=on -count=10 ./... | PASS, exit 0; adapters 543.144 s, CLI 83.374 s |
| go vet ./...; go vet -stdversion=true ./... | PASS, exit 0; no diagnostics, latter is static compatibility only |
| temporary CLI build | PASS; before and after binaries driven through independent fixtures |
| git diff --check | PASS; HEAD absent, so additionally per-file no-index --check covered all 76 untracked project files without issues |
| FuzzRead, 30s, parallel=4 | PASS, exit 0; 463,844 executions, 31.402 s package time |
| FuzzPropagation, 30s, parallel=4 | PASS, exit 0; 153,217 executions, 31.411 s package time |
| BenchmarkPropagation, 100/1,000/10,000 nodes, single/multiple seeds | PASS, exit 0; 300ms, 3 samples; benchmark.log |

Core benchmark medians (single / multiple seeds): 100 nodes 78.990 / 81.108 microseconds; 1,000 nodes 0.750 / 0.774 ms; 10,000 nodes 8.608 / 10.717 ms. These finite samples overlapped other validation; they are not a before/after performance claim. No new lint/LSP dependency was installed; compiler, vet and race checks provide the recorded evidence.

### Remaining P3 and release decision

All five are OPEN, deferred as v0.1.1 candidates; no broad P3 implementation was included.

The read-only P3 evaluator retained fixtures under `<temporary-directory>/risklife-p3/{nf03,nf04,nf06,nf07}`. Their measured output was captured in this conversation, not independent log files; NF-05 was checked with the existing fault-injection test.

| Finding | Reproduction, impact, workaround and candidate |
|---|---|
| NF-03 | Tracked unrelated directory symlink reproduced error/1; dangling does likewise, regular-target symlinks are incomplete. Implementation over-applies HEAD input validation, a fail-closed availability defect. Use ordinary files in an analysis checkout. Candidate: verify symlink text against blob, restrict unsupported-input warnings to real Go inputs. |
| NF-04 | Ignored local *.go FIFO blocks RiskLife and bare go list; each bounded independent probe timed out at 3 s. SIGTERM ended collection in 0.106 s with no process-group residue. Local DoS is real; no internal deadline. Remove special files and use an external timeout for unattended mutable worktrees. Candidate: WalkDir/Lstat preflight of Go/native candidates. Not elevated to a release blocker here: FIFO cannot be supplied by Git clone, separate local creation is needed, cancellation recovers, and no data loss/false-complete was observed. This does not declare arbitrary local worktrees safe. |
| NF-05 | Existing cleanup fault-injection test passes: full output is published, residual temporary hardlink and explicit error/1 are possible. Documented safety-side design limit. Inspect the published result and remove residual link after fixing permissions. Candidate: typed publication result and revised warning/exit contract. |
| NF-06 | 250-byte basename reproduced error/1 during longer temporary-name creation; output absent. Availability defect. Use shorter names. Candidate: fixed short temporary pattern. |
| NF-07 | Tracked testdata/sub/go.mod reproduced incomplete/2 despite Go wildcard exclusion; false-incomplete availability defect. Accept the conservative restriction or rename/untrack fixture module in the analysis checkout. Candidate: match Go's vendor/testdata/dot/underscore path exclusion rules. |

### Compatibility and Mandala

Go 1.23 actual toolchain: NOT VERIFIED; PATH/SDK/Homebrew/toolchain-cache inspection found none, and none was installed. Static stdversion vet passes. macOS arm64 runtime: verified. Linux arm64 CLI cross-build and the output/Git fixtures described above: verified; a full Linux Go-analysis suite was not run (container has no Go). Windows amd64 cross-build: PASS; runtime NOT VERIFIED. Git suppression tested on macOS 2.52.0 and Linux 2.47.3, older-option rejection by shim only.

Mandala preserves Goal `Implement and validate RiskLife v0.1.0`. The fresh initial show confirmed done=15, open=0. Added adapters.nf-01 and cli.nf-02; reopened adapters.rl-009, cli.rl-008, oss and three validation leaves. Finding leaves cover implementation, regression, independent fixtures and docs together and were completed only after all four passed. Final CLI status: 20 cells, 3 expanded groups, 17 required leaves; done=17, open=0, na=0, optional=0. Required gaps: 0; `gaps --required --json` returned `{"schema_version":1,"gaps":[]}` and exit 0. The command is repeated immediately before the final user report. No declared required gaps remain. This is separate from test quality and is not proof of exhaustive coverage or correctness. No clean, reset, direct state edit or NA workaround was used.

## Historical record: second audit remediation (2026-10-09 JST)

The independent audit `<temporary-directory>/risklife-final-audit/AUDIT.md` was read before editing. Its FAIL / NOT READY result is the baseline. This record is remediation evidence, not final independent release approval; the earlier record below is historical.

Environment: Go 1.27.1, darwin/arm64, CGO_ENABLED=1, Apple M1 (benchmark CPU output), Git 2.52.0. Repository HEAD remains absent; source remains untracked. No commit, push, tag, release, new dependency or toolchain installation occurred. Fixture commits exist only in temporary repositories. The existing ignored binary and unrelated files were preserved. Hook-owned .omx logs/state changed automatically, not through manual edits. Source SHA-256 manifest and probes/logs are in `<temporary-directory>/risklife-second-remediation`; these temporary artifacts may disappear under OS cleanup.

Test environment: physical TMPDIR/GOTMPDIR and GOCACHE under that directory; GOPROXY=off, GONOPROXY=none, GOVCS=*:off, GOSUMDB=off, GOTOOLCHAIN=local, GOWORK=off, GOENV=off, empty GOFLAGS/GOEXPERIMENT. An initial baseline HEAD-helper test failed because macOS /var versus /private/var temporary paths were treated as a symlink path. Setting physical /private/tmp fixture paths resolved the test-environment mismatch without weakening HEAD validation.

### Findings: implementation, regression, separate fixtures

| Finding | Fix | Measured evidence |
|---|---|---|
| RL-009 | Shared child environment prevents inherited private/proxy/VCS rules and saved Go settings/injected flags from changing acquisition policy. Explicit target/cache settings remain; unsupported parent flags still warn/incomplete. | Before: new private-module test intercepted 3 ls-remote calls. After: 0, PASS. Effective environment, flags and GOENV tests PASS. Independent CLI fixture with a different module/version: missing private dependency and hostile flags/saved GOENV exit 2; cached ordinary module and cached remote replace exit 0. All four have 0 VCS calls and four policy-controlled Go children. |
| RL-008 | Reject every existing --out entry; publish fully written/synced/closed new 0600 temporary inode with exclusive os.Link. No rename fallback. Post-publication cleanup errors explicitly report published output. | New rejection test failed before the fix (exit 0, old bytes replaced), passes after it. Darwin deny-read ACL and xattr tests PASS without SKIP. Create/write/chmod/sync/close/publish and pre/post-publication cleanup faults PASS. Destination races covering regular/symlink/dangling/hardlink/directory PASS. Regression: 12 processes, exactly 1 success. Independent CLI: 24 processes, exactly 1 success, complete Golden bytes, no temporary names. Existing bytes, inode, mode, owner/group, ACL and xattr unchanged. |
| RL-010 | Documentation limits replacement completeness to packages reached by current ./... dependency/test collection; replacement algorithm unchanged. | Six new cases and existing cached replacement tests PASS. Independent CLI: active local, missing active local and missing remote dependency incomplete/2; unused existing local, unused missing local and unused remote complete/0 without warnings. Cached active remote replace complete/0 without local warning. |

Production changes are only command.go (environment), go.go (parent-flag diagnostic) and cli/fileio.go (output lifecycle). Engine, RiskPropagationV1, normalization, hash and canonical encoder sources are unchanged. Installed cmd/go/os implementation and official [Go command](https://pkg.go.dev/cmd/go), [Modules](https://go.dev/ref/mod), [os.Link](https://pkg.go.dev/os#Link) contracts were inspected. No runtime Mandala dependency was introduced.

### Combined / previous findings

Offline + cached remote replacement: complete/0, 0 VCS calls. Offline + missing dependency: incomplete/2, 0 calls. Offline + hidden HEAD mismatch: Git status empty after assume-unchanged mutation, CLI error/1 with no stdout and 0 VCS calls. ACL + existing/alias outputs retain bytes and metadata. FIFO/FIFO-symlink input with existing output is rejected within a 2-second deadline without changing output. New-file JSON equals the existing Golden bytes. Replacement combinations retain the statuses above.

Original earlier fixtures were copied and replayed with the new binary: all 10 assertions pass. RL-001 hidden import rejected; RL-002 suppressed gitlink change incomplete; RL-003 ignored assembly incomplete and committed assembly native/complete; RL-004 same-input output refused without changes; RL-005 controls escaped; RL-006 FIFO refused; RL-007 cached remote replacement complete; both Golden JSON files identical. All existing regressions also run in the full normal/race/shuffle suites.

### Required commands

| Command | Result |
|---|---|
| gofmt -l . | PASS, empty output |
| go test ./... | PASS, exit 0; adapters 79.201 s, CLI 14.076 s |
| go test -race ./... | PASS, exit 0; adapters 70.109 s, CLI 9.662 s, no races |
| go test -shuffle=on -count=10 ./... | PASS, exit 0; adapters 470.231 s, CLI 76.272 s |
| go vet ./... | PASS, exit 0, no diagnostics |
| go vet -stdversion=true ./... | PASS, static compatibility only |
| go build -o temporary/risklife ./cmd/risklife | PASS; binary exercised by independent fixtures |
| git diff --check | PASS; untracked sources additionally checked with per-file no-index --check (74 Go/docs/module/license/fixture files) |
| FuzzRead, -fuzztime=10s -parallel=2 | PASS; 11.883 s package time, 29,453 executions, 2 workers, no failing corpus |
| FuzzPropagation, -fuzztime=10s -parallel=2 | PASS; 11.634 s package time, 24,333 executions, 2 workers, no failing corpus |

Fuzz commands: go test ./internal/snapshot -run '^$' -fuzz '^FuzzRead$' -fuzztime=10s -parallel=2 and equivalent ./internal/rules/risk / FuzzPropagation. Normal/race/shuffle runs overlapped; fuzz overlapped cross-compilation. Counts are coverage exploration, not throughput or exhaustive proof. gopls is unavailable and installation was previously declined; compiler/vet/race were used. No extra lint dependencies were installed.

### Determinism

The previous standalone Python oracle was copied and pointed at the new binary. It implements normalization, SHA-256, synchronous arithmetic, trace and canonical JSON without RiskLife internals. Seven graphs (chain, diamond, 100 ties, cycle, components, duplicates, Unicode) each passed 200 shuffles and 25 repetitions: 1,400 shuffles / 175 repeats. Hash, JSON bytes, Trace, Generation and Test Targets match independent expectations. Thirty-one invalid-input/status probes also pass. Generation count: 4 for chain/diamond/ties/cycle; 3 for the other graphs. TestJSONMetadataAndTiedCausesAreOrderIndependent adds 200 shuffles of warnings, unknown changes, causes, cells, targets and waves. Existing node/edge/seed and 25-repeat tests pass.

Chain Snapshot SHA-256: 0f37c28a7d4f57d88b88d741493228f696ae57f4acc9e297ed5269838ea6ba60. Chain report SHA-256: 4dd2c6298b608dee59a5a5c5b045a5b2aea2965c1579b332d0b9ed9979659b14. Full hashes/traces are in core/probe-results.json under the evidence directory. Independent --out example Golden SHA-256: 809eb4c476c5a45e7d5f764b443fd269aa5cdef0e9865d421d43aff554edf36a.

### Benchmarks: before / after

Same Go/CPU, sequential runs, 3 samples each; medians below. Before uses the preserved audit source copy; after uses current source. Core commands: go test ./internal/rules/risk -run '^$' -bench '^BenchmarkPropagation$' -benchmem -benchtime=300ms -count=3. HEAD command uses ./internal/adapters, BenchmarkHeadVerification and -benchtime=200ms. Core and HEAD production sources are unchanged; these finite samples include timing noise and do not establish a statistically significant improvement/regression.

| Size | Core single seed before → after | Core multiple seeds before → after | HEAD two passes before → after |
|---|---|---|---|
| 100 | 63.971 → 69.987 µs | 70.424 → 70.893 µs | 14.258 → 13.710 ms |
| 1,000 | 0.707 → 0.660 ms | 0.743 → 0.751 ms | 0.255 → 0.251 s |
| 10,000 | 8.336 → 8.649 ms | 9.074 → 9.157 ms | 2.623 → 2.541 s |

Core includes normalization/hash/propagation/report model, excludes final JSON and adapters; N-1 chain edges, 3 updates. HEAD measures two scans of N × 1 KiB tracked files, excluding tree/go-list/fixture preparation. At 10,000 files current cumulative allocation is about 85.7 MB/op, not peak RSS. Input bytes/count/path work and extra verification passes impose real costs; offline and output guarantees were not relaxed for performance.

### Compatibility, review and limitations

Darwin arm64 runtime, ACL/xattr, concurrent output and FIFO tested. Linux amd64 and Windows amd64 CLI cross-builds PASS (CGO_ENABLED=0); runtime behavior is NOT VERIFIED. os.Link maps to Unix link / Windows CreateHardLink without an unsafe fallback. Go 1.23 runtime is NOT VERIFIED: PATH/SDK/Homebrew/toolchain-cache checks found no installed 1.23, and nothing was installed. New APIs fit Go 1.23; stdversion vet passes.

Go acquisition controls are not OS network isolation; installed tools/wrappers/external cache helpers are trusted. HEAD scans are not transactional against same-inode mutation/reversion, HEAD ABA or changes after final scan. Output requires stable trusted directory/temp names; persistent parent replacement is detected, but malicious path substitution/ABA after checks is not fully excluded. The final hardlink operation cannot replace an existing entry. Unsupported/network filesystems, adversarial directories and crash durability are unverified. Directory moves, cleanup failure or crashes can leave a temporary name. Post-publication failure explicitly reports the complete output. 0600 is a POSIX-mode promise; new files may inherit directory ACLs. Active directory replacements, workspaces, nested modules and custom build conditions retain conservative limits.

A fresh read-only pre-patch investigator independently reproduced RL-009 and checked boundaries/compatibility. The post-patch review agent failed before review because the service reported a usage limit. A separate parent review pass checked final diff, callers, OS Link implementation, hostile flags and legitimate cached workflows. This is not a completed third-party post-fix review. Independent re-audit remains required before release.

### Mandala

Installed Skill and CLI contract were read; CLI v0.3.0, installed outside `PATH` in the user's Go binary directory. Skill file has no embedded version, so v0.4.0 is not asserted. Original Goal Implement and validate RiskLife v0.1.0 and 12 leaves were inspected and preserved. cli.rl-004, oss and three validation leaves were reopened. Only adapters.rl-009, cli.rl-008 and adapters.rl-010 were added, giving 15 required leaves. Unrelated done leaves remained unchanged. No clean, reset, direct state edit, deletion, NA or capacity workaround occurred. Each finding was marked done only after implementation, regression, separate fixture and documentation evidence above. Final CLI status: 18 cells, 3 expanded groups, 15 required leaves; done=15, open=0, na=0. The required-gaps command returned {"schema_version":1,"gaps":[]} with exit 0 in this completion turn, and is repeated immediately before the final user report. No declared required gaps remain. This result does not prove completeness of the declarations, correctness or security.

Remediation verdict: PASS for RL-008–RL-010 and the recorded checks. READY FOR INDEPENDENT RE-AUDIT. No reproduced release blocker remains in the tested scope. Go 1.23 and Linux/Windows runtime verification, and a completed independent post-fix audit, remain prerequisites to any broader release claim. No release was approved or published.

## Historical record: first remediation (RL-001–RL-007)

Recorded on 2026-10-08–09 (Asia/Tokyo). This section replaces the earlier implementation-only validation record. The seven confirmed audit defects were reproduced before editing; passing baseline tests were not treated as defect-remediation evidence. The existing audit report and its original fixtures were read and replayed locally.

## Environment and scope

Go go1.27.1, darwin/arm64, CGO_ENABLED=1, Apple M1; Git 2.52.0. Module: github.com/cottondesu/risklife, go directive 1.23. No Go 1.23 toolchain was found in PATH, the usual SDK/Homebrew installation locations or the existing module toolchain cache. Static review found no newly used API/language feature requiring later than 1.23; actual Go 1.23 execution is NOT VERIFIED. Windows and illumos file-opening code was cross-compiled, not executed.

Tests used unique temporary cache/module-cache/build directories, GOPROXY=off, GOSUMDB=off, GOTOOLCHAIN=local and GOWORK=off. Replacement fixtures prewarm an isolated cache from a temporary file:// proxy; no external network or new package installation was used. No runtime Mandala dependency was added. gopls was unavailable; compiler, race and vet checks were used. No commit, push, tag or release was created.

The project checkout still has no HEAD and its sources remain untracked. Existing .omx content was preserved. Temporary integration repositories contain their own fixture commits only. Git diff --check therefore cannot validate the untracked source tree; per-file no-index whitespace checks supplement it.

## Regression evidence

Each row has an executable regression fixture; the defects were observed on the original implementation, and the remediation tests now pass. The original audit fixtures were also run with the new compiled CLI.

| Finding | Original result | Remediated result | Regression tests |
|---|---|---|---|
| RL-001 | assume-unchanged import removal returned complete with a graph different from HEAD | Original CLI fixture exits 1 with no JSON; byte/mode mismatch is explicit | TestAnalyzeRejectsHiddenHeadMismatch (assume-unchanged, skip-worktree, flag preservation), TestAnalyzeRejectsHiddenExecutableModeMismatch, TestAnalyzeRejectsTransformedWorktreeBytes; existing staged/unstaged/untracked/ignored/symlink cases |
| RL-002 | diff.ignoreSubmodules=all hid a changed gitlink and returned complete with empty changes | Original CLI fixture exits 2, unknown changed gitlink retained | TestAnalyzeGitlinkCannotBeHidden: default, local config, environment config and command override; gitlink named dep.md cannot be excluded as documentation |
| RL-003 | ignored active assembly returned complete; committed native inputs were unowned | Original ignored assembly exits 2; original committed assembly exits 0 with native classification and package seed 3 | TestAnalyzeNativeSourceChanges (SFiles, HFiles, SysoFiles), TestAnalyzeHiddenAssemblyChanges, TestAnalyzeCgoHeaderChanges, TestAnalyzeUnknownNativeInputIsIncomplete, TestNativeInputsAllHaveProductionOwnership |
| RL-004 | same path/hardlink/input symlink/output symlink overwrote input or referenced data | All aliases/symlinks rejected with exit 1 and original bytes preserved; ordinary output succeeds | TestSimulateRejectsOutputThatAliasesInput, TestSimulateRejectsInputSymlinkAliasAsOutput, TestSimulateRejectsOutputSymlinkWithoutChangingTarget, TestSimulateAtomicallyReplacesExistingOutputAndPreservesPermissions, temporary-write/cleanup failure and moved-input identity tests |
| RL-005 | raw newline/ESC allowed forged Status lines and terminal commands | Original malicious Snapshot emits visible escapes with one real Status line | text_security_test.go: all Text fields, CR/LF/TAB/ESC/control/Unicode/ordinary strings, JSON byte equality and caller immutability |
| RL-006 | FIFO and FIFO-symlink blocked without a writer; original SIGTERM case needed SIGKILL | Original FIFO exits 1 immediately; bounded binary tests reject both FIFO forms without changing preexisting output | TestExecutableRejectsFIFOInputWithoutBlocking, directory/missing/malformed/regular input tests, shared regular-file tests |
| RL-007 | cached versioned remote replacement was labeled local and returned incomplete | Original cached replacement exits 0 complete with no local-replacement warning | TestAnalyzeCachedVersionedReplacement, TestAnalyzeUnavailableReplacements (missing local/remote), existing local/no-replace tests |

The native-field ownership unit test covers SFiles, CFiles, HFiles, CXXFiles, MFiles, FFiles, SwigFiles, SwigCXXFiles and SysoFiles. Real Go-list fixtures cover active assembly/header/system-object ownership and CGO_ENABLED=1 header changes; these are metadata/analysis tests, not successful assembly/object-link builds. Other native compiler configurations were not executed. Current-target exclusions and unknown ownership remain incomplete.

Combination coverage: hidden assembly plus HEAD verification (RL-001+003); suppression configuration plus hidden assembly (TestSuppressionSettingsCannotHideAssemblyMismatch, RL-001+002); FIFO/symlink input with preexisting output and timeout (RL-004+006); escaped Text with unchanged canonical JSON (RL-005+JSON); cached remote replacement complete while missing replacements remain incomplete (RL-007+Go Adapter).

Additional review caught a permission regression in the first atomic-output implementation: chmod 0644 bypassed umask 077. A subprocess regression failed with mode 0644 before the refinement and passes with private mode 0600. Existing output permission bits are preserved. Cleanup failures are joined with the primary write failure rather than discarded.

## Required checks

The baseline go test ./... passed before source/test edits. Existing tests were not removed, skipped to hide failures or weakened.

- go test ./...: PASS after all source changes (adapters 40.326 s, CLI 5.043 s). The earlier full post-remediation run also passed (adapters 36.679 s, CLI 3.274 s).
- go test -race ./...: PASS (adapters 56.546 s, CLI 6.116 s); final CLI permission/test-file refinements have focused rechecks.
- go test -shuffle=on -count=10 ./...: PASS (adapters 430.677 s, CLI 49.738 s). After the final CLI permission refinement and test-file move, CLI/fileio race and shuffle-count=10 rechecks also passed (CLI shuffle 50.600 s).
- After the final shared-buffer change and large-blob regression, go test -race -shuffle=on -count=10 -timeout=15m ./internal/adapters: PASS (493.610 s). This checks every adapter test on the final source with both race and shuffle enabled.
- go vet ./...: PASS after all source changes.
- go build -o <unique temporary path>/risklife ./cmd/risklife: PASS after all source changes.
- gofmt -l .: PASS, no output after all source changes.
- git diff --check: PASS, limited to tracked content (no project HEAD).
- Per-file git diff --no-index --check /dev/null FILE: PASS across all 65 project Go/Markdown/JSON/module files, including untracked additions; repeated after final documentation updates.

CLI integration tests exercise complete=0, incomplete=2 and error=1, valid output, no diagnostics in JSON stdout, dirty/error rejection and test-only selection. Original-fixture replay validates the changed classifications/statuses through the compiled binary.

## Determinism and core preservation

Existing fixed-seed tests compare all report fields and bytes over 200 node/edge/seed-order shuffles and 25 repeated runs. A separate actual-binary check performed 200 shuffles and 25 repeats of the example Snapshot: canonical JSON bytes, SHA-256, waves/causes and generation count all matched the baseline and checked-in example.report.json.

Observed input SHA-256: e0c96ca23a1e63430c810825ffd4161ecb4a26c4835398cad2e0b66b072acde8.
Observed report SHA-256: 809eb4c476c5a45e7d5f764b443fd269aa5cdef0e9865d421d43aff554edf36a.
Generation count: 4. Independently checked package and own-test risks: A=3, B=2, C=1, D=0.

No engine, RiskPropagationV1 arithmetic, graph normalization, Snapshot hashing or canonical JSON encoder production code changed. Existing synchronous-update, fixed-point/boundary, tied-cause, cycle, randomized-reference/property and generic boolean-state tests remain successful. JSON golden bytes are unchanged; native is an added analysis classification, not a schema-shape change. This evidence does not establish real-project bug detection accuracy.

## Fuzz evidence

Commands: go test ./internal/snapshot -run '^$' -fuzz '^FuzzRead$' -fuzztime=10s -parallel=2 and the equivalent risk command with FuzzPropagation.

| Target | Budget | Observed elapsed | Executions | Result |
|---|---|---|---|---|
| FuzzRead | 10 s, 2 workers | 11.301 s | 166,122 | PASS |
| FuzzPropagation | 10 s, 2 workers | 11.303 s | 27,646 | PASS |

Fuzzing overlapped other functional checks; these counts are measured executions, not exhaustive proof or throughput benchmarks.

## Performance

Command: go test ./internal/rules/risk -run '^$' -bench . -benchmem -benchtime=200ms -count=3. Each value below is a median of three samples. The sparse graph is a chain with one seed or a seed every ten nodes; every case reports three updates. Risk benchmarks include normalization, hashing, propagation, traces and report-model construction, but exclude JSON serialization and Git/Go collection.

| Nodes | Seeds | Before ns/op | After ns/op | Before/after B/op | Before/after allocs/op |
|---|---|---|---|---|---|
| 100 | one | 82,200 | 67,299 | 120,517 / 120,309 | 187 / 187 |
| 100 | every 10 | 71,619 | 72,220 | 134,690 / 134,560 | 337 / 337 |
| 1,000 | one | 664,835 | 699,053 | 1,352,837 / 1,356,593 | 1,111 / 1,111 |
| 1,000 | every 10 | 748,070 | 773,047 | 1,556,918 / 1,558,864 | 2,540 / 2,540 |
| 10,000 | one | 8,612,663 | 9,686,663 | 18,311,656 / 18,311,657 | 10,295 / 10,295 |
| 10,000 | every 10 | 11,317,447 | 11,192,721 | 21,064,353 / 21,064,374 | 24,338 / 24,338 |

Risk execution source is unchanged and allocation counts remain equal. Timing changes range from -18.1% to +12.5%; three samples and host contention/variation do not establish a statistical regression or improvement. Some early work overlapped the before measurement. After measurements ran after functional workers finished. No substantial risk-engine allocation regression was observed; more samples on controlled hardware are needed for tight timing guarantees.

Additional command: go test ./internal/adapters -run '^$' -bench '^BenchmarkHeadVerification$' -benchmem -benchtime=200ms -count=3. This measures only the new two-pass raw HEAD verification on N tracked regular files, each 1 KiB, with initialized fixture data outside the timer. It excludes Git tree collection, go list and report/JSON construction. No claim of a measured end-to-end adapter before/after comparison is made.

| Files | Initial scan ms/op | Final scan ms/op | Initial/final B/op | Final allocs/op |
|---|---|---|---|---|
| 100 | 14.694 | 24.013 | 7,327,413 / 841,868 | 7,803 |
| 1,000 | 268.937 | 259.514 | 73,339,168 / 7,892,064 | 78,014 |
| 10,000 | 3,446.399 | 2,701.847 | 733,762,248 / 78,661,472 | 780,136 |

The initial scan allocated a 32 KiB io.Copy buffer per file. Final code reuses one 32 KiB buffer per pass, reducing total 10,000-file allocation by about 89%. B/op is cumulative allocation, not peak resident memory. Path resolution/stat/open/hash bookkeeping still allocates; HEAD verification is a real additional cost proportional to file count, path depth and bytes. The copy is bounded to opened size plus one byte and detects a same-size mutation beyond multiple buffer lengths in head_test.go. Smaller-case timing variability remains visible and is not hidden.

## Mandala

CLI mandala v0.3.0. The installed Mandala Skill and CLI contract were read. Existing goal Implement and validate RiskLife v0.1.0 was preserved. Existing adapters, cli, validation and oss declarations were reopened; core stayed done. Required children were added for four adapter findings, three CLI findings and regression/determinism/fuzz validation. No direct state edit, clean or goal replacement occurred.

After all required checks, final status reports 15 cells, 3 expanded groups and 12 required leaves: done=12, open=0, na=0. The required-gaps command returned {"schema_version":1,"gaps":[]} with exit 0. It is repeated immediately before the final user report as required by the Skill contract. Mandala gap counts are declaration tracking, not proof of correctness.

## Remaining limits and readiness

Raw HEAD bytes must match; clean/smudge filters or line-ending transformations are rejected. Symlinks, workspaces, nested modules, directory replacements, custom build flags and unowned/deleted/inactive inputs have conservative limits. Analysis covers current go list build inputs, not all platforms or runtime dependencies, external/system headers or successful native compilation. The HEAD scans add work proportional to tracked file count and bytes.

Stable files are required during analysis/output: mutation and reversion between scans, same-inode concurrent modification, parent-directory changes and the final rename race are not eliminated. Unix nonblocking input opening bounds FIFO behavior; the non-Unix fallback retains a pre-check/open race. Directory crash durability is not guaranteed; Windows replacement can fail safely. Runtime execution on Go 1.23, Windows, Linux and other native toolchains is NOT VERIFIED.

HEAD hashing checks context between files, so a very large regular file can delay cancellation during its read. The current worktree size bounds the copy, not the HEAD blob size; hidden oversized/sparse input can still incur substantial I/O before hash rejection. No repository-wide byte budget or end-to-end large-repository adapter performance guarantee is claimed.

Release approval requires independent re-audit and identification of the approved release artifact. This remediation does not publish or authorize a release.

Remediation verdict: PASS for the seven reproduced findings and the checks described above. Readiness: READY FOR INDEPENDENT RE-AUDIT. A fresh read-only code review found no remaining blocker after the private-output-permission refinement and confirmed that shared-buffer hashing preserves ordinary-file semantics. This is remediation evidence, not independent final release approval.
