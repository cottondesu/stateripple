# StateRipple v0.1.0 (unreleased)

[日本語](README.ja.md)

*Deterministic ripple-effect propagation of state over any graph.*

StateRipple is an offline, game-of-life-style harness. It reads a directed graph and seeds, updates every node (cell) synchronously, generation by generation, and reports the resulting risk, the propagation trace, and a canonical hash of the input. Nodes and edges can describe services, databases, jobs, suppliers, packages, or anything else. Their kinds are free-form labels: no vocabulary is enforced and propagation never depends on them. The one exception in v0.1.0 is a report convention for the label `test`, described under the propagation rule. It runs locally without an LLM, network service, database, Git, Go toolchain, or third-party Go dependency.

Git and Go analysis is an optional adapter (`analyze`) that builds a snapshot from a Go repository. It is not required by the core.

StateRipple was previously developed under the working name RiskLife (`github.com/cottondesu/risklife`). That repository has been deleted; the module path is now `github.com/cottondesu/stateripple`.

Risk is a propagated priority, not a probability. Risk 0 does not prove safety. A `complete` report means the computation over the given input completed, not that the input describes the real world completely or that any system is correct.

## Build and run

Build from a local checkout with Go 1.23 or later:

```sh
go build -o stateripple ./cmd/stateripple
./stateripple version
./stateripple simulate --input testdata/generic-supply-chain.json --format text
./stateripple simulate --input testdata/generic-supply-chain.json --format json
./stateripple simulate --input testdata/example.json --format json --out /tmp/stateripple-report.json
```

`simulate` needs only a snapshot file and the compiled binary. It works outside any repository, in a directory without `.git`, in a freshly initialized repository without HEAD, and with neither `git` nor `go` on `PATH`. `help` and `version` have the same property. See [the schema](docs/SCHEMA.md) for the snapshot and report formats.

A snapshot is one JSON file:

```json
{
  "schema_version": 1,
  "ruleset": "risk-propagation-v1",
  "max_generations": 16,
  "nodes": [{"id": "mine", "kind": "supplier"}, {"id": "smelter", "kind": "facility"}],
  "edges": [{"from": "mine", "to": "smelter", "kind": "ships_to", "attenuation": 0}],
  "seeds": [{"id": "mine", "risk": 3, "reason": "flood"}]
}
```

| Exit | Meaning |
|---|---|
| 0 | Complete result (`status: complete`) |
| 1 | Invalid input, usage, configuration, execution, or output error; no report |
| 2 | Incomplete result (`status: incomplete`): generation limit reached without a confirmed fixed point, or adapter evidence incomplete |

## Propagation rule: RiskPropagationV1

Each node holds risk `R(v)` in `{0,1,2,3}`. Each edge `u → v` means "u's state affects v" and carries an explicit attenuation `a` in `{0,1,2,3}`. Every update reads only the previous generation:

```text
R_next(v) = max(R(v), Seed(v), max over u→v of max(0, R(u) − a(u,v)))
```

| Attenuation | Contribution of a risk-3 source |
|---|---|
| 0 | 3 |
| 1 | 2 |
| 2 | 1 |
| 3 | 0 |

Attenuation 4 or more, negative attenuation, non-integer values and out-of-range numbers are rejected (exit 1). Because risk never exceeds 3, larger attenuations would carry no extra meaning; a different risk range would be a new rule, not a change to this one. Edge kinds are labels only: StateRipple never infers attenuation from a kind name.

| Risk | Name | Interpretation |
|---|---|---|
| 0 | Dormant | Inactive in this run |
| 1 | Watch | Indirect impact candidate |
| 2 | Active | High-priority candidate |
| 3 | Critical | Direct seed or strong propagation support |

Generation 0 applies seeds. The default limit is 16 updates (configurable 1..1024 with `max_generations`), including the final unchanged update that confirms the fixed point. Risk only increases, so every finite graph reaches a fixed point within `max(1, |V|)` updates; cycles and self-loops are safe. Attenuation 0 passes risk unchanged, so a zero-attenuation chain of n nodes needs n updates: with the maximum limit of 1024, a 1024-node chain completes and a 1025-node chain is reported `incomplete` (exit 2), never `complete`.

A result stopped by the limit is a lower bound of the fixed point: no reported risk exceeds its converged value, but a cell can still be too low or 0, `test_targets` can miss a target or list it at a lower risk, and final causes can be empty or partial. Rerun with a larger `max_generations` for a confirmed result. See [trace semantics](docs/SCHEMA.md#trace-semantics).

Unlike Conway's Game of Life (B3/S23), this rule has no birth/death counting and never decreases state: it is a monotone maximum propagation. "Game-of-life style" refers only to synchronous generation updates of cells.

Reports contain all cells, seed reasons, tied maximum causes, first activation generations, changing waves, warnings, the stop reason, and the input SHA-256. The `test_targets` field projects nodes whose kind is exactly `test` and whose risk is positive; generic graphs normally have none. This is the only kind the core interprets, kept in v0.1.0 so that earlier reports stay byte-identical. It applies to `simulate` too: a non-software graph that uses the label `test` gets those nodes in `test_targets` and the test wording in Text output, so use another label if that is not intended. Text output shows Git and test sections only for `analyze` reports or when test targets exist.

## Optional Git + Go adapter: analyze

Analyze a clean, committed single-module Go repository from its root:

```sh
/path/to/stateripple analyze --base HEAD~1 --format json
/path/to/stateripple analyze --base HEAD~1 --format text
/path/to/stateripple analyze --base HEAD~1 --format json --out /tmp/stateripple-analysis.json
```

`analyze` requires Go 1.23+ and Git **2.45.0+** with the global `--no-lazy-fetch` option. StateRipple probes that option and checks the reported version before repository/Go collection, and only when `analyze` runs. Unsupported or missing Git returns a requirements error (exit 1), with no unsafe fallback. Apple Git uses its own distribution version; the bundled Apple Git 2.30.1 tested here is unsupported. `simulate`, `help` and `version` need neither Git nor Go.

The default base is HEAD's parent. Analysis compares base with HEAD and excludes uncommitted work. Tracked changes and nonignored untracked files cause an error. Ignored build artifacts do not. Output defaults to stdout; diagnostics go to stderr. No file is created unless `--out` is supplied. Put output outside the analyzed checkout to avoid introducing an untracked file for the next run.

Before and after Go collection, analysis checks tracked regular-file bytes and executable modes against the HEAD tree, independently of `assume-unchanged` and `skip-worktree`. Raw bytes must match: clean/smudge filters or line-ending conversion that changes worktree bytes cause an error. Ignored files actually used by Go, including native build inputs, make the analysis incomplete. Gitlink changes are detected even with `diff.ignoreSubmodules=all` and are incomplete because submodule analysis is unsupported.

Snapshot input must resolve to a regular file; FIFO, socket, device and directory input is rejected. `--out` creates new files only and rejects every existing entry: regular files, hardlinks, symlinks (including dangling links), directories and special files. Existing bytes, ACLs, xattrs, ownership and permissions are unchanged. A same-directory temporary file is fully written with private `0600` permissions, synced and closed, then published with an exclusive hardlink (`os.Link`). Publication cannot replace an entry created concurrently. Unsupported filesystems return an error without a rename fallback. Choose a fresh output path each run; the example path must not already exist. Text escapes control characters; JSON values and canonical encoding are unchanged. Stderr diagnostics are one line each: control characters, Unicode line separators, bidirectional controls and invalid UTF-8 in a path, revision, flag name or tool message are shown as escapes (`\n`, `\x1b`, `\u202e`), so a hostile file name cannot forge a log line or drive the terminal. Names without such characters are shown unchanged.

The adapter builds a Go profile of the generic graph: `package` and `test` nodes; `prod_import` (package → package, attenuation 1), `test_import` (package → test, attenuation 1) and `own_test` (package → test, attenuation 0). If B imports A, the edge points from A to B. Test nodes have no outgoing edges. The adapter validates this profile on every collected graph and fails (exit 1) if it is violated; the generic core does not impose it on `simulate`.

A source change seeds its package at 3. A test-only change seeds its test at 2 and cannot propagate back into production. Changes to `go.mod` or `go.sum` seed every package at 3. Embedded files seed their owners. Clearly unrelated documentation is recorded as excluded; unknown changes make the analysis incomplete.

Active native inputs reported by `go list` (assembly, C/C++/Objective-C/Fortran, headers, SWIG and system objects) seed their owning production package at 3. Inputs excluded by the current build context or without a known owner remain incomplete. Reachable versioned remote replacements are supported from the local cache. Reachable directory replacements and unresolved local/remote dependencies are incomplete. Unused replace directives, including missing directory targets, do not alone affect completeness: analysis covers packages reached by current `./...` dependency/test collection, not every directive. Workspace/nested-module limits apply independently. Changing `go.mod` still seeds every package.

For A → B → C → D with A seeded at 3, final package risks are 3, 2, 1, 0. Their own tests receive the same risks ([example](testdata/example.json)). Test targets are sorted by risk descending then ID ascending. StateRipple does not run tests.

## Reproducibility and limits

Every Git subprocess receives `GIT_NO_LAZY_FETCH=1`, `GIT_TERMINAL_PROMPT=0`, and `GIT_OPTIONAL_LOCKS=0`, overriding inherited values without changing the parent. Direct commands also prepend `--no-lazy-fetch`. Child environments remove `GIT_TRACE*`, including Trace2, so inherited diagnostic tracing cannot change completeness or write trace files. Actual read errors remain in completeness warnings. Git launched by Go for buildvcs inherits this policy. These are Git/Go acquisition controls, not OS network isolation; installed tools, repository-configured helpers, credential helpers and wrappers are trusted.

Base embed analysis enumerates the complete base tree with NUL-separated `ls-tree` records and reads every regular `.go` blob by object ID with `cat-file blob`, one source at a time. It no longer uses `git grep`: grep can exit 0 despite a missing blob when another source matches. A successful existence check alone would not establish content readability. Source read/parse failures are collected and no base embed classifications are trusted unless the entire required source set succeeds. With a useful current graph, missing/unreadable base sources yield incomplete/exit 2; otherwise execution fails with exit 1. Unowned documents remain `unknown` rather than `excluded_document` when base completeness is unavailable. A base-tree symlink ending in `.go` is an unsupported base source and likewise makes base embed analysis incomplete. Normal complete sources retain `former_embed` ownership. Partial clones must already contain every required base Go blob and the trees/commits needed for collection; StateRipple never fills missing objects or implicitly fetches them. Object-store and fetch-blocker tests cover mixed present/missing sources and hostile inherited no-lazy/trace settings.

JSON and Text share output-path checks. `--out` rejects existing directories with or without trailing separators, directory/dangling symlinks, and dot-directory paths. Any trailing OS path separator is rejected before normalization, including on a nonexistent path; separators are never stripped to reinterpret the destination. Atomic new-file-only No-Clobber publication remains as described above.

Output resolves the original parent path before normalizing its basename: a missing `nx` in `nx/../report.json` is an error; a directory symlink followed by `..` resolves relative to the symlink target. Existing physical destinations remain protected. Stable trusted directories are still required.

Known P3 limitations remain open for v0.1.1 consideration:

| Finding | Current behavior and workaround |
|---|---|
| NF-03 | All tracked symlinks are checked, including unrelated documents. Directory/dangling targets error; regular targets are incomplete. Use a checkout with ordinary files. |
| NF-04 | An ignored local `*.go` FIFO can block `go list` without an internal deadline. Remove special files before analysis. Context cancellation, SIGINT and SIGTERM stopped the direct Go child in measured fixtures. Killing only the StateRipple parent with SIGKILL leaves blocked Go children; the killed parent cannot clean up. Unattended mutable/shared CI must send TERM first and, if forced termination is needed, kill the entire dedicated job process group/container. FIFO is not distributed by Git clone but can be created by local preparation/generators. Deferred P3; treat as a P2 operational risk where such mutable inputs are possible. v0.1.1 candidates: special-input preflight, deadline and platform-specific process-group control. |
| NF-05 | Post-publication cleanup failure returns exit 1 with a published-output diagnostic and may leave a temporary hardlink. Check the complete output and remove the residual temporary link after restoring permissions. |
| NF-06 | Long output basenames can exceed NAME_MAX once the temporary suffix is added. Use a shorter basename. |
| NF-07 | Tracked nested `go.mod`, even under Go-excluded `testdata`, causes incomplete. Accept this limit or rename/untrack fixture module files in an analysis checkout. |
| NEW-01 cost | Base embed analysis starts one `git cat-file blob` per base `.go` file. Measured on Apple M1: about 2.2 s / 20.8 s / 182 s for 100 / 1,000 / 10,000 files. Large repositories should allow for this time in CI. v0.1.1 candidate: an audited streamed `cat-file --batch` reader. |

Measured results, compatibility boundaries and proposed follow-ups are in [VALIDATION.md](docs/VALIDATION.md).

The same normalized snapshot produces the same risks, trace, generation count, SHA-256, and JSON bytes. Collection depends on HEAD, the installed toolchain, cached dependencies, and build context. The engine uses no time, randomness, network, floating-point risk, or concurrent updates.

All Go analysis subprocesses use one environment builder: `GOPROXY=off`, `GONOPROXY=none`, `GOVCS=*:off`, `GOSUMDB=off`, `GOTOOLCHAIN=local`, `GOWORK=off`, `GO111MODULE=on`, `GOENV=off`, `GONOSUMDB=none`, and empty `GOFLAGS`/`GOEXPERIMENT`/`GOPRIVATE`/`GOINSECURE`. List commands use `-mod=readonly`. Inherited private-module rules cannot bypass the disabled proxy; direct VCS and toolchain acquisition are disabled. Saved Go configuration and injected flags are ignored. Nonempty parent GOFLAGS/GOEXPERIMENT still cause an incomplete warning. Explicit GOOS, GOARCH, CGO_ENABLED and cache paths are retained; parent environment and persistent settings are unchanged. Cached modules remain usable; missing dependencies return incomplete or error. This is a Go acquisition policy, not OS network isolation: installed tools, wrappers and external cache helpers are trusted. Snapshot limits and hash encoding are in [SCHEMA.md](docs/SCHEMA.md).

Static imports cannot capture every runtime dependency. Dynamic loading, code generation, alternate build tags, workspaces, nested modules, and local replacement configurations have conservative limits. v0.1.0 analyzes the current target environment and standard build conditions, not every platform. Deleted files and removed test owners are accounted for when ownership is known; unresolved ownership is incomplete. Zero test targets does not mean there is nothing to verify. No comparative quality claim is made.

Use a stable checkout and trusted output directory. Analysis checks are not a filesystem transaction. Unix input opening is nonblocking and validates identity; other platforms retain a pre-check/open race. Output resolves and rechecks parent identity, but malicious parent/temp-name substitution is not fully excluded by portable path APIs. With a stable directory and retained destination entry, concurrent writers publish at most one complete file. Cleanup errors are reported; after publication the error explicitly states that output is already published, and exit 1 does not imply its absence. Cleanup failure or crashes can leave a temporary link. Directory fsync/crash durability is not guaranteed. macOS output and Linux container output on tmpfs/overlay were exercised, including concurrency and metadata preservation. Windows runtime remains unverified. Go 1.23.12 builds and passes the full test suite locally on macOS arm64 and in a Linux arm64 container; GitHub Actions has not run yet. Cross-compilation and precise Linux coverage are recorded separately in [VALIDATION.md](docs/VALIDATION.md).

## Development

```sh
gofmt -l .
go test ./...
go test -race ./...
go test -shuffle=on -count=10 -timeout 30m ./...   # adapters alone take about 10 minutes
go vet ./...
go build ./cmd/stateripple
git status --short   # also review untracked files; HEAD may not exist yet
```

### Continuous integration

[`.github/workflows/ci.yml`](.github/workflows/ci.yml) runs on pushes and pull requests to `main` and on manual dispatch. It only verifies: the token is read-only (`contents: read`), nothing is published, and the only actions are the official `actions/checkout` and `actions/setup-go`, pinned to commit SHAs. Commands, timeouts, offline-check scope and the pre-release gates are in [docs/CI.md](docs/CI.md).

- **Go.** 1.26 and 1.27 are the supported toolchains. 1.23 is the minimum version StateRipple builds and tests with; it is out of Go's security support and is checked for backward compatibility only. All three run the real toolchain (`GOTOOLCHAIN=local`).
- **Git.** Anything that runs `analyze`, including the Git-dependent tests, needs Git 2.45 or later. The first CI step fails with a clear message otherwise; the version is compared as numbers and nothing is skipped.
- **Offline.** `GOPROXY=off`, `GIT_NO_LAZY_FETCH=1` and the related variables are acquisition policy, not network isolation. The built binary is additionally run inside an empty Linux network namespace, with no Git or Go reachable from it, for `simulate`, `help`, `version`, the golden hashes and exit codes. A real `analyze` runs in a second namespace.
- **Runtime versus cross compile.** CI runs the tests on Linux only. Building for another operating system proves that it compiles, not that it works; macOS, Windows and BSD execution is not covered by CI.
- **Status.** The workflow was checked locally but has not run on GitHub Actions yet. A first run on `main`, a manual long shuffle run and the checks listed in docs/CI.md are required before a release.

[Architecture](docs/ARCHITECTURE.md), [schema](docs/SCHEMA.md), and [validation evidence](docs/VALIDATION.md) describe implementation and measured checks. Mandala is used only to track declared development requirements. It is not a StateRipple runtime dependency or domain model.

## Roadmap candidates

- v0.1.1 (reliability and performance): streamed `git cat-file --batch` reading for base embed analysis, fewer Git subprocesses, FIFO preflight/deadlines, tracked-symlink and nested-fixture refinements.
- v0.2.0 (input expansion): stdin snapshots, explicit graph/seed inputs, a Changed Files adapter, separated Git and Go adapters, structured diagnostics, a report v2 with an explicit completeness object, and moving the `test_targets` projection and test wording out of the generic core into an adapter-owned result.
- Later: multiple rules selected by `ruleset`, non-monotone and Conway-style rules with declared completion criteria, multi-state cells, and structured input/output for AI agents.

These are possible future directions, not implemented capabilities. v0.1.0 has one rule (RiskPropagationV1), one input format (a snapshot file; no stdin), and one adapter (Git + Go). It does not modify code, run tests, contact an AI API, or load dynamic plugins.

## License

[MIT](LICENSE), copyright 2026 cottondesu. MIT was selected as a small permissive OSS license for this initial implementation.
