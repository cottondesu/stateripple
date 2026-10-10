# Changelog

## v0.1.0 (unreleased)

StateRipple identity and generic harness:

- Rename the project from the working name RiskLife to StateRipple: module `github.com/cottondesu/stateripple`, command `cmd/stateripple`, binary and diagnostics prefix `stateripple`, repository `https://github.com/cottondesu/stateripple`. The former `cottondesu/risklife` repository was deleted and is not a publication target. Canonical JSON, `input_sha256`, report bytes, ruleset ID and every calculation are unchanged by the rename.
- Accept free-form node and edge kinds (nonempty valid UTF-8, at most 4,096 bytes) in the core graph. Attenuation is an explicit integer 0..3 independent of edge kind; negative values, 4 or more, non-integers and out-of-range numbers are rejected. Risk stays 0..3 and RiskPropagationV1 is unchanged. The generation limit stays 1..1024 (default 16); deep zero-attenuation chains end `incomplete` at the limit, never `complete`.
- Move the Go vocabulary (`package`/`test`, `prod_import`/`own_test`/`test_import`, fixed attenuations, no outgoing test edges) from the core to Go adapter profile validation, so `analyze` stays fail-closed. Snapshot shape is unchanged; the accepted set is a strict superset and every previously valid snapshot keeps its canonical bytes, hash and report.
- Make Text output mode dependent: generic simulations omit Git and test sections; help text describes `simulate` as the primary, Git/Go-free path and exit 2 as an incomplete result.
- Make tests hermetic against inherited `GOFLAGS`, `GOEXPERIMENT` and `GOWORK` (adapters clear them before any test starts; CLI child processes receive explicit environments). Production still reports such inherited settings as incomplete.
- Add generic graph, Go profile, Git-free (GF-01–GF-15), pinned golden hash, reference-model, zero-attenuation and fuzz tests, plus a non-software golden fixture `testdata/generic-supply-chain.json`.
- Post-audit remediation (SR-01, SR-02, SR-03, SR-07; no product code, schema, golden or hash change): ignore local agent state (`/.omx/`, `/.mandala/`) so that it cannot be committed; add regression tests that remove exactly one required field (`attenuation`, `risk`, `kind` and the others) from an otherwise valid snapshot and check the diagnostic, exit 1 and the distinction between omission, null and an explicit 0; document that a `generation_limit` result is a lower bound whose test targets and final causes can be incomplete, that this also occurs with the Go vocabulary under `simulate` (not `analyze`), and that kind `test` is the one label the core still interprets for `test_targets` (separation planned for v0.2.0).
- Final cross-model audit remediation (CA-01–CA-05 and the SR-07 source comment; RiskPropagationV1, the snapshot schema, canonical JSON, reports, goldens and hashes are unchanged):
  - CA-01: stderr diagnostics are escaped as a whole and are one line each. Control characters (LF, CR, ESC, NUL and the other C0/C1 controls), Unicode line and paragraph separators, bidirectional and other format controls and invalid UTF-8 bytes in a path, revision, flag name or Git/Go message appear as `\n`, `\x1b`, `\u202e` and similar escapes, also inside wrapped `PathError`/`LinkError` and joined errors. Diagnostics without such characters are byte-identical to before. Flag parse errors are printed by the CLI instead of the `flag` package, with the same text.
  - CA-05: a failed or short write of help, usage or version text exits 1 with `write output` on stderr instead of 0. Help and usage text, their stream and their exit code on success are unchanged.
  - CA-03 (tests only): the partial-clone offline test builds its fixture without Git's implicit promisor fetch (clone without checkout, explicit local pack of the HEAD blobs, checkout with lazy fetching forbidden), so it passes with an inherited `GIT_NO_LAZY_FETCH=1`. It now runs under seven inherited Git environments, checks that exactly the intended base blobs are absent, that two missing sources are both reported while a present embed source is not, that the object store is unchanged, and that the fetch canary is live. The child-policy test also requires the `--no-lazy-fetch` option, not only the environment variable. The Git adapter itself is unchanged.
  - CA-04: documents no longer state that a `generation_limit` final state is not a fixed point. Reaching the fixed point is unconfirmed; the state can already be the fixed point, and the result is `incomplete` because that is not confirmed. No calculation changed.
  - CA-02: local absolute paths (home, workspace, backup and uniquely named temporary directories) in the validation and design records are replaced by placeholders; recorded commands, conditions and results are kept.
  - SR-07: the `internal/graph` comment now names the `test` projection as the exception to "kinds carry no meaning".

Continuous integration (written and checked locally, then run green on GitHub Actions on all three trigger paths — `push`, `workflow_dispatch` with the long shuffle, and `pull_request`; see [docs/CI.md](docs/CI.md) and [docs/VALIDATION.md](docs/VALIDATION.md)):

- Add `.github/workflows/ci.yml`: Go 1.23 (minimum, compatibility only), 1.26 and 1.27 on `ubuntu-24.04` (gofmt, `go vet`, CLI build, `go test`); Go 1.27 with a strict offline environment, `-race`, `-shuffle=on -count=2` and five named regression groups; the built static binary run inside an empty network namespace (Git/Go-free `simulate`, golden bytes and pinned SHA-256, exit codes 0/1/2, `--out` No-Clobber and path safety) and a real `analyze` including a `blob:none` partial clone; a manual-only `-shuffle=on -count=10` job. Read-only token, no publishing step, official `actions/checkout` v7.0.1 and `actions/setup-go` v7.0.0 pinned to commit SHAs, no third-party action.
- Add `scripts/ci/`: a Git 2.45+ gate that compares versions as numbers (with a self-test against stand-in Git executables), a `go test -json` wrapper that fails on skips, on zero tests run and on missing required tests, and the acceptance scripts above. No product source, snapshot schema, RiskPropagationV1, golden or hash changed.

Audit remediation (recorded under the former name RiskLife):

- NEW-01: read every regular base Go blob explicitly by OID after NUL tree enumeration; distinguish no directive from missing/corrupt input independently of grep exit status. A base-tree symlink ending in `.go` is an unsupported base source (incomplete) instead of being skipped. Aggregate diagnostics, return useful incomplete results, and keep unowned documents unknown when base completeness cannot be established. Local mixed partial-clone fixtures verify no fetch or object-store mutation.
- NEW-02: require Git 2.45.0+ and actually probe `--no-lazy-fetch` before collection. The installed official release notes correct the audit's tentative 2.44 minimum. Unsupported/missing Git gives a requirements error with no unsafe fallback; simulation still needs neither Git nor Go.
- NEW-03: remove inherited `GIT_TRACE*` from Git/Go child environments while retaining real read-error diagnostics and parent settings.
- NEW-04: resolve the original output parent before lexical cleaning; missing intermediate directories fail and symlink/`..` follows the physical parent. Preserve No-Clobber and metadata protection.
- NF-04: independently confirm FIFO hangs and orphaned Go children after parent-only SIGKILL. Defer P3 with P2 operational treatment for mutable/shared CI; document TERM-first/job-group termination and v0.1.1 candidates.

- NF-01: forbid implicit promisor fetch with a shared child-only Git environment, including Go-launched Git; require `--no-lazy-fetch` for direct Git calls so unsupported Git fails closed. NEW-01 completes missing-base-source detection beyond the earlier diagnostic-bearing grep exit 1 check.
- NF-02: reject trailing OS separators before output normalization and inspect the original output argument; JSON/Text always reject directories and symlinks while retaining exclusive new-file No-Clobber publication.
- Document five remaining P3 limitations (tracked symlinks, ignored Go FIFO, post-publication cleanup, long basenames, excluded fixture modules) and workarounds. See the current final-blocker validation record.

- RL-001: verify raw HEAD blobs and executable modes before/after collection without trusting index optimization flags; reject transformed or mismatched worktrees.
- RL-002: force submodule changes into raw Git diffs and classify gitlink mode changes as incomplete.
- RL-003: include all reported native build inputs in ownership and committed-input checks.
- RL-004: reject input/output aliases and output symlinks; preserve partial-write and FIFO safety. Output is now new-file-only under RL-008.
- RL-005: escape untrusted control characters in Text reports without changing JSON encoding.
- RL-006: require regular Snapshot files and use nonblocking Unix opens with identity checks.
- RL-007: distinguish directory and versioned remote replacements using Go's `Replace.Version` metadata.
- RL-008: intentionally reject all existing --out entries before first release; publish new 0600 output with an exclusive same-directory hardlink after write/sync/close. Preserve ACLs, xattrs and bytes; report cleanup failure after publication accurately.
- RL-009: enforce GONOPROXY=none and GOVCS=*:off alongside GOPROXY=off; disable saved GOENV, injected flags, workspace and toolchain acquisition in all Go analysis subprocesses. Retain explicit target/cache environment and warn for unsupported parent flags.
- RL-010: document completeness for reachable replacement packages; unused local/remote directives alone do not cause incomplete. Missing reachable dependencies remain incomplete; cached remote replacements remain supported.
- Add individual and combined regression coverage; see the current remediation evidence in [VALIDATION.md](docs/VALIDATION.md).

Initial implementation:

- Introduce a validated and normalized package/test graph (generalized above).
- Add synchronous deterministic RiskPropagationV1, fixed-point detection, update limits, seed provenance, and maximum-support traces.
- Add bounded JSON snapshots, canonical report JSON, and normalized-input SHA-256.
- Add offline Git/Go analysis for committed changes in clean single-module repositories, with conservative completeness reporting.
- Rank test targets and report changes, waves, build conditions, exclusions, and warnings.
- Provide `version`, `simulate`, and `analyze`, JSON/text output, output files, and distinct complete/error/incomplete exit codes.
- Include unit, integration, determinism, fuzz/property, and benchmark coverage plus English/Japanese documentation.

Mandala is a development tracking tool only. No stdin input, explicit-input or Changed Files adapter, second rule, AI integration, automatic code modification, automatic test execution, or PatchLife is included.
