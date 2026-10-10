# StateRipple（旧RiskLife） 汎用ハーネス仕様・名称レビュー・最小実装計画（設計記録）

| 項目 | 内容 |
|---|---|
| 作成日 | 2026-10-09 JST |
| 対象 | `github.com/cottondesu/stateripple` v0.1.0（未リリース、HEADなし）。作成時は`github.com/cottondesu/risklife` |
| 状態 | **採用・実装済み**（2026-10-10。決定事項は §0.1。以下の本文は作成時点の調査記録で、AS-ISは改名・汎用化前の状態を指す） |
| 作業範囲 | 調査・仕様策定・独立fixture実行・GitHub読み取り専用検索のみ。コード、既存テスト、既存ドキュメント、`.mandala/`は未変更。commit/push/tag/release/改名なし |
| 本書の扱い | 新規・未追跡ファイル。採否はユーザーが決定し、不要なら削除してよい |

根拠の記法は `path:line`。fixture証跡は作成時の一時領域にあった（以下`<evidence-dir>`と表記する）。本文中の`<evidence-dir>`、`<workspace>`、`<project-root>`は公開にあたってローカルの絶対パスを置き換えたプレースホルダーであり、そのまま実行できるパスではない。

---

## 0. Phase 0 — Preflight結果

| 項目 | 結果 | 根拠 |
|---|---|---|
| Repository root | `<workspace>/risklife` | `pwd` |
| Git status | 全ファイル未追跡（`??`）。`.mandala/`はrepository-local excludeで非表示 | `git status --short` |
| HEAD | **なし**（`fatal: ambiguous argument 'HEAD'`） | `git rev-parse HEAD` |
| Go | go1.27.1 darwin/arm64（`go.mod`は`go 1.23`。toolchain取得は発生しない） | `go version`, `go.mod:3` |
| Git | 2.52.0 | `git --version` |
| go.mod | `module github.com/cottondesu/risklife`、依存なし | `go.mod:1` |
| AGENTS.md | **存在しない**（repo・親・HOMEいずれも） | `ls` |
| Source tree | `cmd/risklife`、`internal/{adapters,canonical,cli,engine,fileio,graph,report,rules/risk,snapshot}`。Go約5,600行 | `wc -l` |
| CLI | `help`/`--help`/`-h`、`version`、`simulate`、`analyze` | `internal/cli/cli.go:18-43` |
| 既存テスト | 全PASS（`env -u GOFLAGS GOTOOLCHAIN=local GOPROXY=off go test -count=1 ./...`）。`internal/graph`は単体テストなし | 実行ログ |
| テスト環境依存 | 親環境に`GOFLAGS=-mod=readonly`があると、トップレベル15テスト（adapters 14、cli 1）がFAILする。原因は`go.go:58-60`の意図的なfail-closed警告で製品不具合ではないが、**テストがhermeticではない** | 初回実行ログ |
| ドキュメント | README.md / README.ja.md / docs/ARCHITECTURE.md / docs/SCHEMA.md / docs/VALIDATION.md / docs/IMPLEMENTATION.md / CHANGELOG.md を確認 | — |
| CI / Release workflow | `.github/` なし | `ls .github` |

### Mandala（読み取り専用）

- CLI: PATH上になし → `~/go/bin/mandala`（`mandala v0.3.0`、Skill検証済みの版と一致）。
- `show --json` / `gaps --json` / `status`: goal `Implement and validate RiskLife v0.1.0`、23セル、required `done=20 open=0 na=0`、**required gaps 0**（exit 0）。
- 構造: root 5/8（`adapters`、`cli`、`core`、`oss`、`validation`）。`adapters`の子は **8/8で満杯**、`cli` 7/8、`validation` 3/8、`core`・`oss`は葉で`done`。総数23/72。
- 今回Mandalaは変更していない。不足項目と追加案は §30 を参照。gaps 0は「宣言済みの必須セルが解決済み」という意味にすぎず、汎用ハーネスとしての完成を示すものではない。

## 0.1 確定した決定事項（2026-10-10、実装時）

本書の作成後にユーザーが確定した事項。本文の記述と矛盾する場合はこの節が優先する。

| # | 決定 | 本文への影響 |
|---|---|---|
| D1 | 正式名称は**StateRipple**。ADR-010の条件付き推奨を採用 | 本文中の「RiskLife」は作成時点の名称（史実）として残す |
| D2 | Go Moduleは`github.com/cottondesu/stateripple`、CLIは`stateripple`（`cmd/stateripple`） | §22・Phase Nは実施済み |
| D3 | 作業ディレクトリは`<workspace>/stateripple` | §0・§30のパス`<workspace>/risklife`は作成時点のもの |
| D4 | 旧GitHubリポジトリ`cottondesu/risklife`は**削除済み**。公開先は`https://github.com/cottondesu/stateripple` | §17・§22のrepo renameは不要。旧repoへのpushは禁止 |
| D5 | RiskPropagationV1のRiskは**0..3**を維持 | §6のとおり |
| D6 | attenuationは**0..3**。4以上・負値・非整数・範囲外は不正入力。Riskが最大3である限りattenuation 3以上の寄与はすべて0なので、**0..5などへの値域拡張は行わない**。異なるRisk範囲は新しいRule（またはversioned rule）で検討する | §5.3・§7・ADR-002のとおり。未決事項2は解決 |
| D7 | Mandala Skillで実装を追跡する。既存goal `Implement and validate RiskLife v0.1.0`は再初期化せず保持する | §30の案を容量に合わせて配置（root `generic`・`naming`、`validation`・`cli`への追加） |
| D8 | commit・push・tag・Release公開は行わない | — |
| D9 | テストのhermetic化（B8）はv0.1.0で実施する | 未決事項5は解決 |
| D10 | kindはv0.1.0では必須のまま | 未決事項3は解決 |

実装状況（§28との対応）: B1（Core検証の汎用化）、B2（`internal/adapters/profile.go`の`validateGoProfile`）、B3（Text表示のモード別化）、B4（help文）、B5（GF-01〜15と固定Hash・Goldenの自動テスト、`internal/graph`単体テスト）、B6（ドキュメント）、B8（GOFLAGS/GOEXPERIMENT/GOWORKのテスト隔離）、Phase N（改名）を実施した。検証結果は`docs/VALIDATION.md`の最新記録を参照。

実装時に判明した補足（§6.5の精緻化）: `generation_limit`で停止した不完全な結果では、固定点への到達が確認されていない。最終状態が固定点でない場合、正のRiskを持つセルの最終`causes`が空になりうる（支持元がその後さらに増加した場合）。該当セルの原因はwavesに記録され、結果は常に`incomplete`（exit 2）である。既存Reportのバイト互換を保つため、計算方法は変更せず文書化した。

訂正（独立監査SR-03、2026-10-10）: 上の補足には当初「旧語彙では減衰が一律1のため発生しない」と記載していたが、誤りだった。旧語彙の減衰は一律ではなく、`own_test`は減衰0である。`pkg:A`（seed 3）→`pkg:B`（seed 1）を`prod_import`、`pkg:B`→`test:B`を`own_test`で結び、`max_generations`を1にすると、改名前のバイナリでも`test:B`がrisk 1・最終causes空になり、現行バイナリの出力とバイト一致する。したがってこれは未収束時の既存挙動であり、汎用化で新たに生じたものではない。`generation_limit`で停止した結果のRiskは固定点の下限であり（収束値を上回らないが、下回りうる）、`test_targets`の欠落・過小と最終causesの欠落がありうる。`analyze`では発生しない。`analyze`は常にL=16のGo Profileを生成し、§6.3 P5のとおり4更新以内に収束するためである（Go Profile 4,000件の実測でも`generation_count`の最大は4、未収束0件）。発生には、`simulate`で伝播の深さより小さい`max_generations`を指定する必要がある。

訂正（最終クロスモデル監査CA-04、2026-10-10）: 上の補足は当初「最終状態が固定点ではないため」と断定していたが、不正確だった。`generation_limit`で停止した場合、固定点への到達は確認されていない。最終状態が実際には固定点である場合もあるが（L=1024でのattenuation 0・1025 Nodeの鎖は、全セルが固定点の値に達している）、確認できないため`incomplete`として扱う。「数学的に固定点である」「実装が固定点到達を確認した（`converged`）」「世代上限で確認を打ち切った（`generation_limit`）」「`complete`と判定できる」は別の事柄であり、§6.2の記述（`R_L`が実際には固定点であっても未完了として扱う）が正しい。計算と判定は変更していない。

補足（独立監査SR-07、2026-10-10）: 汎用Coreは、kindが`test`に完全一致しRiskが正のNodeを`test_targets`へ投影し、Text出力をテスト向けの文言に切り替える（`internal/rules/risk/run.go`、`internal/report/text.go`）。これは`simulate`の非ソフトウェアGraphにも適用される。v0.1.0では旧Reportとのバイト互換を優先して挙動を変えず、README・SCHEMA・ARCHITECTUREにこの例外を明記した。Coreからの責務分離は§27の「ドメイン固有のResult Adapter（test_targets等の分離）」としてv0.2.0で設計する。

---

# Part A — Product Specification

## 1. Executive Summary

1. **CoreエンジンはすでにGit・Goから独立している。** `simulate`は`.git`なし、HEADなし、Git/Go実行ファイルなし（`PATH`を空にした状態）で成功し、Golden JSONとバイト一致した（GF-01〜04、GF-11、GF-12）。`engine.Rule[S comparable]`は任意の状態型に対応した汎用設計である（`internal/engine/engine.go:6-53`）。
2. **ただし入力Schemaの語彙がGoに固定されている。** Node kindは`package|test`、Edge kindは`prod_import|own_test|test_import`に限られ（`internal/graph/graph.go:54-86`）、attenuationはEdge kindから一意に決まる（`internal/snapshot/snapshot.go:57-65`）。さらに「test nodeは出力Edgeを持てない」という制約がある（`graph.go:72-74`）。非ソフトウェアのGraphも`package`/`prod_import`を流用すれば計算できるが、自然な表現では検証で拒否される（GF-05/06はPARTIAL）。
3. **最小修正方針:** Snapshot Schema v1の**フィールド形状を変えず**、検証条件だけを緩める。kindは自由ラベル、attenuationは0..3の明示値とする。Go固有の制約はGo Adapter側の検証に移す。これまで有効だった入力は正規化後のバイト列もHashも同一になるため、**既存のGolden・Hash・Report形状はいずれも変わらない**（§14で証明）。
4. **RiskPropagationV1は仕様式とコードが一致する。** 式から独立に書いたPython参照モデルとバイナリを、ランダムGraph 800件で差分比較し、最終状態・generation_count・waves・stop_reason・exit codeがすべて一致した。現行語彙では`generation_count ≤ 4`が成り立つ（example.jsonで上界4に到達する）。attenuation 0を一般化すると上界は`max(1,|V|)`になる（§6）。
5. **Completenessの4層**（Input Validity、Adapter Completeness、Simulation Completion、External-World Coverage）は、既存のexit code・`status`・`stop_reason`・`build_context`から一意に判別できる（§8の定理）。v0.1.0でReport形状を変える必要はない。
6. **名称:** 方向性は **B. Generic State Propagation Harness**。GitHub衝突確認（2026-10-09実施）では事前候補11件のうち**StateTideとRuleLatticeを除外**した（事前検索が見落としていたハイフン区切りの同名repoが存在した）。最推奨は **StateRipple**、次点は **StateCascade**。判断は **RENAME BEFORE v0.1.0（条件付き推奨）**。名称の最終決定はユーザーが行う。
7. **判定:** Architecture = **MINOR REFACTOR REQUIRED**、Naming = **RENAME BEFORE v0.1.0（ユーザー承認と追加確認を条件とする）**、Release Specification = **SPECIFICATION READY WITH OPEN DECISIONS**。

## 2. Product Vision

> 任意の有向グラフとSeedを受け取り、セル（Node）の状態・リスク・影響を世代単位の同期更新で決定論的に伝播させる、オフライン動作の汎用ハーネス。

- Coreの入力は**Canonical Snapshot**だけである。Git/Go/Test Impactは、Snapshotを生成したりドメイン固有の出力を付加したりする**任意のAdapter**として扱う。
- 「ライフゲーム型」は、**セルの状態を世代ごとに同期更新する構造**を指す。現行RuleはConwayのB3/S23ではない。誕生・死滅（状態の減少）はなく、単調増加する最大値伝播である（`docs/ARCHITECTURE.md:49`で既に明記）。名称や説明文でConway型の意味論を保証してはならない。
- 不変条件の優先順位は依頼書 §2.1 に従う（Generic → Correctness → Determinism → …）。

## 3. AS-IS Architecture

### 3.1 責務

| 要素 | 実装 | 汎用性 |
|---|---|---|
| Graph / Node / Edge | `internal/graph/graph.go:15-25` 構造体、`:42-102` Normalize | **構造は汎用だが、検証がGo語彙に固定** |
| Seed | `internal/snapshot/snapshot.go:21-25`（`id, risk, reason`） | 汎用（Gitの概念に依存しない） |
| Snapshot | `snapshot.go:27-34`、Read `read.go:14-72` | 形状は汎用、検証は部分的にGo固有（`snapshot.go:57-65`） |
| Rule | `internal/rules/risk/rule.go:20-26`（Next）、`:28-45`（causes） | 汎用な最大値伝播 |
| State | `[]int`（0..3） | 汎用 |
| Engine | `internal/engine/engine.go:6-53` `Rule[S comparable]`、同期更新、limit 1..1024 | **完全に汎用** |
| Validation | `graph.Normalize` + `snapshot.Normalize` + `snapshot.Read` | 構文・参照・上限の検証は汎用、語彙の検証はGo固有 |
| Canonical Encoder | `internal/canonical/json.go:5-11`（`encoding/json` + LF） | 完全に汎用 |
| Hash | `snapshot.go:97-107`（正規化Snapshotの正準JSONのSHA-256） | 完全に汎用。プロジェクト名を含まない |
| Trace | `run.go:45-77`（waves）、`rule.go:28-45`（causes） | 汎用 |
| Report | `internal/report/model.go:50-64`、`json.go`、`text.go` | 汎用部分とドメイン部分が同じ型に混在（§11） |
| Git Adapter | `internal/adapters/git.go`、`head.go`、`embed.go`、`git_requirement.go`、`command.go` | Git固有（`analyze`の中でのみ実行） |
| Go Adapter | `internal/adapters/go.go`、`inputs.go`、`module_inputs.go`、`analyze.go:151-239`（classify） | Go固有 |
| CLI | `internal/cli/cli.go`、`fileio.go`、`internal/fileio/` | 汎用。ただしhelpとText表示にソフトウェア用語が含まれる |

### 3.2 Coupling Audit

| # | 確認項目 | 結果 | 根拠 |
|---|---|---|---|
| C1 | CoreがGit固有型を使っていないか | **使っていない**。`graph`・`engine`・`rules/risk`・`snapshot`・`canonical`はadaptersをimportしない | import文。`docs/ARCHITECTURE.md:20` |
| C2 | CoreがGo packageを前提にしていないか | **前提にしている（語彙レベル）**。kindは`package|test`のみ | `graph.go:54-58` |
| C3 | Node IDに`pkg:`/`test:`が必須か | **必須ではない**。IDは任意の非空UTF-8（≤4096B）。`pkg:`/`test:`はGo Adapterの慣習 | `graph.go:40`、`go.go:132-134`、fixture GF05（`東京駅`、`a b/../x`、`\u0007bell`を受理） |
| C4 | Edge TypeがGo依存関係に固定されていないか | **固定されている**。`prod_import|own_test|test_import`以外は拒否、Edge kindごとに許可されるtarget kindも固定 | `graph.go:75-86`、fixture（`ships_to`は拒否） |
| C5 | Attenuationの独立性 | **独立していない**。`own_test`=0、それ以外=1 に固定 | `snapshot.go:57-65`、fixture（attenuation 0/2は拒否） |
| C6 | SeedsがGit変更情報に結合していないか | **結合していない**。Seedは`id, risk, reason`のみ。Git変更はclassifyがSeedへ変換する | `analyze.go:201-237` |
| C7 | Test Targetsが汎用Resultに強く結合していないか | **中程度に結合**。`risk.Run`が`kind=="test"`のNodeから`test_targets`を生成し、Report型に必須フィールドとして含まれる | `run.go:81-83`、`model.go:57` |
| C8 | AdapterのcompleteとEngineのcompleteの混同 | **フィールドとしては分離されているが、`status`で合成されている**。`stop_reason`はEngine専用、`status`はEngineとAdapterのAND | `run.go:40-44`、`cli.go:100-102`、`docs/SCHEMA.md:193,213` |
| C9 | CLI起動時の不要なGit/Go初期化 | **ない**。`requireGit`は`adapters.Analyze`の内部（`analyze.go:34`）でのみ呼ばれる。adapters/cli/cmdにpackage-level変数や`init()`はない | `cli.go:76-88`、grep結果、fixture GF11 |
| C10 | 責務の混在 | `risk.Run`が「正規化・Hash・Engine実行・Trace生成・Test Target投影・Report組み立て」を1関数で担う。`rules/risk`が`report`に依存している | `run.go:11-92`、`rule.go:5` |
| C11 | Text表示のドメイン語 | simulateでも「Changed files: none (snapshot simulations have no Git metadata)」「Recommended test targets」「Selected tests have not been executed」を出力する | `report/text.go:20-33,58` |
| C12 | Git AdapterとGo Adapterの結合 | 単一の`adapters.Analyze`内で密に結合している（classifyがGo package情報とGit変更を同時に使う） | `analyze.go:24-133` |

### 3.3 分類

| 分類 | 対象 |
|---|---|
| 1. 完全に汎用 | engine、canonical、Hash、Seed、Trace（waves/causes）、snapshot.Readの構文・Unicode・上限・未知フィールド検証、fileio、`--out`安全出力、exit codeの骨格 |
| 2. Git依存 | git.go、head.go、embed.go、git_requirement.go、command.goのGit経路、`report.FileChange` |
| 3. Go依存 | go.go、inputs.go、module_inputs.go、classify、`report.BuildContext`、**graph/snapshotのkind・attenuation検証（Core内にある）** |
| 4. ソフトウェア開発用途に特化 | `test_targets`、Text表示の文言、help文、README冒頭（「Go code changesのためのエンジン」） |
| 5. 責務混在 | C7、C8、C10、C12 |
| 6. 変更すべき | C2/C4/C5のCore検証 → Go Adapterへ移す（v0.1.0）。C11 → Text表示をモード別にする（v0.1.0）。C10/C12 → v0.2.0で分離 |
| 7. 現状維持 | engine、Hash、Canonical、Read、Trace、Report JSON形状、exit code、`--out`、Git/Goの安全制御 |

## 4. TO-BE Architecture

```text
                     CLI (simulate / analyze)
                         |
       +-----------------+----------------------+
       |                 |                      |
 Snapshot File     [v0.2.0] stdin /       Git + Go Adapter (analyze)
 (simulate)        Explicit Graph/Seeds   └─ Go Profile Validation  ← v0.1.0で新設
       |                 |                      |
       +-----------------+----------------------+
                         |
            snapshot.Read / snapshot.Normalize      ← v0.1.0: 汎用検証のみ
                         |
              Canonical Snapshot (+ input_sha256)
                         |
                engine.Run[S] (同期更新)
                         |
                 RiskPropagationV1 (rule)
                         |
          Generic Result (cells, waves, generation_count, stop_reason)
                         |
     Report v1 = Generic部 + Domain部(test_targets / changes / build_context …)
                         |
                JSON (canonical) / Text (モード別表示)  ← v0.1.0: Text表示を修正
```

既存実装との対応: Snapshot File、Git+Go Adapter、Normalize、Canonical、Engine、Rule、Report、JSON/Textは**実装済み**。Go Profile Validationの新設、Normalizeの汎用化、Text表示の修正は**v0.1.0で必要**。stdin・Explicit入力・Adapter分離・Result Adapterは**将来案**。

## 5. Generic Core Contract

### 5.1 概念型

```text
Snapshot  S = (schema_version = 1, ruleset = "risk-propagation-v1", L, V, E, Σ)
  L ∈ [1, 1024]                 max_generations（JSONで省略時は16に正規化）
Node      v = (id: Str, kind: Str)
Edge      e = (from: Str, to: Str, kind: Str, attenuation: a ∈ {0,1,2,3})
Seed      s = (id: Str, risk: r ∈ {0,1,2,3}, reason: Str)
Str       = { x | x は妥当なUTF-8、1 ≤ |x|_bytes ≤ 4096 }
```

### 5.2 Node Contract

| 観点 | 規定（TO-BE v0.1.0） | AS-IS差分 |
|---|---|---|
| ID | `Str`。区切り文字・パス・prefixとして解釈しない（`a b/../x`もそのまま扱う） | 同じ |
| 許容文字 | 任意のUnicodeスカラー値（制御文字・NULを含む）。不正UTF-8と孤立surrogateのJSON escapeは拒否 | 同じ（`read.go:22-24`、`unicode.go`） |
| Unicode正規化 | **しない**。NFC形とNFD形は別のIDになる（fixture nfc-nfdで2 Nodeとして受理） | 同じ |
| 重複 | 同一IDは、kindが同じでも異なっても **REJECT** | 同じ（`graph.go:59-61`） |
| ソート | Goの文字列バイト比較で昇順（UTF-8ではコードポイント順と同じ） | 同じ |
| kind | **`Str`の自由ラベル**（意味はAdapterまたは利用者が定める） | AS-ISは`package|test`のみ |
| メタデータ | v1にはない（未知フィールドはREJECT） | 同じ |
| 空文字列 | REJECT | 同じ |
| 上限 | 20,000 Node | 同じ |

### 5.3 Edge Contract

| 観点 | 規定（TO-BE v0.1.0） | AS-IS差分 |
|---|---|---|
| From/To | 既存Node ID。存在しない場合はREJECT | 同じ |
| Direction | `from → to` は「fromの状態がtoへ影響する」という意味 | 同じ |
| Attenuation | 整数 **0..3**。kindとは独立。範囲外（負値、4以上、int範囲外）はREJECT | AS-ISは`own_test`=0、それ以外=1に固定 |
| Duplicate | `(from,to,kind,attenuation)`が完全一致する重複は **NORMALIZE（統合）** | 同じ |
| 同一端点の矛盾 | `(from,to)`が同じでkindまたはattenuationが異なる場合は **REJECT**（ordered pairごとにEdgeは高々1本） | 同じ（`graph.go:87-98`） |
| Self-loop | **ACCEPT**。a ≥ 0 なので値には影響しない（§6 P1）。a=0の場合は最終causesに自己参照として現れうる（§6.5） | AS-ISはa=1のみ可能 |
| Cycle | ACCEPT | 同じ |
| Edge kind | **`Str`の自由ラベル**。propagated causeの`reason`として使う | AS-ISは3種の列挙 |
| 出力Edge制約 | なし | AS-ISは「test nodeは出力Edgeを持てない」（Go Profileへ移す） |
| target kind制約 | なし | AS-ISは`prod_import→package`、`*test*→test`（Go Profileへ移す） |
| 上限 | 統合前で200,000本 | 同じ |

### 5.4 Seed Contract

| 観点 | 規定 |
|---|---|
| Target | 既存Node ID。存在しない場合はREJECT（`snapshot.go:68-70`） |
| Initial state | `σ(v) = max({0} ∪ {s.risk | s.id = v})` |
| Duplicate | `(id,risk,reason)`が完全一致する重複はNORMALIZE。同じidでriskまたはreasonが異なるSeedは保持し、最大値を使う |
| Missing seeds | `seeds: []` は有効（全Node 0、1世代で収束）。フィールド自体の省略はREJECT |
| Invalid risk | 0..3の範囲外、非整数リテラル（`1.0`、`1e0`）、文字列、int範囲外はREJECT。**risk 0はACCEPT**（Seed一覧とHashには残るがwaveは生じない） |
| reason | `Str`（空はREJECT） |
| Ordering | `(id↑, risk↓, reason↑)` |
| 生成元 | Git変更はSeed生成手段の一つにすぎない（`analyze.go:205,234`） |
| 上限 | 統合前で100,000件 |

### 5.5 不変条件（Normalize後）

- **I1** Nodeはid昇順で、重複がない。
- **I2** Edgeは`(from,to,kind,attenuation)`昇順で、ordered pair `(from,to)`が一意である。
- **I3** Seedは`(id↑,risk↓,reason↑)`順で、重複がない。
- **I4** EdgeとSeedの参照先はすべて存在する。
- **I5** 冪等性: `Normalize(Normalize(S)) = Normalize(S)`（`snapshot.Hash`が再正規化に依存しているため必須）。
- **I6** `input_sha256 = SHA256(canonical(Normalize(S)))`。
- **I7** 順序非依存性: V・E・Σの列挙順、JSONの空白、オブジェクトのキー順、完全一致する重複は、正規化結果に影響しない（fixture GF09とGF15で実証）。

### 5.6 エラー条件と実装の対応

| エラー分類 | 例 | 実装箇所 |
|---|---|---|
| E-JSON | 構文不正、末尾に余分な値、null、重複キー、深さ>16 | `read.go:25-32,95-157` |
| E-UNICODE | 不正UTF-8、孤立surrogate | `read.go:22-24`、`unicode.go` |
| E-SHAPE | 必須フィールドの欠落、未知フィールド、大文字小文字違いの別名、型不一致 | `read.go:33-70,74-93` |
| E-LIMIT | 16MiB超、配列上限超、文字列長超 | `read.go:15-21,126-151`、`graph.go:43-45` |
| E-VERSION | schema_version≠1、ruleset不一致、max_generations範囲外 | `snapshot.go:37-45` |
| E-DUP | Node ID重複、矛盾Edge | `graph.go:59-61,87-98` |
| E-REF | Edge端点・Seed対象が存在しない | `graph.go:67-71`、`snapshot.go:68-70` |
| E-DOMAIN | risk範囲外、attenuation範囲外、空のreason | `snapshot.go:57-65,71-76` |

いずれもexit 1、Reportなし、stderrに`risklife: read snapshot: …`を出す。エラー文言は現在は自由形式である。安定したエラーコード（構造化Diagnostics）はv0.2.0候補とする。

## 6. Formal RiskPropagationV1 Specification

### 6.1 定義

状態域を `D = {0,1,2,3}`、状態を `R : V → D` とする。`In(v) = {(u,a) | (u,v,·,a) ∈ E}`、空集合のmaxは0とする。

```text
R_0(v)     = σ(v)
R_{k+1}(v) = max( R_k(v), σ(v), max_{(u,a) ∈ In(v)} max(0, R_k(u) − a) )
```

依頼書の式 `R_next(v) = max(R_current(v), Seed(v), max[u→v](max(0, R_current(u) − attenuation(u,v))))` と同一である。

**コードとの一致:** `rule.go:20-26` は `next = max(prev[i], initial[i])` を計算し、続けて `max(next, prev[u] − a)` を取る。`max(0,·)`がなくても等価になる理由は、`next ≥ prev[i] ≥ 0` なので負の項がmaxに影響しないためである（補題: 全kで`R_k ≥ 0`）。`initial[i]`は`run.go:26-30`でSeedの最大値として計算される。
**実証:** 式から独立に実装したPython参照モデルとバイナリを、ランダムGraph 800件（Node 0〜12、cycle・self-loop・同点・limit 1〜16を含む）で比較し、最終状態・generation_count・各waveの変化Node集合・stop_reason・exit codeが**全件一致**した（`<evidence-dir>/diff/ref.py`）。

### 6.2 同期更新と世代

- Generation 0は `R_0 = σ`。更新kは、凍結した `R_{k−1}` だけから全Nodeを計算し、全体を一括で置き換える（`engine.go:34-44`）。
- `generation_count` = 実行した更新回数。最後の「変化なし確認」を含み、generation 0は含まない（`engine.go:43`）。
- 更新kで変化がなければ `converged`、`stop_reason = "converged"` とする。
- **Generation limit L:** 更新Lでまだ変化があれば `stop_reason = "generation_limit"`、`status = "incomplete"`、警告を出してexit 2。`R_L`が実際には固定点であっても、確認更新をしていないので未完了として扱う（fail-closed。`run.go:40-44`）。

### 6.3 性質と証明

- **P1 値域:** a ≥ 0 かつ σ ≤ 3 なので `R_k(u) − a ≤ 3`。maxが`R_k(v) ≥ 0`を含むので下限は0。**aが負になると値域を超えるため、attenuationの下限0は必須の検証条件である。**
- **P2 単調性:** 項`R_k(v)`を含むので `R_{k+1} ≥ R_k`（inflationary）。演算子は順序保存的である（`R ≤ R' ⇒ F(R) ≤ F(R')`）。
- **P3 停止性:** 状態空間 `D^V` は有限で高さは `3|V|`。厳密な増加は高々`3|V|`回なので、必ず固定点に到達する。
- **P4 経路表現（より強い上界）:** 帰納法により `R_k(v) = max(0, max{ σ(start(w)) − A(w) | w は v で終わる長さ ≤ k のwalk })`（A(w)はattenuationの総和）。a ≥ 0 なのでcycleを除いても値は下がらず、最適walkは単純路に取れる。よって `R_{|V|−1} = R*` で、**`generation_count ≤ max(1, |V|)`**。
- **P5 現行語彙での上界:** 正の寄与を持つwalkでは`A(w) ≤ 2`。AS-ISではa=1のEdgeが高々2本で、a=0は`own_test`（出力Edgeを持たないtest nodeへ入るEdge）だけなので、walk長は3以下になり **`generation_count ≤ 4`**。example.jsonが上界4を達成する（`docs/SCHEMA.md:231`）。差分試験ではlimit 16のとき最大3だった。**帰結:** analyzeの出力（Go AdapterはL=16で生成する。`go.go:37`）と、L ≥ 4のAS-ISで有効なSnapshotでは、`generation_limit`は発生しない。Lを1〜3に明示したSnapshotでは、旧語彙でも`generation_limit`になりうる（§0.1の訂正を参照。`TestSimulateGenerationLimitExitCode`はL=1の旧語彙Snapshotでexit 2を確認している）。
- **P6 一般化後の上界:** a ∈ 0..3を許すと、a=0の鎖は深さ分の世代を要する。Pythonモデルで、長さn=1,5,50,1100の鎖がそれぞれ`generation_count = n`になることと、ランダムGraph 3000件で「固定点 = 単純路の最大値」「`generation_count ≤ max(1,|V|)`」が成り立つことを確認した（`<evidence-dir>/diff/bound.py`）。L ≤ 1024 かつ |V| ≤ 20,000 なので、**深さ1024以上のa=0鎖は`generation_limit`で未完了になる**。これは誤った完了ではなく、明示的な未完了である。
- **P7 最小固定点:** `G(R) = max(σ, Prop(R))` とすると `R_k = G^{k+1}(⊥)`。Kleene反復なので `R*` は σ 以上の **最小固定点**であり、評価順序に依存しない（最終状態の一意性）。ただしwaveの構成とgeneration_countは同期スケジュールに依存し、その規約はEngineが固定する。

### 6.4 境界ケース

| ケース | 規定（実測） |
|---|---|
| 複数Seed | 最大値を採用し、Seed記録はすべて保持（GF08） |
| 同点の原因 | maxを与える原因をすべて列挙し、`(source, reason, edge)`順に並べる（GF08: cの原因はaとbの2件） |
| Cycle | 正常に収束（GF07: a→b→c→a、generation_count 3） |
| Self-loop | 値に影響しない（fixture self-loop-only） |
| 非連結 | 成分ごとに独立（GF07のz=0） |
| 空Graph | generation_count 1、converged、complete（`cli-empty.golden.json`） |
| risk 0のSeed | 受理。wave 0には現れない |
| L到達 | `generation_limit`、incomplete、exit 2（fixture limit-hit） |

### 6.5 Traceの意味論

- **wave 0:** σ(v) > 0 のNodeについて、原因は risk = σ(v) のSeed。
- **wave g ≥ 1:** `R_g(v) > R_{g−1}(v)` となったNode。原因は `R_{g−1}(u) − a = R_g(v)` を満たすEdge。`R_g(v) > σ(v)` なので、g ≥ 1でSeedが原因に入ることはない。
- **起源の整礎性:** g ≥ 1で原因となるuは、g−1の時点ですでに `R_g(v) + a ≥ R_g(v)` の値に達し、vより先に活性化している。waveの原因を遡ると必ずgeneration 0のSeedで止まる（a=0でも同じ）。
- **最終causes:** `R*(v) > 0` の場合に、`R*(v)`を支える全Seed・全Edgeを示す。これは「起源」ではなく「**支持**」である。AS-ISでは循環しないが、一般化後はa=0のcycleやself-loopで循環した支持が表れうる。この意味論をSCHEMAに明記することがv0.1.0の要件である。

### 6.6 将来Ruleの拡張境界（評価）

- `engine.Rule[S comparable]{ Next(i, previous []S) S }` は、任意の比較可能な状態と近傍関数（隣接情報はRuleが保持する）に対応している。**近傍数に依存するRule、複数状態のRule、状態減少を許すRule、Conway型Rule**も、このインターフェースで表現できる（`engine_test.go`のbool Ruleで汎用性を実証済み）。
- 現状で不足しているもの（いずれも将来対応でよい）:
  1. **停止の意味論:** 非単調Ruleは振動して固定点に到達しないことがある。Engineは「変化なし」か「limit到達」でしか止まらないため、Conway型の「N世代を実行すること自体が目的」の場合も現行ではincompleteになる。Rulesetごとに完了基準（`fixed_point`か`fixed_generations`か）を宣言し、必要なら状態履歴Hashで周期を検出する仕組みが要る。
  2. **Snapshot Schemaは`risk:int`前提:** 別の状態型には別ruleset、必要ならschema versionが必要になる。
  3. **Ruleset dispatch:** `cli.go:93`が`risk.Run`を直接呼んでいる。2つ目のRuleを追加する時点で、`ruleset`による一箇所のswitchを導入すれば足りる。
- **結論:** v0.1.0は「RiskPropagationV1をengine.Ruleの一実装として維持する」だけで十分である。プラグイン機構や抽象Rule APIは導入しない（過剰な抽象化を避ける）。§6.3 P3〜P7は**RiskPropagationV1に固有の保証**であり、Engine一般の保証ではない（`docs/ARCHITECTURE.md:57`の既存記述を維持する）。

## 7. Canonical Snapshot Schema

### 7.1 Schema v1（TO-BE。フィールド形状はAS-ISと同一）

| Field | 必須 | 型と規定 |
|---|---|---|
| `schema_version` | 必須 | 整数1 |
| `ruleset` | 必須 | `"risk-propagation-v1"` |
| `max_generations` | 任意 | 1..1024、省略時は16（正規化後はHashに明示的に含まれる） |
| `nodes[]` | 必須 | `{id: Str, kind: Str}` |
| `edges[]` | 必須 | `{from: Str, to: Str, kind: Str, attenuation: 0..3}` |
| `seeds[]` | 必須 | `{id: Str, risk: 0..3, reason: Str}` |
| Rule Parameters | — | v1では`max_generations`だけ。新しいフィールドは追加しない |
| Optional Metadata | — | v1では追加しない（未知フィールドはREJECTする現行方針を維持）。追加する場合はHashへの算入・除外の方針とversionを同時に決める（v0.2.0以降） |

存在しないフィールドを必須化しない。`kind`は必須のまま残し、語彙だけを自由にする。kindを省略可能にする案（既定値を補う）は、Hashの既定値規約を伴うためv0.2.0で検討する。

### 7.2 Validation Matrix

| 入力 | 判定 | 理由・実測 |
|---|---|---|
| Invalid JSON | REJECT | exit 1 `unexpected end of JSON input` |
| Empty Graph | ACCEPT | complete、gen 1。Golden |
| Duplicate Nodes | REJECT | kindが異なっても曖昧なので拒否 |
| Duplicate Edges（完全一致） | NORMALIZE | 統合しHash不変（GF09） |
| Duplicate Edges（同一端点で属性違い） | REJECT | 伝播係数が曖昧になるため |
| Unknown Node References | REJECT | `edge endpoint not found`、`seed node not found` |
| Invalid Risk | REJECT | -1、4、`1.0`、`1e0`、`"1"`、`99999999999999999999` |
| Invalid Attenuation | REJECT | TO-BE: 0..3の範囲外。int64範囲外はdecode時点で拒否 |
| Invalid Unicode | REJECT | `\xff`、`\ud800` |
| Unicode正規化の差 | ACCEPT（別ID） | 暗黙の同一視をしない |
| Unknown Fields | REJECT | root・nested、`ID`のような大文字小文字の別名も拒否 |
| Excessive Input Size | REJECT | 16MiB+1、20,001 Node、上限超の配列、4097Bの文字列 |
| Arithmetic Overflow | 構造的に発生しない | 値0..3・attenuation 0..3を事前に検証し、整数はdecode時点で範囲を確認 |
| Unsupported Schema Version / ruleset | REJECT | `unsupported schema_version 2`、`unsupported ruleset "conway"` |
| max_generations 省略 / 0 / 1025 | NORMALIZE(16) / REJECT / REJECT | 0を「既定値の要求」と解釈しない |
| null、重複キー、末尾の値、深さ>16 | REJECT | — |
| Self-loop、Cycle、非連結 | ACCEPT | §6.4 |
| Node/Edge kind（非Go語彙） | AS-IS REJECT → **TO-BE ACCEPT** | ADR-002 |

### 7.3 互換方式と版管理方式の比較

| 案 | 内容 | Hash/Golden | 評価 |
|---|---|---|---|
| **α. v1の緩和（superset）** | 形状は同一で、検証だけを緩める | **不変** | **採用**。未リリースで、公開repoも空なので外部利用者がいない。既存入力のバイト列・Hash・Reportが同一 |
| β. schema_version 2を新設 | v1（Go用）とv2（汎用）を併存 | v1は不変 | 初回リリースで2つのSchemaを抱える意味がない |
| γ. 新しいruleset ID | 例: `state-propagation-v1` | **全Hashが変わる**（rulesetはHash対象） | Golden・Hashを無断で変える案なので不採用 |

## 8. Completeness and Error Contract

| 層 | 意味 | 判定する主体 | Report / exit表現 |
|---|---|---|---|
| 1. Input Validity | JSONとGraphが形式的に妥当か | `snapshot.Read`/`Normalize`、analyzeでは収集後の`Normalize`（`analyze.go:123`） | 不正 → **exit 1**、Reportなし |
| 2. Adapter Completeness | 外部資源を必要な範囲で取得・解析できたか | Adapterの`Metadata.Incomplete`（`analyze.go:22`） | 不完全 → `status:"incomplete"`、`warnings`/`unknown_changes`、**exit 2**。収集自体が不能ならexit 1。**simulateには適用しない（Adapterがないので対象外）** |
| 3. Simulation Completion | 与えられたGraphとRuleで計算が正常に終わったか | Engine | `stop_reason`（`converged`/`generation_limit`）。limit到達 → incomplete、**exit 2** |
| 4. External-World Coverage | 現実の依存がすべて入力に含まれているか | **どの層も判定しない** | 表現しない。`complete`は「与えられた入力に対する計算が完了し、Adapterが既知の欠落を検出しなかった」という意味にすぎない |

**状態関数:** `status = complete ⇔ stop_reason = converged ∧ ¬AdapterIncomplete`。exit = エラーなら1、`status=incomplete`なら2、それ以外は0。出力書き込みの失敗は1（公開後のcleanup失敗も1。ファイルは存在しうる。`fileio.go:70-79`）。

**判別可能性の定理（v0.1.0でReport形状を変えなくてよい根拠）:**
1. simulateでは`Metadata`がゼロ値（`cli.go:73,98-102`）なので、`status=incomplete ⇔ generation_limit`。
2. analyzeでは、生成されるSnapshotがGo語彙でL=16なので、P5により`stop_reason`は常に`converged`。よって`status=incomplete ⇔ Adapter incomplete`。
3. モードは`build_context`で判別できる（simulateではnull、analyzeでReportが出るときは常に非null。`go.go:57`はGo環境を取得できた後にのみReportへ進む）。
4. したがって、層1は(exit 1)、層2は(analyze ∧ status incomplete)、層3は(stop_reason)で一意に判別できる。

**意味の変更は不要。** ただし、将来のAdapter（例: Changed Files + 汎用Graph）では層2と層3が同時に起こりうる。その時点でReportに明示的な`completeness`オブジェクトを導入する（report schema v2、v0.2.0候補）。Snapshot入力にGit由来の完全性制約を適用しないことは、`cli.go:76-81`がsimulateでadaptersを呼ばないことにより保証されている。

## 9. Input Adapter Contract

| 項目 | 最小契約（概念。v0.1.0では新しいinterfaceを作らない） | 現行の対応物 |
|---|---|---|
| Graph | Core検証と**Adapter Profile検証**の両方を通るSnapshot | `snapshot.Snapshot`、`analyze.go:123` |
| Seeds | Snapshot内の`seeds`。reasonに由来を記録する（例: `source_changed:<path>`） | `analyze.go:205,234` |
| Diagnostics | 整列・重複除去済みの`warnings`、`unknown_changes` | `analyze.go:127-131` |
| Completeness | 既知の欠落が1つでもあれば`Incomplete=true`。欠落を知りながら`false`を返してはならない | `Metadata.warn` |
| Provenance | ドメイン固有の`changes`（Git）と`build_context`（Go）。汎用のadapter名・版はv0.2.0 | `model.go:35-49` |
| Error | 有用なGraphを作れない場合はerror（exit 1） | `Analyze`の戻り値 |

- **Git Adapter:** 変更検出、base解決、rename無効（削除＋追加として扱う）、gitlink、former embed、欠落blob・partial cloneの検出、暗黙fetchの禁止、Git 2.45.0以上（`--no-lazy-fetch`の実地検査）。Gitの最低版は**Adapterの要件であってCoreの要件ではない**（`git_requirement.go:10`はAnalyze内でのみ使われる）。
- **Go Adapter:** package graph、import、test関係、build context、Go固有の診断、offline環境ポリシー。**v0.1.0で追加:** これまでCoreが行っていたGo語彙の検証を、`validateGoProfile`（仮称）としてAdapter内に置く。検証内容は、kind ∈ {package,test}、`prod_import`: package→package（a=1）、`test_import`: package→test（a=1）、`own_test`: package→test（a=0）、test nodeは出力Edgeを持たない、の5点で、AS-ISとまったく同じにする。これによりanalyzeのfail-closed性は維持される。
- Git AdapterとGo Adapterの分離はv0.2.0（Changed Files Adapterを導入する時点）で行う。

## 10. CLI Specification

| コマンド | 契約 | 状態 |
|---|---|---|
| `help`/`--help`/`-h` | stdout、exit 0。Git/Goを起動しない | 実装済み（GF11） |
| `version` | `risklife v0.1.0`、引数があればexit 1 | 実装済み |
| 引数なし | helpをstderrに出してexit 1。**解析を開始しない** | 実装済み（GF11 noargs） |
| `simulate --input FILE [--format json\|text] [--out PATH]` | **Coreの汎用実行経路**。Git、Go、`.git`、HEAD、ネットワークのいずれも不要。入力はregular fileに限る | 実装済み（GF01〜04） |
| `analyze [--base REV] [--format] [--out]` | 任意のGit+Go Adapter。明示的に指定した場合だけGit/Goを実行する | 実装済み（GF13/14） |
| Exit codes | 0 complete / 1 入力・使用法・実行・出力のエラー / 2 incomplete（simulateはlimit到達、analyzeはAdapterの欠落） | 実装済み。**help文の「2 incomplete analysis」は「incomplete result」に改める（v0.1.0）** |

**v0.1.0で必要な変更:** help文（`cli.go:18`）を、simulateを中心とする汎用的な説明に改め、exit 2の文言を両モードに対応させる。新しいコマンドは**追加しない**（Explicit Inputは現行のsimulateで満たせる）。

| 将来候補 | 判定 | 理由 |
|---|---|---|
| stdin（`--input -`） | v0.2.0 | 現在は`fileio.OpenRegular`がregular fileを要求している。FIFO・identityの安全性契約を再設計する必要がある |
| Explicit Graph / Seeds（ファイル分割、Seedの上書き） | v0.2.0 | Snapshot 1ファイルで代替できる |
| Local Source / Changed Files Adapter | v0.2.0 | Adapterの分離が前提 |
| Git Mode Selection（`analyze go`など） | v0.2.0以降 | Adapterが1種類のうちは不要 |

## 11. Generic Output Contract

| 区分 | フィールド | v0.1.0での扱い |
|---|---|---|
| Generic | `schema_version`、`ruleset`、`status`、`input_sha256`、`generation_count`、`cells[] {id, kind, risk, seeds, first_active_generation, causes}`、`waves[]`、`warnings`、`stop_reason` | 維持 |
| Domain（Test Impact投影） | `test_targets`（kind=="test"かつrisk>0のNode） | **維持**（互換のため）。「kind `test`の正のNodeを投影した結果」と定義する。汎用Graphでは通常`[]` |
| Domain（Git） | `changes`、`unknown_changes` | 維持。simulateでは`[]` |
| Domain（Go） | `build_context` | 維持。simulateではnull |

- Final Node States = `cells[].risk`、Generation Count = `generation_count`、Convergence = `stop_reason`、Trace = `waves` + `cells[].causes`、Diagnostics = `warnings`、Canonical Representation = 正準JSON、Hash = `input_sha256`。
- **Test Targetsは汎用計算の必須要件ではない**（計算後の投影にすぎない）。Report型からの分離（Result Adapter化）はReport形状を変えるので、v0.2.0（report schema v2）とする。
- **Text表示（v0.1.0で必要）:** simulateで「Changed files」「Recommended test targets」「Selected tests have not been executed」を出さない。Git/Testの節はanalyzeモード（`build_context != null`）か、`test_targets`が空でないときだけ表示する。末尾の注意書きは汎用文（例: 「Risk is a propagated priority, not a probability.」）とanalyze用の文に分ける。Text出力はHashにもJSON Goldenにも関係しない。

## 12. Determinism Contract

| 対象 | 規約 |
|---|---|
| Node | id昇順（バイト比較） |
| Edge | `(from,to,kind,attenuation)`昇順。完全一致の重複は統合 |
| Seed | `(id↑,risk↓,reason↑)`。完全一致の重複は統合 |
| 同点の原因 | `source → reason → edge(from,to,kind,attenuation)`。edgeなしが先 |
| Trace | waveはgeneration昇順、wave内の変化はid昇順 |
| Warning / unknown_changes | 昇順＋重複除去 |
| Canonical JSON | `encoding/json`のcompact出力（HTML escapeあり）＋LF 1個。フィールド順は構造体の定義順。RFC 8785ではない |
| Snapshot SHA-256 | 正規化Snapshotの正準JSONのSHA-256。対象は`schema_version, ruleset, max_generations, nodes, edges, seeds`。**プロジェクト名、時刻、パス、Report、Adapterのメタデータは含まない** |
| Generation / 最終状態 | 正規化入力が同じなら同一（P7と同期スケジュールの固定による） |

**区別:**
1. **Snapshot Hash:** `input_sha256`。入力の同一性を表す。
2. **Simulation Result Hash:** 現在は出力していない。simulateではReport JSONのバイト列自体が決定論的なので、外部でそのHashを取れば代替できる。専用フィールドはv0.2.0候補。
3. **Report bytes:** simulateでは正規化入力とバイナリの意味論だけで決まる。analyzeではAdapterのメタデータ（環境依存）も含む。
4. **Domain metadata:** `changes`、`unknown_changes`、`build_context`、Adapter警告はHash対象外。

実測: 入力順のシャッフルでバイト一致（小規模GF09、大規模GF15: 15,250,815 B）。差分試験800件で再現した。Goldenは引き続き`encoding/json`の挙動変化に対する歯止めとして機能する（Go 1.27.1でPASS）。

## 13. Git-Free Acceptance Tests

判定: **PASS** = 現行実装で成立（証跡あり） / **PASS+TEST** = 成立するが自動テストが不足 / **IMPL** = 実装が必要。

| ID | シナリオ | 期待値 | 現状 | 証跡 / 必要な作業 |
|---|---|---|---|---|
| GF-01 | `.git`なし | simulate成功 | **PASS** | fixture（Golden一致）、`TestExecutableSimulatesOutsideRepositoryWithoutGoOrGit` |
| GF-02 | git init直後、HEADなし | simulate成功 | **PASS+TEST** | fixture（Git有無の両方でGolden一致）。自動テストを追加する |
| GF-03 | Git実行ファイルなし | simulate成功 | **PASS** | `PATH`空のfixture、既存の実行ファイルテスト |
| GF-04 | Go実行ファイルなし | ビルド済みsimulate成功 | **PASS** | 同上 |
| GF-05 | 任意のNode ID・kind | Schemaに適合すれば受理 | **IMPL** | IDはPASS。kind `facility`、Edge kind `ships_to`、attenuation 0/2はREJECT |
| GF-06 | 非ソフトウェアGraph | 伝播計算成功 | **IMPL** | Go語彙を流用すれば計算できる（感染接触網でPASS）。自然な語彙では不可 |
| GF-07 | 循環Graph | 正常に固定点判定 | **PASS** | fixture、`TestPropertiesWhenRandomFiniteGraphs` |
| GF-08 | 複数Seeds | 期待どおりの結果 | **PASS** | fixture、`TestCausesWhenMaximumHasTies` |
| GF-09 | 入力順シャッフル | Canonical Result一致 | **PASS** | fixture（バイト一致）、`TestDeterminismWhenInputOrderChanges` |
| GF-10 | 不正Endpoint | 明確なValidation Error | **PASS** | exit 1 `edge endpoint not found: "a" -> "ghost"` |
| GF-11 | Gitなしでhelp/version | 成功 | **PASS+TEST** | `PATH`空でhelp/version/simulate --helpがexit 0。実行ファイルを使うテストを追加する |
| GF-12 | Snapshot Golden | バイト一致 | **PASS** | `TestSimulateGoldenJSON`、`TestChainGoldenWhenSnapshotLoaded`、cmp |
| GF-13 | Git解析の明示実行、Gitなし | 適切な診断 | **PASS** | exit 1 `analyze requires Git 2.45.0 or later …`、`git_requirement_unix_test.go` |
| GF-14 | 不完全なAdapter入力 | 誤ったcompleteにならない | **PASS** | fixture: 未分類の`data.bin` → incomplete/exit 2、stop_reason converged。親環境GOFLAGS → incomplete |
| GF-15 | 大規模Graph | 正確性と性能の計測 | **PASS** | 20,000 Node／200,000 Edge／1,000 Seed: 0.61秒（3回）、最大RSS約173MB、シャッフル後もバイト一致。20,001 Nodeと16MiB超はREJECT |
| GF-16（新） | Go Profile検証の移設 | analyzeが旧制約を維持 | **IMPL** | 旧6ケース（node_kind、edge_kind、test_out、prod_to_test、test_to_prod、attenuation）をAdapter側のテストへ移す |
| GF-17（新） | 緩和の互換性 | 全Golden・既知Hashが不変 | **IMPL（テスト）** | example.json / cli-empty の`input_sha256`とReportのバイト列を固定するテスト |
| GF-18（新） | attenuationの範囲 | -1、4、int範囲外をREJECT。0..3をACCEPT | **IMPL** | — |
| GF-19（新） | a=0の深い鎖 | 深さ ≤ L−1 で収束、それ以上は`generation_limit` | **IMPL（テスト）** | P6 |
| GF-20（新） | a=0のcycleのcauses | 最終causesの支持循環とwavesの整礎性 | **IMPL（テスト）** | §6.5 |
| GF-21（新） | simulateのText表示 | Git/Test用の文言を出さない | **IMPL** | §11 |

既存の正常系はすべて維持する。変更が必要な既存テストは`TestRejectWhenDomainInputInvalid`の6ケース（拒否から受理に変わるため、GF-16へ移す）だけである。

## 14. Compatibility Assessment

| 対象 | 影響 |
|---|---|
| Snapshotの形状 | 変化なし |
| 受理集合 | 厳密なsuperset（REJECT→ACCEPTのみ。ACCEPT→REJECTはなし） |
| 既存入力の`input_sha256` | **不変**。正規化（並べ替え・統合）を変えず、検証は通過判定だけの役割なので、旧条件を通過する入力は新条件も通過し、正規化結果も同一になる |
| Report JSONの形状・Golden | 不変（`testdata/example.report.json`、`cli-empty.golden.json`） |
| ruleset ID | 不変（式も不変。入力の定義域が広がるだけ） |
| analyzeの挙動 | 不変（Go Profile検証で旧制約を再現する） |
| Text出力 | simulateで変化（Golden対象外。`TestSimulateJSONAndText`は`Status: complete`を確認しているだけなので通る） |
| 既存テスト | 6ケースを移設。その他は維持 |
| ドキュメント | README/README.ja冒頭の位置づけ、SCHEMA（kind・attenuation・causesの意味論・Completenessの層）、ARCHITECTURE（Graph節、上界）、VALIDATION、CHANGELOGを更新する（実装フェーズで実施） |

---

# Part B — Naming Review

## 15. Naming Direction

| 方向 | 適合度 | 判断 |
|---|---|---|
| A. Risk-Oriented Harness | 低い | 状態の一種（リスク）しか表さない。将来のRule（状態減少、多状態）と合わない |
| **B. Generic State Propagation Harness** | **最も高い** | Vision（任意Graph上の状態・リスク・影響の伝播）と一致する。現行Ruleは状態伝播の一例として位置づけられる |
| C. Generalized Life-Inspired Rule Engine | 中程度 | 未実装のConway型・多状態Ruleを示唆し、過大な約束になる。「Rule Engine」は業務ルールエンジン（Drools等）と混同されやすい |

→ **Bを採用**する。ただし「Seed」「Cell」「波及」のように、セル・オートマトン由来の比喩を控えめに使うのは許容する。

## 16. RiskLife維持案

- **メリット:** 継続性（ドキュメント・Mandala goal・検証記録がそのまま使える）。CLIが短い（8文字）。覚えやすい。GitHubの同名repoは本人所有の1件だけ。
- **制約:** 「Risk」は状態の一種に限定され、Vision #1（Generic）と食い違う。「Life」はConway型の意味論を示唆するが未実装である。休眠ユーザーアカウント`risklife`（2016年作成、公開repo 0）が存在し、`github.com/risklife`と混同されうる。保険・健康分野の「risk lifestyle / lifecycle」系repoとの語感の重なりもある。
- **評価:** 67/100（§19）。改名しないのも有効な選択肢だが、Visionとの整合では劣る。

## 17. GitHub衝突調査の方法と実施日

- **実施:** 2026-10-09 23:26〜23:32 JST。`gh` 2.83.2、認証済みアカウント`cottondesu`。**読み取り専用のGET**だけを使った。
- **方法:** 各候補について次を行った。(1) `search/repositories` `q="<Name> in:name fork:true"`、(2) 単語に分割した形（例: `state-tide`）で同じ検索、(3) 名前の区切り文字（`-`、`_`、`.`）を除いて小文字化した値で完全一致を判定、(4) `users/<Name>`でユーザー・Organizationの有無を確認。最終候補にはさらに (5) `in:name,description,topics,readme`と空白区切りのフレーズ検索を行った。archive・forkも対象に含めた。
- **限界:** GitHub検索の索引の網羅性は保証されない。0件であることは**商標上の安全を意味しない**。2026-10-09以降の空き状況も保証されない。
- **本人所有の扱い:** `cottondesu/risklife`（public、**空repo**、コミット・タグ・Release・forkはいずれも0、2026-10-08作成）は衝突として扱わない。

## 18. 競合を除外した正式候補

| 候補 | 判定 | GitHubの所見 |
|---|---|---|
| StateTide（事前候補） | **除外** | `dxj1031/State-Tide` が存在（事前検索はハイフン形を見落としていた） |
| RuleLattice（事前候補） | **除外** | `506058115-cmd/rule-lattice`（「reproducible elementary cellular automata」で**同じ領域**） |
| GraphTide（追加） | **除外** | `wheatman/GraphTide`とそのfork、Organization `GraphTides`（stream graph processing） |
| CellTide（追加） | **除外** | `clemenshug/celltide` |
| SeedWave（追加） | **除外** | 同名repo 3件、ユーザー`SeedWave`、「Seedwave™」の表記あり |
| StateWave（追加） | **除外** | `smaramwbc/statewave`（★374、AI memory runtime、fork多数）、Organizationあり |
| GraphLife（追加） | **除外** | `ulyssesyang/Graph-Life`、ユーザーあり |
| SeedLife（追加） | **除外** | 同名repo 5件、Organizationあり |
| LifeCascade（追加） | **除外** | 完全一致はないが、`NoeBrt/Game-Of-Life-Cascade`（Conway領域）と紛らわしい |
| RuleTide（事前候補） | **条件付き（減点）** | 同名repoはない。休眠ユーザー`ruletide`（2020年作成、公開repo 0）が存在 |
| StateRipple | **有効** | 完全一致0、近似0、詳細検索0、アカウントなし |
| StateCascade | **有効** | 完全一致0、近似0、詳細検索0、アカウントなし |
| SeedRipple | **有効** | 完全一致0（近似は無関係のfork `ionic-seed-ripple`のみ） |
| CellRipple | **有効** | 完全一致0、近似0 |
| GraphBloom | **有効** | 完全一致0、近似0 |
| SeedTide | **有効** | 完全一致0（近似は無関係の`Seed-TideRain`） |
| TideCell（追加） | **有効** | 0件 |
| RippleLife（追加） | **有効** | 完全一致0（近似は無関係の`ripple-lifestyle`/`ripple-lifecycle`） |
| RuleCurrent | **有効** | 完全一致0 |
| GraphCurrent | **有効** | 完全一致0（「graph current」は電流グラフの一般語句として多用されている） |

## 19. 有効候補の100点評価

配点: Vision 25 / 汎用性 20 / 識別性 20 / 覚えやすさ・発音 15 / CLI 10 / 拡張性 10。GitHub衝突は加減点ではなくゲートとして扱う（§18で除外済み）。

| 候補 | Vision | 汎用 | 識別 | 記憶 | CLI | 拡張 | **計** | 主な減点理由 |
|---|---|---|---|---|---|---|---|---|
| **StateRipple** | 23 | 18 | 16 | 13 | 8 | 8 | **86** | XRPL・Ripple社（`RippleState`はXRPLのledger型名）を連想させうる。11文字 |
| **StateCascade** | 23 | 18 | 15 | 13 | 7 | 8 | **84** | 「Cascade」は一般語で、開発者向けAI製品（Windsurf Cascade）と語が重なる。12文字 |
| **CellRipple** | 21 | 15 | 16 | 13 | 8 | 8 | **81** | 「Cell」は格子・生物・通信を連想させ、汎用性がやや下がる |
| SeedRipple | 22 | 16 | 14 | 13 | 8 | 7 | 80 | 決定論ツールなのに「seed」が乱数seedを連想させる |
| GraphBloom | 19 | 18 | 13 | 14 | 8 | 8 | 80 | Bloom filter（「graph bloom filter」は既存の研究用語）・BLOOM LLMと混同されうる |
| SeedTide | 19 | 16 | 16 | 12 | 9 | 7 | 79 | 意味がやや曖昧 |
| TideCell | 17 | 14 | 17 | 12 | 9 | 8 | 77 | 潮力電池などを連想させ、Visionと遠い |
| RippleLife | 19 | 15 | 14 | 13 | 9 | 6 | 76 | 「Life」の過大な約束、Ripple社との連想 |
| RuleTide（条件付き） | 16 | 16 | 11 | 12 | 9 | 9 | 73 | ユーザーアカウント衝突、Rule中心の語感 |
| RuleCurrent | 15 | 15 | 14 | 11 | 8 | 9 | 72 | 「Current」が電流と「現在の」の二義になる |
| GraphCurrent | 17 | 17 | 10 | 11 | 7 | 7 | 69 | 一般語句と重なる |
| RiskLife（現行） | 14 | 9 | 15 | 14 | 9 | 6 | 67 | 状態をRiskに限定、Conwayの示唆、休眠アカウント |

**上位5:** StateRipple 86、StateCascade 84、CellRipple 81、SeedRipple 80、GraphBloom 80。

## 20. 上位3名称の詳細比較

| 観点 | StateRipple | StateCascade | CellRipple |
|---|---|---|---|
| 意味 | 「状態の波及」。英語の"ripple effect"は「波及効果」で、ソフトウェア変更影響分析の古典用語でもある。波紋は距離とともに弱まり、**attenuationと一致**する | 「状態の連鎖」。依存の下流へ流れる同期wave | 「セルの波紋」。セル・オートマトン由来であることを残せる |
| Vision適合 | 状態・リスク・影響のすべてに当てはまる | 状態伝播に強く当てはまるが、障害の連鎖を連想させる | ライフゲーム型の出自に合うが、「Cell」が格子を想起させる |
| 将来Rule | 単調伝播に最も合う。振動するRuleには比喩が弱い | 一方向の流れを示唆する | 多状態のCAとの相性がよい |
| CLI | `stateripple`（11文字） | `statecascade`（12文字） | `cellripple`（10文字） |
| リスク | 「Ripple」商標（Ripple Labs）との類否は未確認 | 「Cascade」を含む既存製品の商標は未確認 | 同上（Ripple） |
| GitHub | 0件、アカウントなし | 0件、アカウントなし | 0件、アカウントなし |

## 21. 最も推奨する名称

| 項目 | 推奨 |
|---|---|
| 名称 | **StateRipple**（次点 StateCascade。Ripple系の商標確認が不利な結果になった場合に繰り上げる） |
| CLIバイナリ | `stateripple` |
| Go Module Path | `github.com/cottondesu/stateripple` |
| 英語Tagline | *Deterministic ripple-effect propagation of state over any graph.* |
| 日本語説明文 | 任意の有向グラフ上で、Seedから始まる状態・リスク・影響の波及を世代ごとの同期更新で決定論的に計算する、オフライン動作の汎用ハーネス。 |
| ruleset ID | **`risk-propagation-v1`を維持する**（Ruleの名前であってプロダクト名ではない。変更するとHashが変わる） |

## 22. GitHub・Go Module・CLIへの影響

| 対象 | 影響 | 規模 |
|---|---|---|
| GitHub repository | `cottondesu/risklife`は空のpublic repo。rename（旧URLはリダイレクトされる）か、新規作成して旧repoを削除する。**現時点では利用者・fork・Releaseが0** | 小 |
| Go module path / import | `go.mod:1` と import path 60行（機械的な置換） | 小 |
| `cmd/risklife` → `cmd/stateripple` | ディレクトリ名、`.gitignore`の`/risklife`・`/risklife.exe` | 小 |
| CLI help / version / エラー接頭辞 | `cli.go:18,31,34,35,41,65,69,78,90,95,121`、`TestVersion`（`cli_test.go:32`） | 小 |
| Text Report見出し | `report/text.go:12` `RiskLife v0.1.0` | 小（JSON・Hashへの影響なし） |
| JSON Schema / Report JSON / Snapshot Hash | **影響なし**（プロダクト名を含まない。`model.go`、`snapshot.go:27-34,97-107`） | なし |
| Golden / fixture | JSON Goldenは不変。テスト内の文字列（`RiskLife Test`、`com.risklife.audit`、`RISKLIFE_REAL_GIT`、`RISKLIFE_VCS_LOG`）は任意で変更 | 小 |
| GitHub Actions / Release workflow | 存在しない | なし |
| README / docs / CHANGELOG | 名称を含む行数: README.md 14、README.ja.md 11、VALIDATION 15、ARCHITECTURE 3、SCHEMA 2、IMPLEMENTATION 1。過去の検証記録の本文は史実として残し、注記を加える | 中 |
| Mandala | goal文字列`Implement and validate RiskLife v0.1.0`は**CLIで変更できない**（変更には`clean`→`init`という破壊的操作が必要で、別途明示的な依頼を要する）。旧名のgoalを履歴として残すことを推奨 | 小 |
| AI Agent連携 | 現時点ではない（README Roadmapの候補にすぎない） | なし |
| Go package名 | `internal/*`はプロダクト名を含まないので不変 | なし |

**名称変更と計算結果の変更は別の作業である。** 改名の前後で、全JSON Golden、`input_sha256`、差分試験の結果がバイト一致することを改名Phaseの完了条件とする。

**移行コストの比較:** 初回公開前なら、文字列置換とrepo renameだけで済み、利用者への告知は不要。公開後は、module pathの変更が`go install …@latest`の利用者に対する破壊的変更になり、旧module pathへの`retract`や告知、旧repo名へのリンクの維持が必要になる。なお、空repoのためpkg.go.dev・proxy.golang.orgに版はないと推定されるが、**GitHub以外への通信を禁止しているため未確認**である。

## 23. 改名の必要性に関する判断

**判断: RENAME BEFORE v0.1.0（条件付き推奨）**

- **理由:** (1) VisionのGeneric性と現行名の「Risk」「Life」が食い違う。(2) 改名コストが最小になるのは今である（空repo、未公開module、Hash・JSONへの影響なし）。(3) 公開後に改名するのが最も高くつくため、**DEFER RENAMEは推奨しない**。
- **条件（すべて満たされない場合はKEEP RISKLIFEを選ぶ）:**
  1. ユーザーが名称を承認すること。
  2. 改名直前にGitHubを再確認すること（§17の手順を再実行）。
  3. GitHub以外の確認（未実施 = **NOT VERIFIED**）: pkg.go.dev / Go module proxy、npm、PyPI、crates.io、主要ドメイン（`.dev` `.io` `.com`）、既存製品名、商標（特に**「RIPPLE」系の登録商標**、次点の場合は「CASCADE」系）。
- 名称は評価・提案にとどめ、**改名は実施していない**。

---

# Part C — Implementation Strategy

## 24. Architecture Decision Records

### ADR-001 — Product Identity
- **Context:** README冒頭は「Go code changesのためのcellular risk engine」（README.md:5）。一方、Core（engine/simulate）はすでにGitに依存していない。
- **Alternatives:** (a) Git/Go変更解析ツール。(b) Generic Propagation Harness（Adapterは任意）。
- **Decision:** (b)。
- **Rationale:** Vision #1。Coreの実装がすでに独立しているので、正式な位置づけを実装に合わせる。
- **Trade-offs:** Go利用者向けの訴求が弱まる。analyzeを「同梱Adapter」として明示的に説明する必要がある。
- **Compatibility Impact:** コードと形式への影響はない。README・helpの文言を変える。
- **Acceptance Criteria:** README/helpの最初の使用例がsimulateであること。GF-01〜04・11が自動テストでPASSすること。
- **Implementation Impact:** ドキュメントとhelp文の修正のみ。

### ADR-002 — Core Graph Model
- **Context:** `graph.go:54-86`と`snapshot.go:57-65`にGo語彙とattenuation規則がある。
- **Alternatives:** (a) Go固有Graphを維持する。(b) v1を緩和して汎用Graphにする（kindは自由、attenuationは0..3）。(c) v2 Schemaを新設する。(d) 新しいruleset IDにする。
- **Decision:** (b)。Go語彙はGo Adapterの`validateGoProfile`へ移す。
- **Rationale:** 受理集合のsupersetなので、既存のHash・Golden・Reportが不変（§14）。初回リリースで二重のSchemaを避けられる。
- **Trade-offs:** 手書きのGo風Snapshotに対するGo制約の検証がsimulateからなくなる（意味論上はGo Adapterの責務）。a=0のcycleでは最終causesが循環しうるため、文書化が必要。
- **Compatibility Impact:** REJECT→ACCEPTの変化のみ。6つのテストケースを移設する。
- **Acceptance Criteria:** GF-05/06/16/17/18/19/20がPASSし、既存Goldenが不変であること。
- **Implementation Impact:** `graph.go`・`snapshot.go`の検証を削減し、adaptersに約40行の検証関数を追加し、テストを移設する。

### ADR-003 — Input Contract
- **Context:** simulate（Snapshot）とanalyze（Git diff）がある。
- **Alternatives:** (a) Git Diffを基本にする。(b) Canonical Snapshotを基本にする。
- **Decision:** (b)。Git diffはSnapshot（Seeds）を生成する手段の一つとする。
- **Rationale:** Git/HEADがなくても動き、Hash対象がSnapshotで閉じている。
- **Trade-offs:** なし（現状と一致）。
- **Compatibility Impact:** なし。
- **Acceptance Criteria:** GF-01〜04、GF-12。
- **Implementation Impact:** ドキュメントのみ。

### ADR-004 — Git Dependency
- **Decision:** 任意Adapterとする。Git 2.45.0以上はanalyzeだけの要件。
- **Context / Rationale:** `requireGit`はAnalyze内でのみ呼ばれる（`analyze.go:34`）。help/version/simulateはGitを起動しない（GF-11、C9）。
- **Alternatives:** 必須依存にする案は不採用。
- **Trade-offs:** なし。
- **Compatibility Impact:** なし。
- **Acceptance Criteria:** GF-02/03/11/13。
- **Implementation Impact:** テストの追加のみ。

### ADR-005 — Go Dependency
- **Decision:** 任意Adapterとする。Go toolchainはanalyzeの実行時とビルド時だけに必要。
- **Context / Rationale:** `go`コマンドの呼び出しは`goGraph`（`go.go:38,61,78,87`）に限られる。ビルド済みバイナリのsimulateにGoは不要（GF-04）。
- **Alternatives:** 必須依存にする案は不採用。
- **Trade-offs:** なし。
- **Compatibility Impact:** なし（Go語彙の検証をAdapterへ移す点はADR-002で扱う）。
- **Acceptance Criteria:** GF-04、GF-16。
- **Implementation Impact:** ADR-002に含まれる。

### ADR-006 — Rule Extensibility
- **Context:** `engine.Rule[S]`は汎用で、RiskPropagationV1は唯一の実装。
- **Alternatives:** (a) RiskPropagationV1に固定。(b) プラグイン機構や抽象Rule APIを導入。(c) 現行の`engine.Rule`境界を維持し、2つ目のRuleを追加する時点でrulesetによるswitchを導入する。
- **Decision:** (c)。
- **Rationale:** 過剰な抽象化を避ける。停止性の保証はRuleに固有であり、非単調Ruleには完了基準と周期検出が別途必要になる（§6.6）。
- **Trade-offs:** 2つ目のRuleの追加時にCLIのdispatchと、Snapshotの状態型（版の扱い）を設計する必要がある。
- **Compatibility Impact:** なし。
- **Acceptance Criteria:** 停止性の保証がRiskPropagationV1に限定されることがARCHITECTURE/SCHEMAに明記されていること。
- **Implementation Impact:** v0.1.0ではドキュメントのみ。

### ADR-007 — Generic Result
- **Context:** Reportに汎用部とドメイン部（test_targets/changes/build_context）が混在している。
- **Alternatives:** (a) Test Targetsを中心にする。(b) 状態伝播の結果を中心にし、ドメイン部はReport v1にそのまま残す（互換）。(c) 今すぐreport v2に分離する。
- **Decision:** (b)。Text表示だけはモード別にする。(c)はv0.2.0とする。
- **Rationale:** Goldenを変えずに、汎用の意味を定義できる。
- **Trade-offs:** JSONにはsimulateでも空のドメイン項目が残る。
- **Compatibility Impact:** JSONは不変。Textはsimulateで変わる。
- **Acceptance Criteria:** GF-21。§11の定義がSCHEMAに反映されていること。
- **Implementation Impact:** `report/text.go`（小）。

### ADR-008 — CLI Evolution
- **Decision:** 既存のsimulate/analyzeを活用し、新しいコマンドは追加しない。help文の改訂のみ行う。stdinとExplicit入力はv0.2.0。
- **Context / Rationale:** simulateがExplicit Inputの要件を満たしている。
- **Alternatives:** 新CLIを全面的に導入する案は不採用。
- **Trade-offs:** stdinパイプラインは使えない。
- **Compatibility Impact:** なし。
- **Acceptance Criteria:** help文にsimulate中心の説明とexit codeの統一定義が含まれ、`TestHelpUsesStdout`が維持されること。
- **Implementation Impact:** `cli.go:18`。

### ADR-009 — Completeness Semantics
- **Decision:** §8の4層を正式化する。v0.1.0ではReport形状を変えない（判別可能性の定理により十分）。層2と層3が同時に起こりうるAdapterを導入する時点で、`completeness`オブジェクト（report v2）を追加する。
- **Context / Rationale:** `status`は合成値だが、`stop_reason`とモードから分解できる。
- **Alternatives:** 今すぐ`completeness`オブジェクトを追加する案は、Goldenが変わるため不採用。
- **Trade-offs:** 利用者が判別規則を知る必要がある（SCHEMAに明記する）。
- **Compatibility Impact:** なし。
- **Acceptance Criteria:** GF-14の維持、SCHEMAへの4層表の記載、simulateにAdapter由来の制約がかからないこと。
- **Implementation Impact:** ドキュメントとテスト。

### ADR-010 — Project Naming
- **Decision:** 方向性はB。**RENAME BEFORE v0.1.0を条件付きで推奨**し、名称はStateRipple（次点StateCascade）とする。最終決定はユーザーが行う。条件が満たされなければKEEP RISKLIFEとし、DEFERは避ける。
- **Context / Rationale:** §15〜23。
- **Alternatives:** KEEP / RENAME / DEFER。
- **Trade-offs:** 改名すると、ドキュメントの更新とMandala goal名の不一致が生じる。改名しなければ、Visionとの不一致が恒久化する。
- **Compatibility Impact:** JSON・Hashへの影響なし。Text・help・versionが変わる。
- **Acceptance Criteria:** ユーザーの承認。GitHubの再確認。GitHub以外の確認の記録。改名の前後でJSON Golden・Hashがバイト一致すること。
- **Implementation Impact:** §22（独立したPhase N）。

## 25. v0.1.0 Minimum Requirements

| 要件 | 状態 |
|---|---|
| Git非依存のCore | **済** |
| 任意のNode ID | **済** |
| 任意のNode/Edge kindとattenuation 0..3 | **要実装（B1・B2）** |
| RiskPropagationV1 | **済**（式と一致） |
| Canonical Snapshot / Deterministic JSON / SHA-256 | **済** |
| Gitなしで使えるCLI | **済** |
| 入力の明示的なValidation | **済**（汎用化後も維持） |
| Adapterの任意性 | **済** |
| 既存の安全性・完全性の維持 | **済**（B2でanalyzeの制約を維持） |
| 汎用的なText表示・help | **要実装（B3・B4）** |
| Gitなし受け入れテストの自動化 | **要追加（B5）** |
| プロダクトの説明と名称の確定 | **要決定（B6・B7）** |

## 26. v0.1.1 Roadmap（Reliability and Performance）

| 候補 | 必要性 | 難度 | 既存機能への影響 | 延期可否 |
|---|---|---|---|---|
| Git objectのbatch読み取り（`cat-file --batch`） | 高（1万ファイルで182秒、`README.md`のNEW-01 cost） | 中〜高（framing・サイズ・エラーの検証が必要） | analyzeの性能のみ | 可（大規模repoでは実用上の問題） |
| Git subprocess数の削減 | 中 | 中 | 同上 | 可 |
| FIFOハング対策（事前検査・deadline・process group） | 中（NF-04、P2の運用リスク） | 中〜高（プラットフォーム依存） | analyzeのみ | 可 |
| tracked symlinkの過剰拒否の改善 | 低〜中（NF-03） | 中 | 完全性の判定 | 可 |
| testdataの扱い（入れ子のgo.mod、NF-07） | 低 | 低〜中 | 完全性の判定 | 可 |
| 一時ファイル名の制約（NF-06） | 低 | 低 | 出力 | 可 |
| cleanupエラー契約（NF-05） | 低 | 低 | 出力 | 可 |
| **テストのhermetic化**（adaptersテストで親環境のGOFLAGS/GOEXPERIMENTを除去） | 中（CIで15件が誤ってFAILしうる） | 低 | テストのみ | 可（v0.1.0に前倒しを推奨） |
| `internal/graph`の単体テスト | 中 | 低 | テストのみ | 可（v0.1.0のB1で同時に追加すると効率的） |

## 27. v0.2.0 Roadmap（Input Expansion）

| 候補 | 必要性 | 難度 | 影響 | 延期可否 |
|---|---|---|---|---|
| stdin Snapshot | 中 | 低〜中（安全性契約の再設計） | CLI | 可 |
| Explicit Graph/Seed入力（分割ファイル、Seedの上書き） | 中 | 中 | CLI | 可 |
| Git/Go Adapterの分離と汎用Adapter契約 | 中 | 中 | adapters | 可 |
| Changed Files Adapter / Local Input Adapter | 中 | 中 | 新規 | 可 |
| 構造化Diagnostics（code、source、message） | 中 | 中 | Report v2 | 可 |
| `completeness`オブジェクト、`result_sha256` | 中 | 低〜中 | Report v2 | 可 |
| ドメイン固有のResult Adapter（test_targets等の分離） | 中 | 中 | Report v2 | 可 |
| kindの省略可能化、Optional Metadata | 低 | 中（Hash規約） | Schema | 可 |

## Future

複数Rule（rulesetによるdispatch）、非単調な状態遷移（完了基準の宣言と周期検出）、Conway型Rule（固定世代数での完了）、多状態Cellular Automata（状態型の版管理）、AIエージェント向けの構造化入出力。いずれもv0.1.0の設計を変えずに追加できる（§6.6）。

## 28. Minimal Implementation Plan

### A. Already Implemented
engine、Hash、Canonical、Read（構文・Unicode・上限）、Trace、Gitなしのsimulate、exit code、`--out`の安全出力、analyzeの安全制御、Determinism、大規模入力の上限。**重複して実装しない。**

### B. Required Before v0.1.0

| ID | 変更理由 | 対象ファイル | 現在の責務 → 変更後 | 具体仕様 | 難度 | 回帰リスク | Schema | Determinism | テスト | Mandala |
|---|---|---|---|---|---|---|---|---|---|---|
| **B1** Core検証の汎用化 | Go語彙がCoreに固定されている | `internal/graph/graph.go`、`internal/snapshot/snapshot.go` | Go語彙と構造を検証 → 構造（ID・参照・重複・矛盾）とattenuation 0..3だけを検証 | §5.2〜5.4。`graph.go:54-58,72-86`を削除し、kindは`ValidString`で検証。`snapshot.go:57-65`を`0 ≤ a ≤ 3`に置き換え。ordered pairの一意性は維持 | 低 | 中（受理集合の変化） | 形状不変、受理集合を拡大 | 既存入力は不変 | GF-05/06/17/18/19/20、`internal/graph`の単体テストを新設 | `generic.schema` |
| **B2** Go Profile検証の移設 | analyzeのfail-closedを維持する | `internal/adapters/`（新規`profile.go`、`analyze.go:123`付近から呼ぶ） | なし → Go語彙の5制約を検証 | §9。違反は「collected graph violates Go profile」でexit 1 | 低 | 低 | なし | なし | GF-16（旧6ケースの移設）、analyzeの全既存テスト | `generic.go-profile` |
| **B3** Text表示の汎用化 | simulateでソフトウェア文言が出る | `internal/report/text.go` | 常にGit/Test節を表示 → モード別に表示 | §11 | 低 | 低（JSONは不変） | なし | Textのみ | GF-21、`TestTextIncludesAnalysisEvidence`の維持、text_security | `generic.text` |
| **B4** help文 | 位置づけとexit 2の文言 | `internal/cli/cli.go:18` | analyze中心 → simulate中心 | §10 | 低 | 低 | なし | なし | `TestHelpUsesStdout`、GF-11 | `generic.text` |
| **B5** 受け入れテスト | GF系の自動化 | `internal/cli/*_test.go`、`internal/rules/risk/*_test.go`、`internal/snapshot/*_test.go` | — | GF-02/11/17〜21を追加。HashとGoldenを固定 | 低 | なし | なし | 固定して検証 | 本表 | `generic.acceptance` |
| **B6** ドキュメント | 位置づけの変更 | README/README.ja/SCHEMA/ARCHITECTURE/VALIDATION/CHANGELOG | Go変更解析中心 → 汎用ハーネス＋任意Adapter | §2、§5〜§12 | 低 | 低 | 文書のみ | なし | リンク確認 | `generic.docs` |
| **B7** 名称決定 | ADR-010 | — | — | ユーザーの決定、GitHub再確認、GitHub以外の確認 | — | — | — | — | — | `naming.decision` |
| （推奨）**B8** テストのhermetic化 | CIで誤FAILしうる | adaptersのテスト補助 | 親環境を継承 → `GOFLAGS`/`GOEXPERIMENT`を空にする | テスト内で`t.Setenv` | 低 | 低 | なし | なし | `GOFLAGS=-mod=readonly go test ./...`がPASSすること | `generic.acceptance` |

### Phase N — 改名（承認後のみ。独立したPhase）

1. GitHubの再確認（§17）とGitHub以外の確認を記録する。
2. `go.mod`のmodule path、import path 60行、`cmd/<name>`、`.gitignore`、CLI文字列、`text.go:12`、`TestVersion`を変更する。
3. **改名の前後で、全JSON Golden、example/cli-emptyの`input_sha256`、差分試験の結果がバイト一致することを確認する**（計算結果を変えていないことの証明）。
4. README/docsを新名称にし、過去の検証記録には旧名を史実として残して注記する。
5. GitHub repoのrenameは、利用者が承認・実行する外部操作として最後に行う。

**推奨順序:** 名称決定（B7）→ B1・B2・B5（Core汎用化。名称には触れない）→ B3・B4 → Phase N（承認された場合）→ B6（ドキュメントを一度で更新）→ 全検証（`gofmt`、`go vet`、`go test`の通常・race・shuffle、Golden、差分試験、GF fixture）。

### C. Recommended for v0.1.1
§26の表のとおり。

### D. Future Scope
§27とFuture。

## 29. Regression Risk Matrix

| 変更 | 影響しうる契約 | 発生確率 | 影響度 | 検出手段 |
|---|---|---|---|---|
| B1 検証の緩和 | Hash・正規化の同一性 | 低 | 高 | GF-17（Hash/Goldenの固定）、差分試験、`TestHashWhenDefaultsNormalized` |
| B1 attenuation 0..3 | 停止性・limit | 中（深いa=0鎖） | 中（明示的な未完了） | GF-19、P6 |
| B1 a=0のcycle | Traceの解釈 | 中 | 低〜中 | GF-20、SCHEMAへの明記 |
| B2 Profileの移設漏れ | analyzeのfail-closed | 低 | 高 | GF-16、adaptersの既存テスト全件 |
| B3 Text | 表示・escape | 低 | 低 | text_security_test、`TestTextIncludesAnalysisEvidence` |
| B4 help | stdout/stderrの分離 | 低 | 低 | `TestHelpUsesStdout`、`TestInvalidArgumentsSeparateDiagnostics` |
| Phase N 改名 | import・ビルド・テスト文字列 | 中 | 中 | `go build`、全テスト、Golden・Hashのバイト一致 |
| Phase N 改名 | Hash | なし（名称はHash対象外） | — | GF-17 |

## 30. Mandala Coverage Plan

既存Mandalaで追跡できる範囲: Adapterの安全性（RL/NF/NEW系）、CLIの安全出力、Determinism、fuzz、回帰。いずれも`done`である。

**不足している項目:** 汎用Schema、Go Profileの移設、Text・helpの汎用化、Gitなし受け入れテスト、Completenessの4層の明記、名称決定、改名Phase、v0.1.1の信頼性改善。

**構造上の制約:** `adapters`の子は8/8で満杯なので、新しいAdapter作業は新しいrootに置く必要がある。root枠は残り3、総セル枠は残り49。goalの文字列はCLIで変更できない。

**作成時の提案（2026-10-10の実装時に、容量を確認したうえで下記に読み替えて実行済み。実際の配置は`mandala --project <project-root> show --json`を参照）:** root `generic`（node-kind、edge-kind、attenuation、go-boundary、snapshot-compat、git-free、graph-tests、determinism）、root `naming`（identity、module、imports、cli、docs、old-refs、regression）、`validation`への追加（goflags、golden、sha256、race-shuffle、compat）、`cli.text-help`。

元の提案（`<project-root>`は作成時のリポジトリルートを表すプレースホルダーで、このままでは実行できない）:

```text
mandala --project <project-root> add generic
mandala --project <project-root> add generic.schema        # B1
mandala --project <project-root> add generic.go-profile    # B2
mandala --project <project-root> add generic.text          # B3, B4
mandala --project <project-root> add generic.acceptance    # B5, B8 (GF-01〜21)
mandala --project <project-root> add generic.completeness  # §8のSCHEMA反映とGF-14
mandala --project <project-root> add generic.docs          # B6
mandala --project <project-root> add naming
mandala --project <project-root> add naming.decision       # B7
mandala --project <project-root> add naming.recheck        # GitHub再確認とGitHub以外の確認
mandala --project <project-root> add naming.rename         # Phase N（KEEPを選んだ場合は、ユーザーの決定を根拠にna）
```

v0.1.1の作業は、root枠の最後の1つを使って`add --optional reliability`とその子（`reliability.batch-read`、`reliability.fifo`、`reliability.symlink`、`reliability.test-hermetic`）にするか、v0.1.0の完了後に別のgoalとして扱うか（`clean`が必要＝破壊的）を、ユーザーが選ぶ。

---

# Final Verdict

| 判定 | 結論 |
|---|---|
| **Architecture Verdict** | **MINOR REFACTOR REQUIRED** — Engine、Hash、CLIの経路はすでに汎用。変更はCore検証の語彙を外すこと、Go Adapterへの検証移設、Text・helpの修正に限られる。packageの再編は不要 |
| **Naming Verdict** | **RENAME BEFORE v0.1.0**（条件付き推奨: ユーザー承認、GitHub再確認、商標・パッケージレジストリ確認。満たされなければKEEP RISKLIFE。DEFERは非推奨） |
| **Release Specification Verdict** | **SPECIFICATION READY WITH OPEN DECISIONS** |

**未決事項（推奨案つき）:**
1. 名称 → StateRipple（次点StateCascade）、または維持。
2. attenuationの範囲 → 0..3（代替案は{0,1}のまま。その場合GF-06の表現力が下がる）。
3. kindを必須のまま残すか → v0.1.0は必須（省略可能化はv0.2.0）。
4. Mandalaへのセル追加とv0.1.1の追跡方法 → §30の案。
5. B8（テストのhermetic化）をv0.1.0に前倒しするか → 前倒しを推奨。

### Final Recommendation

1. **汎用ハーネスとしての完成度:** 実行基盤（Engine・Determinism・Hash・Gitなしのsimulate・安全な入出力）は完成している。入力語彙が汎用でない点だけが不足しており、GF 15項目のうち13項目がPASS（うち2項目は自動テストが不足）、2項目（GF-05/06）が実装待ちである。
2. **初回リリースまでに必要な設計変更:** B1（kindの自由化とattenuation 0..3）、B2（Go Profile検証の移設）、B3/B4（Text・help）、B5（受け入れテスト）、B6（ドキュメント）。いずれも小規模で、Hash・Golden・Report形状は不変。
3. **名称変更の必要性:** Visionとの整合のためには推奨されるが、必須ではない。改名するなら今が最も安い。
4. **改名先として最適な名前:** **StateRipple**（`stateripple`、`github.com/cottondesu/stateripple`）。ただし商標確認が前提である。
5. **汎用化と安全性を両立する最小手順:** 名称決定 → Core検証の緩和とGo Profile移設を同一変更で行う（analyzeの制約を途切れさせない）→ Hash・Goldenの固定テストで互換性を証明 → Text・help → （承認時のみ）改名Phase。改名の前後でJSON・Hashのバイト一致を確認 → ドキュメントの一括更新 → 全検証。

---

## 付録: 品質ゲート（依頼書 §19）

| # | 項目 | 本書 |
|---|---|---|
| 1 | 本質的な目的 | §2 |
| 2 | Git/GoとCoreの責務分離 | §3.2、§9、ADR-004/005 |
| 3 | 任意のGraphを扱えるか | §1、§13（現状PARTIAL → B1で解消） |
| 4 | Node/Edge/Seedの形式 | §5 |
| 5 | RiskPropagationV1の数学的定義 | §6 |
| 6 | 将来Ruleの拡張境界 | §6.6、ADR-006 |
| 7 | Snapshot Schema | §7 |
| 8 | Validation Rules | §7.2 |
| 9 | Completeness Semantics | §8、ADR-009 |
| 10 | CLI契約 | §10 |
| 11 | 汎用Result | §11 |
| 12 | Determinism Contract | §12 |
| 13 | Gitなし受け入れテスト | §13 |
| 14 | 既存機能との互換性 | §14 |
| 15 | 最小実装計画 | §28 |
| 16 | GitHubで競合しない名称候補 | §18 |
| 17 | RiskLife維持と改名の比較 | §16、§19、§23 |
| 18 | 改名する場合の移行計画 | §22、Phase N |
| 19 | Mandalaで追跡すべき項目 | §30 |
| 20 | v0.1.0の必須作業と延期可能な作業 | §25〜28 |

## 付録: 証跡と再現方法

| 証跡 | 場所（一時領域） |
|---|---|
| 検証用バイナリ（このrepoからビルド。sha256 `134552da…3285`） | `<evidence-dir>/risklife` |
| GF・Validation fixture（`env -i PATH=<空dir>`で実行） | `<evidence-dir>/fx/` |
| GF-14 analyze fixture | `<evidence-dir>/gf14.sh` |
| 差分試験（式の参照モデルとバイナリの比較） | `<evidence-dir>/diff/ref.py`（`python3 ref.py <seed> <n>`） |
| 上界の性質検証 | `<evidence-dir>/diff/bound.py` |
| GitHub検索スクリプトと結果JSON | `<evidence-dir>/naming/check.py`、`deep.py`、`result-*.json` |
