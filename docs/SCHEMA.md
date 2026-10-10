# Schema v1

Snapshot and report `schema_version` are 1. The only supported ruleset is `risk-propagation-v1`. Unknown schema versions and rulesets are errors, not partial analyses. Future incompatible changes require another schema version; rules with different semantics require a distinct identifier. Internal Go APIs are not a stable public API in v0.1.0.

## Snapshot

| Field | Type | Meaning |
|---|---|---|
| `schema_version` | integer | Required, exactly 1 |
| `ruleset` | string | Required, `risk-propagation-v1` |
| `max_generations` | integer | Optional in JSON; omitted means 16; explicit value must be 1..1024 |
| `nodes` | array | Required; objects `{id, kind}` |
| `edges` | array | Required; objects `{from, to, kind, attenuation}` |
| `seeds` | array | Required; objects `{id, risk, reason}` |

Node `kind` and edge `kind` are free-form labels: any nonempty valid UTF-8 string of at most 4,096 bytes, for example `service`, `database`, `job`, `package`, `test`, `depends_on`, `ships_to` or `own_test`. No vocabulary is enforced, propagation never depends on a kind, and attenuation is never inferred from an edge kind. One report convention interprets a node kind in v0.1.0: nodes whose kind is exactly `test` are projected into `test_targets` (see Report). `attenuation` is an explicit integer 0..3 (risk-3 contributions 3, 2, 1, 0); negative values, 4 or more, non-integer literals such as `1.0`, strings and numbers outside the integer range are rejected. Edge direction `from → to` means the state of `from` affects `to`. Both endpoints must exist. Self-loops and cycles are valid. Risk is an integer 0..3 and seed reasons are nonempty. IDs need not use Go import paths.

The Go adapter (`analyze`) emits a Go profile of this schema and validates it before simulation: node kinds `package` and `test`; `prod_import` package → package with attenuation 1; `test_import` package → test with attenuation 1; `own_test` package → test with attenuation 0; no outgoing edges from test nodes. `simulate` does not apply this profile. Before the generic change the core enforced it; every snapshot accepted then is still accepted and normalizes to the same canonical bytes and `input_sha256`.

All object fields above are required except `max_generations`. Field names are case-sensitive. Unknown fields, duplicate JSON object keys, null values, invalid UTF-8 or unpaired Unicode surrogate escapes, extra trailing JSON values, wrong types, noninteger risks and invalid graph data are rejected. Empty arrays are valid. An explicit generation limit of 0 is invalid; it is not a default request.

### Limits

| Input dimension | Maximum |
|---|---|
| Input JSON bytes | 16 MiB (16,777,216), including whitespace |
| Nodes | 20,000 |
| Edges before merging | 200,000 |
| Seeds before merging | 100,000 |
| Node ID, node/edge kind, seed reason string | 4,096 UTF-8 bytes, nonempty |
| JSON nesting depth | 16 |
| Update count | 1,024 |

These initial bounds allow sizable sparse projects while bounding input allocation and iteration. They are resource limits, not performance guarantees. Edge endpoints must refer to validated IDs; kinds are validated strings, not enumerations. Snapshot reading uses a limited reader before JSON decoding, and token scanning checks array cardinality before allocating decoded object arrays.

### Normalization

1. Nodes sort by ID ascending, using Go string byte comparison. Duplicate IDs are errors, even for identical nodes.
2. Edges sort by `(from, to, kind, attenuation)` ascending. Exact duplicates merge. Conflicting kind or attenuation on the same endpoints is an error. Go collection omits same-package `test_import` in favor of `own_test`.
3. Seeds sort by `(id ascending, risk descending, reason ascending)`. Exact `(id,risk,reason)` duplicates merge. Distinct reasons and lower-risk records are retained. The engine uses the largest risk for each node.
4. All empty normalized arrays are `[]`, not null. The default update limit is explicit after reading.

Normalization does not infer ownership from arbitrary IDs, repair malformed edges, trim IDs, normalize Unicode forms, or reinterpret unsupported rules. Applying normalization again gives the same input.

### Hash and canonical encoding

`input_sha256` is lowercase hexadecimal SHA-256 of the normalized Snapshot's canonical JSON bytes, including its one final LF. The hash input contains these fields in this exact order:

```text
schema_version, ruleset, max_generations, nodes, edges, seeds
```

Node field order is `id, kind`; edge order is `from, to, kind, attenuation`; seed order is `id, risk, reason`. Arrays use the normalization order above. Encoding is compact Go `encoding/json` encoding, including its standard string and HTML escaping, without indentation or floating-point values. It ends in exactly one LF. The hash itself, report, warnings, changed paths, build context, time and absolute filesystem paths are excluded.

Thus JSON whitespace, input object field order, array order, and exact duplicate edges/seeds do not change the normalized hash. Different seed reasons or an explicit different limit do change it.

## Report

Top-level fields are emitted in this exact order:

```text
schema_version, ruleset, status, input_sha256, generation_count,
cells, test_targets, waves, unknown_changes, warnings, changes,
build_context, stop_reason
```

| Field | Meaning |
|---|---|
| `status` | `complete` or `incomplete`; does not indicate test success |
| `generation_count` | Number of updates, including unchanged convergence confirmation; excludes generation 0 |
| `cells` | Every node, including risk 0 |
| `test_targets` | Nodes whose kind is exactly `test` (case-sensitive) and whose risk is positive, sorted risk descending then ID ascending. This applies to every report, including `simulate` on a non-software graph that uses the label `test`; it is the only kind the core interprets and is kept so that earlier reports stay byte-identical |
| `waves` | Generation-0 positive seed application and subsequent actual increases; no no-change confirmation wave |
| `unknown_changes` | Changed paths whose impact was not safely classified |
| `warnings` | Completeness or convergence warnings |
| `changes` | Git records with path, status and classification; empty for simulation |
| `build_context` | Analysis conditions; null for simulation |
| `stop_reason` | `converged` or `generation_limit`; describes engine stopping, independently of adapter completeness |

A cell's field order is `id, kind, risk, seeds, first_active_generation, causes`. `first_active_generation` is null for never-active nodes. `seeds` retains original normalized seed information. A cause contains `source`, optional `edge`, and `reason`. Seed causes omit `edge`; propagated causes contain the complete edge and use the edge kind as reason. Only positive causes producing the maximum are listed. Final-state support and historical change reasons are distinguished by their placement.

A wave contains `generation, changes`. A change contains `id, before, after, causes`. A test target contains `id, risk`. A file change contains `path, status, classification`; Git rename detection is disabled, so movement appears as deletion/addition. Typical classifications are `source`, `native`, `test`, `embed`, `test_embed`, `module`, `excluded_document`, and `unknown`; shared ownership may combine classifications with commas. Active native input changes seed the owning package at 3 with reason `native_changed:<path>`. Gitlink changes are `unknown`, including paths resembling documentation.

`build_context` contains `go_version, goos, goarch, cgo_enabled, module, build_tags, tool_tags, release_tags`. `cgo_enabled` is a string from Go environment output. The current-target conditions apply; this is not a result for every build configuration.

### Report ordering

Cells and wave changes sort by node ID ascending. Waves sort by generation ascending. Report seed records sort by ID ascending, risk descending, and reason ascending. Causes sort by source, reason, then edge (`from, to, kind, attenuation`); absent edges precede present edges when earlier keys tie. Changed files sort by path, status, classification. Warnings, unknown paths and build-tag arrays sort ascending and deduplicate. Empty arrays encode as `[]`.

Report JSON is compact with one final LF, using the same canonical encoding convention as snapshot hashing. No runtime timestamp, temporary filename, or checkout absolute path is intentionally included. Canonical here means StateRipple's specified encoding, not a claim to implement RFC 8785.

### Completeness layers

| Layer | Meaning | Decided by | Report / exit |
|---|---|---|---|
| 1. Input validity | JSON and graph are well formed | Snapshot reader and normalization (also run on collected analyze graphs) | Invalid → exit 1, no report |
| 2. Adapter completeness | The adapter collected the required external evidence | Adapter metadata (analyze only; does not apply to `simulate`) | Known gaps → `status: incomplete`, `warnings`/`unknown_changes`, exit 2 |
| 3. Simulation completion | The rule reached a confirmed fixed point within the limit | Engine | `stop_reason` `converged` or `generation_limit`; limit → `incomplete`, exit 2 |
| 4. External-world coverage | The input contains every real dependency | Nothing | Not represented |

`status` is `complete` exactly when `stop_reason` is `converged` and the adapter reported no known gap. For `simulate`, `build_context` is null and `incomplete` always means `generation_limit`. For `analyze`, the Go profile bounds walks so that `stop_reason` is `converged` and `incomplete` means adapter incompleteness. `complete` never claims layer 4.

### Trace semantics

Wave 0 lists nodes whose maximum seed is positive, with the seeds of that value as causes. Wave g ≥ 1 lists nodes whose risk increased in update g; each cause is an edge whose source had, in the previous generation, `risk − attenuation` equal to the new value. Following wave causes always reaches a wave-0 seed, including with attenuation 0. Final `cells[].causes` list every seed and incoming edge that supports the final value in the final state. They are support, not origin: with attenuation-0 cycles or self-loops a cell can be supported by a cycle neighbor or itself. If the run stopped at the generation limit (`incomplete`), the final state has not been confirmed as a fixed point. When it is not one, a positive cell can have no final cause because its supporting neighbor increased afterwards; its wave records the cause.

A `generation_limit` result is a lower bound of the fixed point, not an approximation of it. Reaching the fixed point was not confirmed: the last permitted update still changed a cell, so the unchanged update that confirms convergence never ran. The final state can in fact already be the fixed point (a 1025-node zero-attenuation chain with limit 1024 ends with every value final), but this cannot be told from the run, so the result is `incomplete` either way. Risk only increases, so no cell exceeds its converged value, but a cell can be lower, including 0; `test_targets` can miss a target or list it at a lower risk; final causes can be empty or list only part of the eventual support. Waves are exact for the updates that ran. Rerun with a larger `max_generations` (at most 1024) for a confirmed result.

This state needs `simulate` with a `max_generations` smaller than the propagation depth. The Go vocabulary allows it as well, because `own_test` has attenuation 0: `pkg:A` (seed 3) → `pkg:B` (seed 1) by `prod_import`, `pkg:B` → `test:B` by `own_test` and `max_generations` 1 leave `test:B` at risk 1 with no final cause. `analyze` does not produce it: it always emits `max_generations` 16 and a validated Go profile, in which a positive contribution crosses at most two attenuation-1 edges followed by at most one `own_test` edge into a test node without outgoing edges, so it converges within 4 updates.

### Status and exit boundary

NEW-01/02/03/04 retain the Snapshot SHA-256, canonical JSON encoder, report key shape and RiskPropagationV1. Analysis classifications change only where completeness was previously overstated: every regular base Go blob must be read successfully, and no base `.go` entry may be a symlink, before former ownership or unowned document exclusion is trusted. Any missing/unreadable base source produces a diagnostic warning and incomplete/exit 2 with a useful graph; unowned documents are `unknown`. Complete sources retain `former_embed`/`former_test_embed`. A missing commit/tree that prevents collection yields error/exit 1. Inherited Git trace settings do not affect status. Unsupported Git (below 2.45.0 or lacking `--no-lazy-fetch`) errors before collection. These requirements do not apply to simulation.

Output parent paths are resolved before lexical normalization, preserving directory-symlink/`..` semantics and refusing nonexistent intermediate directories. Existing-entry No-Clobber and trailing-separator rejection remain unchanged. Ignored Go FIFOs are an analyze availability limit (NF-04), separate from regular Snapshot-input rejection.

Valid simulation or analysis with confirmed convergence and complete adapter metadata exits 0. Unknown changes, dependency errors, unsupported configurations, unresolved deleted ownership or unconfirmed convergence yield useful partial results with status `incomplete` and exit 2. Malformed inputs or execution errors without a useful result exit 1 with stderr diagnostics. Each diagnostic is one line: nongraphic characters in it (control characters, line and paragraph separators, bidirectional and other format controls, invalid UTF-8 bytes), whether they come from a path, a revision, a flag name or tool output, are shown as escapes such as `\n`, `\x1b` or `\u202e`. A failed or short write of help, usage or version text also exits 1. Adapter incompleteness can coexist with `stop_reason: converged` because the engine may converge on the known subgraph; such a report describes only the collected evidence, and unknown changes are not seeded.

HEAD/worktree raw-byte or mode mismatch is an execution error (exit 1), regardless of index flags. Uncommitted active ignored inputs, unresolved native ownership and changed gitlinks make analysis incomplete (exit 2). Reachable cached versioned remote replacements are supported; reachable directory replacements and unresolved dependencies are incomplete. Unused local/remote directives, even missing unused directory targets, do not alone affect completeness. Scope is current ./... dependency/test packages, subject independently to workspace/nested-module restrictions. A go.mod change still seeds all packages.

Snapshot/Report shape, normalization and hash encoding are unchanged by RL-008–RL-010. CLI paths are a separate boundary: regular Snapshot files only; --out rejects every existing entry and input alias. New output is private 0600 and published by exclusive same-directory hardlink after full write/sync/close. Existing metadata and bytes are unchanged. Cleanup failure after publication returns exit 1 with an explicit published-output diagnostic; the complete file may exist. Text escapes untrusted nongraphic characters for display. JSON retains actual values through its existing escaping rules. Acquisition controls and output concurrency limits are in [ARCHITECTURE.md](ARCHITECTURE.md).

## Sample input and output

NF-01/NF-02 do not change Snapshot/Report encoding, normalized-input SHA-256 or RiskPropagationV1. CLI output arguments ending in an OS separator are rejected before normalization, even if nonexistent. Existing directories (including dot paths), directory symlinks and dangling symlinks are rejected by the shared JSON/Text writer; no dir/dir reinterpretation occurs. No-Clobber is still exclusive new-file publication, not replacement. Missing Git objects cannot be fetched implicitly: useful partial results are incomplete/exit 2 and execution failures are exit 1. Git/Go acquisition policy details are in ARCHITECTURE.md; it is not OS network isolation.

This minimal input uses a changed A, its importer B, and A's own test:

```json
{"schema_version":1,"ruleset":"risk-propagation-v1","max_generations":16,"nodes":[{"id":"pkg:A","kind":"package"},{"id":"pkg:B","kind":"package"},{"id":"test:A","kind":"test"}],"edges":[{"from":"pkg:A","to":"pkg:B","kind":"prod_import","attenuation":1},{"from":"pkg:A","to":"test:A","kind":"own_test","attenuation":0}],"seeds":[{"id":"pkg:A","risk":3,"reason":"source_changed"}]}
```

For this minimal input, the final risks are A=3, B=2, test:A=3. Generation 1 activates B and test:A; generation 2 confirms convergence.

The runnable [example snapshot](../testdata/example.json) extends the chain through C and D and includes each package's own test. Run `stateripple simulate --input testdata/example.json --format json`. The final package risks are 3, 2, 1, 0, with the same values for their tests. Generation 3 activates test:C at 1, and generation 4 confirms convergence. Its exact generated report is recorded in [example.report.json](../testdata/example.report.json), including the input hash and complete trace.

The [generic supply-chain snapshot](../testdata/generic-supply-chain.json) uses only non-software kinds (`supplier`, `facility`, `warehouse`, `retailer`, `ships_to`, `delivers_to`) and attenuations 0, 1 and 2. Final risks are mine 3, smelter 3, port 2, factory 2, store 1, backup 3, with `generation_count` 5 and `input_sha256` `c34ed81c120cb7e518ee6c407ef1f94c0a77f7cc53819cc76157ffc8db3093de`. Its exact report is [generic-supply-chain.report.json](../testdata/generic-supply-chain.report.json).

A complete small output example, observed from the CLI for an empty graph with limit 16, is:

```json
{"schema_version":1,"ruleset":"risk-propagation-v1","status":"complete","input_sha256":"a246bcb6c6b2cf0244b5c9d2ba58e6909a89c5bf6527ffb7fcd8626e642e6818","generation_count":1,"cells":[],"test_targets":[],"waves":[],"unknown_changes":[],"warnings":[],"changes":[],"build_context":null,"stop_reason":"converged"}
```

This corresponds to `{"schema_version":1,"ruleset":"risk-propagation-v1","max_generations":16,"nodes":[],"edges":[],"seeds":[]}`. The displayed report has one trailing LF in actual output.
