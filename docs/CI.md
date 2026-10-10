# Continuous integration

This document describes the GitHub Actions workflow of StateRipple v0.1.0, what it verifies and what it does not, and the gates that must pass before a first release.

**Status (2026-10-10).** The workflow and its helper scripts were written and checked locally (see [VALIDATION.md](VALIDATION.md)), and the workflow has since run green on GitHub Actions on all three of its trigger paths — `push`, `workflow_dispatch` and `pull_request`:

| Run | Event | Result |
|---|---|---|
| [38026796484](https://github.com/cottondesu/stateripple/actions/runs/38026796484) | `push` to `main` (initial commit 31cf6e7) | every job green in 91 s wall; `long-shuffle` skipped by design |
| [38027059887](https://github.com/cottondesu/stateripple/actions/runs/38027059887) | `workflow_dispatch` with `long_shuffle` (same commit) | every job green in 2 min wall; `long-shuffle` passed in about 2 min |
| [38027821273](https://github.com/cottondesu/stateripple/actions/runs/38027821273) | `pull_request` (PR #1, branch `docs/verify-pr-ci-trigger`, base `main`) | every job green in 80 s wall; `long-shuffle` skipped by design |
| [38028313271](https://github.com/cottondesu/stateripple/actions/runs/38028313271) | `pull_request` (same PR, commit a88ff5b) | every job green in 93 s wall; `long-shuffle` skipped by design |

The hosted runner was `ubuntu-24.04` (Linux 6.17.0-1022-azure x86_64) with Git 2.55.0. PR #1 has completed two successful CI runs (38027821273 and 38028313271). They executed sequentially — the first completed before the second began, without an overlapping in-progress run — so no run was ever cancelled by a newer one, and PR-scoped concurrency cancellation remains not verified (see [Not verified yet](#not-verified-yet)).

CI is verification only. It publishes nothing: no release, tag, package, container image, artifact upload or write to the repository. A release is a separate, manual step.

## Workflow

`.github/workflows/ci.yml`, name `CI`.

| Trigger | Condition |
|---|---|
| `push` | branch `main` |
| `pull_request` | target `main` (no `pull_request_target`, no `workflow_run`, no schedule) |
| `workflow_dispatch` | manual; input `long_shuffle` (boolean, default false) adds the long shuffle job |

| Job | Runner | Go | Purpose | Job timeout |
|---|---|---|---|---|
| `compat` (matrix) | `ubuntu-24.04` | 1.23, 1.26, 1.27 (one job each, `fail-fast: false`) | gofmt, `go vet`, CLI build, `go test` | 20 min |
| `enhanced` | `ubuntu-24.04` | 1.27 | strict offline environment, race detector, shuffled order, named regression groups | 45 min |
| `linux-binary` | `ubuntu-24.04` | 1.27 | built static binary: Git/Go-free and offline acceptance, then a real `analyze`, each inside an empty network namespace | 20 min |
| `long-shuffle` | `ubuntu-24.04` | 1.27 | `-shuffle=on -count=10`; only for `workflow_dispatch` with `long_shuffle` | 90 min |
| `ci-ok` | `ubuntu-24.04` | none | one status for branch protection: fails unless `compat`, `enhanced` and `linux-binary` succeeded | 5 min |

The jobs do not depend on each other (`ci-ok` only reads their results), so a failure in one does not hide the others.

### Security configuration

- `permissions: contents: read` for the whole workflow; no job widens it. No secret is referenced and the workflow never names `GITHUB_TOKEN`; the default token is only what `actions/checkout` and `actions/setup-go` receive on their own.
- `actions/checkout` runs with `persist-credentials: false`, so no credential stays in `.git/config` for later steps.
- Only two actions are used, both official and pinned to a full commit SHA with the release in a comment: `actions/checkout@3d3c42e5…ba90b1 # v7.0.1` and `actions/setup-go@b7ad1dad…4303e # v7.0.0` (SHAs resolved with the GitHub API on 2026-10-10: the `v7.0.1` / `v7.0.0` tags, which were the latest releases, and the moving `v7` tags point to the same commits). `setup-go` runs with `cache: false`, so no build cache is restored or saved.
- No third-party action, no `curl | sh`, no `pull_request_target`/`workflow_run`, no expression from untrusted event data in a script (the only expression inside a `run:` is the static matrix value).
- Concurrency: a new push cancels the previous run of the same pull request only; runs on `main` are never cancelled.
- `GOTOOLCHAIN=local` is set for the whole workflow: the `go` command never switches to or downloads another toolchain. `setup-go` downloading the official Go toolchain is CI preparation and is distinct from StateRipple's runtime, which must not use the network.
- Every `run:` step uses `bash -eo pipefail`. No step uses `|| true`, `continue-on-error` or an unconditional skip.

To update an action: resolve the new release tag to its commit (`gh api repos/actions/<name>/commits/<tag> --jq .sha`), change the SHA and the comment together, and run the workflow. The repository setting "Require actions to be pinned to a full-length commit SHA" can enforce this; it is currently off on the GitHub repository.

## Go versions

| Go | Role |
|---|---|
| 1.26, 1.27 | Supported Go (Go supports the two latest releases). Security-supported. |
| 1.23 | The minimum version StateRipple builds and tests with (`go 1.23` in `go.mod`). Out of Go's security support: it is a backward-compatibility check, not a recommendation to run on it. |

All three run the real toolchain from `actions/setup-go`, which resolves a patch release of the requested minor version — 1.23.12, 1.26.8 and 1.27.2 in the recorded hosted runs (the local measurements in [VALIDATION.md](VALIDATION.md) used 1.26.9, so the resolved patch is not always the newest). The workflow pins only the minor version and does not guarantee the latest patch. The job prints `go version` and `go env GOVERSION GOTOOLCHAIN` and fails if the toolchain is not the requested minor version or `GOTOOLCHAIN` is not `local`. If Go 1.23 fails to build or test, that is a failure of the minimum-version promise and is reported as such; tests are never skipped to make it green.

## Git 2.45 or later

`analyze` needs Git 2.45.0+ with the global `--no-lazy-fetch` option. Every job that runs Git-dependent tests starts with `scripts/ci/require-git.sh`:

1. runs `git --version` and prints it;
2. parses major and minor as numbers (never as a string, so `2.9` is older than `2.45`), and fails below 2.45;
3. probes `git --no-lazy-fetch version`.

`scripts/ci/test-require-git.sh` runs the gate against stand-in `git` executables (2.44.2, 2.9.9, 2.5.0, 1.99.0, Apple Git, a garbled banner, an empty banner, no Git, a Git that rejects the option, and 2.45.0, 2.52.0, 2.100.0, 3.0.0 as passes). The version the hosted runner has is not assumed: the gate decides on the real job.

Tests that need Git fail, never skip, when Git is too old. The suite itself asserts this (`TestAnalyzeRejectsUnsupportedGitBeforeGoCollection`), and the `gotest.sh` wrapper below turns any skip into a failure.

## Test commands

`scripts/ci/gotest.sh <label> <go test arguments>` runs `go test -json` and fails when a test fails, when the build fails, when any test was skipped, when no test passed (a `-run` pattern that matches nothing is green in plain `go test`), or when a test named in `REQUIRE_TESTS` did not pass. With `GOTEST_ONLY_REQUIRED=1` it runs exactly those names.

| Job | Command | Timeout argument |
|---|---|---|
| `compat` | `gofmt -l .` (must print nothing), `go vet ./...`, `go build ./cmd/stateripple` and `stateripple version`, `go test -count=1 ./...` | `-timeout=5m` |
| `enhanced` | `go test -race -count=1 ./...` | `-timeout=5m` |
| `enhanced` | `go test -shuffle=on -count=2 ./...` | `-timeout=10m` |
| `enhanced` | five named groups (below), each `go test -count=1 ./...` limited to its tests | `-timeout=5m` |
| `long-shuffle` (manual) | `go test -shuffle=on -count=10 ./...` | `-timeout=60m` |

Named groups in `enhanced` (the exact test names are in the workflow):

| Group | Covers |
|---|---|
| Generic graph regression | generic node/edge kinds, attenuation 0..3, zero-attenuation cycles and chains at the generation limit, reference-model comparison, large sparse graph, input-order determinism |
| Snapshot golden and hash | pinned golden reports and hashes, defaults normalization, repeatable runs |
| Git-free acceptance | the built binary with an empty `PATH` and no Git or Go (GF-01 to GF-15), version, generic Text output |
| Offline security | missing blob never fetched (partial clone, seven inherited Git environments), child environment policy, private-VCS rejection, saved Go configuration ignored, unsupported Git rejected before Go collection, HEAD integrity |
| Output safety and No-Clobber | existing output, directories, symlinks, aliases, trailing separators, concurrent writers, permissions, partial writes, hostile file names in diagnostics |

`TestExistingOutputRetainsDarwinACLAndXattr` only exists on macOS and is not part of Linux CI.

### Expected time

Measured locally (not on GitHub Actions); hosted runners have different CPUs, so treat these as the order of magnitude and expect the first real run to differ:

| Command | macOS arm64 (Apple M1), Go 1.23.12 / 1.26.9 / 1.27.2 | Linux arm64 container (Debian 13, Git 2.47.3, no network), Go 1.23.12 / 1.26.9 / 1.27.2 |
|---|---|---|
| `go test -count=1 ./...` (cold build cache) | 60 s / 71 s / 66 s | 28 s / 32 s / 33 s |
| `go test -race -count=1 ./...` | 96 s (Go 1.27.1) | 68 s (1.27.2) |
| `go test -shuffle=on -count=2 ./...` | 147 s (Go 1.27.1) | 69 s (1.27.2) |
| five named groups together | 28 s | 13 s |
| `binary-acceptance.sh` + `analyze-acceptance.sh` | about 11 s | about 10 s (inside a network namespace) |

Test results were 504 passes on macOS and 501 on Linux (the three differences are `TestExistingOutputRetainsDarwinACLAndXattr` and its two subtests, which only exist on macOS), with no failure and no skip on either.

The adapters package dominates (it starts many Git and Go processes). `-timeout=5m` per package had a wide margin in every local run above; if hosted runners prove slower, raise the timeout in the workflow rather than skipping tests.

The first hosted runs (2026-10-10, run IDs in the status table above) were faster than every local row: the `push` run took 91 s wall (compat 37–44 s per Go version, `enhanced` 79 s, `linux-binary` 21 s, `ci-ok` 4 s), the manual run with the long shuffle 120 s wall (its `long-shuffle` job 116 s) and the two `pull_request` runs 80 s and 93 s wall (compat 37–46 s, `enhanced` 70–83 s, `linux-binary` 21–24 s). The suite has no third-party module to download, so the offline policy costs nothing on the runner. Each compat job, including the build and the full suite, finished in under a minute, so the per-package `-timeout=5m` kept the same wide margin as locally.

### Running the long shuffle later

Actions tab, workflow `CI`, "Run workflow", tick `long_shuffle`. Locally:

```sh
GOTOOLCHAIN=local go test -shuffle=on -count=10 -timeout=60m ./...
```

Earlier local runs took about 10 minutes for the adapters package at `-count=10` on Apple M1. Without an explicit `-timeout` Go stops a package after 10 minutes, so the timeout is always given. On the hosted runner the whole `long-shuffle` job passed in about 2 minutes (run 38027059887), far inside its 90-minute job timeout.

## Offline checks and their limits

The strict path (jobs `enhanced`, `linux-binary`, `long-shuffle`) sets:

```text
GOPROXY=off  GOSUMDB=off  GOTOOLCHAIN=local  GOWORK=off
GIT_NO_LAZY_FETCH=1  GIT_OPTIONAL_LOCKS=0  GIT_TERMINAL_PROMPT=0
```

These variables are acquisition policy for the Go and Git tools. **They do not isolate the network**; the product also sets them for its own child processes, and the tests include hostile inherited values.

OS-level isolation is only claimed where it is created and checked:

- `scripts/ci/in-netns.sh` runs a command as the current user inside a new, empty Linux network namespace (`sudo unshare --net`, then `setpriv` back to the unprivileged user). It fails if the namespace cannot be created; there is no fallback.
- `scripts/ci/binary-acceptance.sh … --expect-no-network` first checks isolation from inside (the IPv4 routing table in `/proc/net/route` is empty, and an outbound and a loopback TCP connection both fail; `/proc/net` is used because `/sys/class/net` shows the namespace sysfs was mounted in, and fallback tunnel devices such as `tunl0` can exist in any namespace, so the interface list is only printed), then drives the built binary with `env -i` and an empty `PATH`. It checks `version`/`help`, the three golden reports byte for byte and by pinned SHA-256 (report and `input_sha256`), canonical output independent of input order and spacing, five identical repeats, the exit-code contract, `analyze` failing closed without Git, and the `--out` rules.
- `scripts/ci/analyze-acceptance.sh` runs a real `analyze` (Git and Go present) in a second empty namespace: a complete analysis with HEAD, index, object store and work tree unchanged; `--out` No-Clobber and path safety; a Git reporting 2.44.2 rejected before Go starts; a `blob:none` partial clone with two missing base blobs giving exit 2, with a canary upload-pack proving that nothing was fetched, plus a control showing that the canary reacts when lazy fetching is allowed.

What "no Git or Go" means: the binary is started with an empty `PATH`, and the script verifies that `git` and `go` cannot be found from that environment. Git and Go are still installed on the runner. A container image that physically lacks them would be stricter; that is not part of CI.

Fixture construction (git, `go list`) happens inside the checks and is not part of the product's behavior; the product's own offline guarantees are what the tests assert.

### Exit-code contract

| Exit | Meaning | Checked with |
|---|---|---|
| 0 | complete | goldens, `limit-2` chain, `version`, `help`, complete `analyze` |
| 1 | error (invalid input, usage, configuration, output refusal, missing Git) | invalid JSON, missing file, omitted `--input`, unknown command, flag or format, attenuation 4 or omitted, unknown ruleset, `--out` refusals, uncommitted change, unknown base, Git too old |
| 2 | incomplete | generation limit (`max_generations: 1` on a zero-attenuation chain), `analyze` of a partial clone with missing base blobs |

## Runtime tests and cross compilation

CI executes the test suite on Linux (x86-64 hosted runners) only. Cross-compilation checks, which build or `go vet` for other targets without running anything, are not part of this workflow: they show that code compiles, not that it behaves correctly. Execution on macOS, Windows and the BSDs is not covered by CI; see [VALIDATION.md](VALIDATION.md) for what was run locally and what was only compiled.

## Before the first release

All of the following are required; a green CI run alone is not a release decision.

1. Done (2026-10-10): the first push ran the `CI` workflow on `main` and every job in `compat` (1.23, 1.26, 1.27), `enhanced`, `linux-binary` and `ci-ok` is green — [run 38026796484](https://github.com/cottondesu/stateripple/actions/runs/38026796484). Timings are recorded under [Expected time](#expected-time), and the hosted results are recorded in [VALIDATION.md](VALIDATION.md#current-record-github-actions-hosted-runs-2026-10-10) under "Current record: GitHub Actions hosted runs".
2. Done (2026-10-10): the `long_shuffle` manual run is green — [run 38027059887](https://github.com/cottondesu/stateripple/actions/runs/38027059887).
3. Done (2026-10-10): the runner printed Git 2.55.0 on both runs and the gate passed.
4. Done (2026-10-10): the hosted runs were faster than the local expectations and nothing was skipped; see [Expected time](#expected-time). Hosted CI results are recorded in [VALIDATION.md](VALIDATION.md#current-record-github-actions-hosted-runs-2026-10-10) under "Current record: GitHub Actions hosted runs".
5. Done (2026-10-10): pull request runs from a branch are green — [run 38027821273](https://github.com/cottondesu/stateripple/actions/runs/38027821273) and [run 38028313271](https://github.com/cottondesu/stateripple/actions/runs/38028313271), both event `pull_request`, PR #1 (`docs/verify-pr-ci-trigger` → `main`); every required job passed and `long-shuffle` skipped as designed in both. Concurrency cancellation was not exercised: the two runs executed sequentially, so no in-progress run was superseded.
6. Done (2026-10-10): the platform statement for the release notes is fixed — Linux amd64: execution-verified on GitHub Actions; Linux arm64: execution-verified in a local Docker environment; macOS arm64: execution-verified locally; Windows, macOS Intel, the BSDs and illumos: not execution-verified. Cross-compilation success is never presented as execution verification.
7. Release itself (tag, GitHub Release) is done by hand after the above and is not automated.

## Not verified yet

- PR-scoped concurrency cancellation: the `pull_request` event path itself is verified (runs 38027821273 and 38028313271 on PR #1), but a newer run cancelling an in-progress run of the same pull request has not been observed — the two runs executed sequentially, without an overlapping in-progress run (runs on `main` are never cancelled by design).
- CPU count and speed of the hosted `ubuntu-24.04` runners: one runner's timings and its Git 2.55.0 are recorded above, but the fleet is not profiled.
- macOS Intel, Windows, FreeBSD/OpenBSD/NetBSD, illumos execution.

## Known limits

- The checks assert StateRipple's behavior in the tested environments; they do not prove the absence of other defects, and a pattern scan cannot prove the absence of every secret.
- `-race` needs cgo and a C compiler, which the hosted Linux runner provides.
- Pinned action commits do not update themselves; see the update procedure above.
