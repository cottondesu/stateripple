# StateRipple v0.1.0（未リリース）

[English](README.md)

*任意のグラフ上で、状態の波及を決定論的に計算する。*

StateRippleは、オフラインで動作するライフゲーム型の汎用ハーネスです。有向グラフとSeedを読み込み、すべてのNode（セル）を世代ごとに同期更新して、最終的なRisk、伝播のTrace、入力のCanonical SHA-256を出力します。NodeとEdgeは、サービス、データベース、ジョブ、サプライヤー、パッケージなど何を表してもかまいません。kindは自由なラベルで、語彙の制限はなく、伝播はkindに依存しません。v0.1.0で唯一の例外は、ラベル`test`に対するReport上の規約です（伝播ルールの節で説明します）。LLM、ネットワークサービス、DB、Git、Go toolchain、外部Goライブラリのいずれも必要としません。

GitとGoによる解析は、GoリポジトリからSnapshotを作る任意のAdapter（`analyze`）です。Coreの必須要件ではありません。

StateRippleは、以前は作業名RiskLife（`github.com/cottondesu/risklife`）として開発していました。旧リポジトリは削除済みで、現在のModuleは`github.com/cottondesu/stateripple`です。

Riskは伝播した優先度であり、確率ではありません。Risk=0は安全性の証明ではありません。`complete`は、与えられた入力に対する計算が完了したことを意味します。入力が現実を完全に表していることや、システムが正しいことは意味しません。

## ビルドと実行

Go 1.23以降を使い、ローカルのチェックアウトでビルドします。

```sh
go build -o stateripple ./cmd/stateripple
./stateripple version
./stateripple simulate --input testdata/generic-supply-chain.json --format text
./stateripple simulate --input testdata/generic-supply-chain.json --format json
./stateripple simulate --input testdata/example.json --format json --out /tmp/stateripple-report.json
```

`simulate`はSnapshotファイルと実行ファイルだけで動作します。リポジトリ外、`.git`のないdirectory、`git init`直後でHEADのないリポジトリ、`PATH`に`git`も`go`もない環境のいずれでも動作します。`help`と`version`も同様です。SnapshotとReportの形式は[スキーマ文書](docs/SCHEMA.md)を参照してください。

Snapshotは1つのJSONファイルです。

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

| 終了コード | 意味 |
|---|---|
| 0 | 完了した結果（`status: complete`） |
| 1 | 入力・使用法・設定・実行・出力のエラー。終了コード1は出力の不存在を保証しません。stdoutに部分出力が残る場合や、公開後のcleanup失敗で完成済みの`--out`ファイルが残る場合があります |
| 2 | 不完全な結果（`status: incomplete`）。固定点を確認せずに世代上限へ達した場合、またはAdapterの証拠が不完全な場合 |

## 伝播ルール：RiskPropagationV1

各NodeのRisk `R(v)`は`{0,1,2,3}`です。Edge `u → v`は「uの状態がvに影響する」ことを表し、明示的なattenuation `a ∈ {0,1,2,3}`を持ちます。各更新は前世代だけを読みます。

```text
R_next(v) = max(R(v), Seed(v), max over u→v of max(0, R(u) − a(u,v)))
```

| Attenuation | Risk 3の伝播元からの寄与 |
|---|---|
| 0 | 3 |
| 1 | 2 |
| 2 | 1 |
| 3 | 0 |

attenuation 4以上、負値、非整数、範囲外の数値は拒否します（exit 1）。Riskが3を超えないため、より大きなattenuationには追加の意味がありません。Riskの範囲を変える場合は、このRuleの変更ではなく新しいRuleとして扱います。Edge kindはラベルにすぎず、kind名からattenuationを推測しません。

| Risk | 名前 | 意味 |
|---|---|---|
| 0 | Dormant | 今回の伝播では非活性 |
| 1 | Watch | 間接的な影響候補 |
| 2 | Active | 優先度の高い候補 |
| 3 | Critical | 直接のSeedまたは強い伝播根拠 |

Generation 0でSeedを適用します。既定の上限は16更新です（`max_generations`で1..1024を指定可能）。上限には、固定点を確認する最後の無変化更新を含みます。Riskは増加しかしないため、有限のGraphは`max(1, |V|)`回以内の更新で固定点に達します。cycleやself-loopがあっても停止します。attenuation 0はRiskを減衰させずに伝えるため、n Nodeのattenuation 0の鎖はn回の更新を必要とします。上限1024では、1024 Nodeの鎖はcomplete、1025 Nodeの鎖は`incomplete`（exit 2）となり、`complete`にはなりません。

上限で停止した結果は、固定点の下限です。出力されたRiskが収束値を上回ることはありませんが、セルのRiskが収束値より低い（0を含む）、`test_targets`に対象が欠ける、または低いRiskで載る、最終causesが空または一部だけになる、ということがありえます。確定した結果が必要な場合は、`max_generations`を大きくして再実行してください。詳細は[Trace semantics](docs/SCHEMA.md#trace-semantics)を参照してください。

ConwayのGame of Life（B3/S23）とは異なり、近傍数による誕生・死滅はなく、状態は減少しません。単調な最大値伝播です。「ライフゲーム型」は、セルを世代ごとに同期更新する構造だけを指します。

Reportには、全セル、Seed理由、最大Riskを支える同点原因、初回活性化世代、変化した世代履歴、警告、停止理由、入力SHA-256が含まれます。`test_targets`は、kindが`test`に完全一致し、Riskが正のNodeを投影したフィールドで、汎用Graphでは通常空です。Coreが解釈するkindはこれだけで、v0.1.0では以前のReportとのバイト互換を保つために残しています。この規約は`simulate`にも適用されます。ソフトウェア以外のGraphでもラベル`test`を使うと、そのNodeが`test_targets`に入り、Text出力がテスト向けの文言になります。意図しない場合は別のラベルを使ってください。Text出力のGit節とテスト節は、`analyze`のReportか、テスト対象がある場合にだけ表示します。

## 任意のGit + Go Adapter：analyze

単一Goモジュールの、コミット済みでクリーンなリポジトリのルートから解析します。

```sh
/path/to/stateripple analyze --base HEAD~1 --format json
/path/to/stateripple analyze --base HEAD~1 --format text
/path/to/stateripple analyze --base HEAD~1 --format json --out /tmp/stateripple-analysis.json
```

`analyze`にはGo 1.23以上と、global option `--no-lazy-fetch`を備えた**Git 2.45.0以上**が必要です。`analyze`の実行時に限り、情報収集の前にオプションの実動作とversionを確認します。非対応・未インストールなら要件エラー/exit 1で停止し、安全性を下げるfallbackはありません。Apple Gitは独自の配布versionを持ち、実測した同梱Apple Git 2.30.1は非対応です。`simulate`、`help`、`version`はGitもGoも必要としません。

baseの既定値はHEADの親です。比較先はHEADで、未コミット変更は対象外です。追跡対象の変更やignoreされていない未追跡ファイルがある場合は拒否します。ignoreされたビルド成果物は区別します。出力はstdout、診断はstderrです。`--out`を指定しなければファイルは作成しません。次回解析時の未追跡ファイルを避けるため、出力先はリポジトリ外を推奨します。

Go情報収集の前後で、追跡対象の通常ファイルの生バイトと実行属性をHEADのtreeと照合します。`assume-unchanged`や`skip-worktree`は一致の証拠として扱いません。clean/smudge filterや改行変換によって生バイトが異なる場合もエラーです。Goが実際に使用するignoredファイルは、native build入力も含めincompleteとなります。`diff.ignoreSubmodules=all`でもgitlink変更を検出し、submodule解析は未対応のためincompleteとします。

Snapshot入力は通常ファイルに限定し、FIFO、socket、device、directoryは拒否します。`--out`は新規ファイルだけを作成します。既存の通常ファイル、hardlink、symlink（danglingを含む）、directory、特殊ファイルはすべて拒否し、内容、ACL、xattr、owner/group、permission bitsを変更しません。同じdirectoryの一時ファイルへprivateな`0600`で全内容を書込・sync・closeし、排他的hardlink（`os.Link`）で公開します。競合で作成されたentryも置換しません。hardlink非対応環境ではエラーとし、renameへのfallbackは行いません。毎回新しい出力先を指定してください。使用例の出力先も未作成である必要があります。Textの制御文字escape、JSONの値とCanonical JSONの契約は維持します。stderrの診断は1件につき1行です。パス、revision、flag名、tool出力に含まれる制御文字、Unicodeの行区切り、双方向制御文字、不正なUTF-8は、`\n`、`\x1b`、`\u202e`のようなescapeで表示します。そのため、悪意のあるファイル名でログ行を偽装したり端末を操作したりすることはできません。これらの文字を含まない名前の表示は変わりません。

AdapterはGo Profileの汎用Graphを生成します。Nodeは`package`と`test`、Edgeは`prod_import`（package → package、attenuation 1）、`test_import`（package → test、attenuation 1）、`own_test`（package → test、attenuation 0）です。BがAをimportすると、EdgeはAからBに向きます。testノードから外向きのEdgeはありません。Adapterは収集したGraphごとにこのProfileを検証し、違反時はexit 1で失敗します。汎用Coreはこの制約を`simulate`に課しません。

通常のGoソース変更はpackageを3、テストのみの変更はtestを2でSeedします。テスト変更だけでproduction側へ逆伝播しません。`go.mod`または`go.sum`変更は全packageを3にします。embed対象は所有するpackageまたはtestをSeedします。明確に無関係な文書は除外した事実を記録し、不明な変更はincompleteとします。

現在の`go list`が報告するassembly、C/C++/Objective-C/Fortran、header、SWIG、system object入力は所属packageを3でSeedします。現在のビルド条件で除外された入力や所有先不明の入力はincompleteです。到達可能なversion付きremote replaceはcacheで正常解決できる場合に対応します。到達可能なdirectory replaceや未解決のlocal/remote依存はincompleteです。未使用replace（存在しないdirectoryを含む）だけではincompleteにしません。対象は現在の`./...`から到達するdependency/test packageで、全directiveではありません。workspace/nested moduleの制約は別途適用します。`go.mod`変更時は未使用directiveでも全packageをSeedします。

A → B → C → DでAを3にすると、最終Riskは3、2、1、0です。自身のテストも同じ値になります（[例](testdata/example.json)）。テスト対象はRisk降順、同点ではID昇順です。テストは自動実行しません。

## 決定論と制約

すべてのGit子プロセスへ`GIT_NO_LAZY_FETCH=1`、`GIT_TERMINAL_PROMPT=0`、`GIT_OPTIONAL_LOCKS=0`を指定し、親の値を上書きします。親環境は変更しません。直接のGit呼び出しには`--no-lazy-fetch`も付けます。Trace2を含む`GIT_TRACE*`を子環境から除外し、診断traceによる誤incompleteやtraceファイルへの書き込みを防ぎます。実際の読み取りエラーはwarningへ残します。Goのbuildvcsが起動するGitにも同じ環境を継承します。これはGit/Go取得制御でありOS通信隔離ではありません。インストール済みtool、repository設定のhelper、credential helper、wrapperは信頼対象です。

base embed解析はNUL区切りの`ls-tree`でbaseツリーを列挙し、通常の全`.go` blobをOID指定の`cat-file blob`で1つずつ読み取ります。欠落blobがあっても別ソースが一致するとexit 0になる`git grep`への依存を解消しました。存在確認だけでは内容の読み取り成功を証明できません。全必要ソースの読み取り・対象コメント解析が成功するまでbase分類を採用せず、各失敗を診断します。有効な現在グラフがある場合はincomplete/exit 2、継続不能ならerror/exit 1です。base完全性が未確認の未所有文書は`excluded_document`でなく`unknown`とします。baseツリー上の`.go`で終わるsymlinkも未対応のbaseソースとしてbase embed解析をincompleteにします。完全なbaseでは従来の`former_embed`分類を維持します。partial cloneは解析に必要なcommit/treeと全base Go blobをローカルに保持する必要があります。欠落を補完せず、暗黙fetchしません。混在欠落fixtureでobject store不変とfetch試行0を検証しています。

JSONとTextは同じ出力パス検査を使用します。既存directoryは末尾セパレーターの有無にかかわらず拒否し、directory/dangling symlinkと`.`・`..`も拒否します。OSのパス区切り文字で終わる引数は、存在しないパスも含め正規化前に拒否します。末尾区切りを削除して別の出力先へ読み替えません。新規ファイル限定のatomic No-Clobber公開は維持します。

親パスは元の表記のまま解決します。`nx`が存在しない`nx/../report.json`はエラーとなり、directory symlinkに続く`..`はsymlink先を基準に解決します。実際の出力先にある既存entryも保護します。安定した信頼できるdirectoryが必要という制約は残ります。

次のP3は未修正で、v0.1.1の対応候補です。

| Finding | 現在の動作と回避策 |
|---|---|
| NF-03 | 無関係な文書を含む全tracked symlinkを検査します。directory/dangling targetはerror、通常targetはincompleteです。通常ファイルのcheckoutを使用してください。 |
| NF-04 | ignored `*.go` FIFOで`go list`が無期限待機する場合があります。解析前に特殊ファイルを除去してください。実測ではcontext cancellation・SIGINT・SIGTERMで直接のGo子プロセスが終了しました。親だけをSIGKILLするとGoが残留し、殺された親自身はcleanupできません。共有・可変CIはTERMを先行し、強制終了時には専用jobのprocess group/container全体を対象にしてください。FIFOはGit cloneで配布されませんが、準備工程やgeneratorで作成できます。P3として延期し、混入可能なCIではP2相当の運用リスクとして扱います。v0.1.1候補は特殊入力preflight・期限・OS別process group制御です。 |
| NF-05 | 公開後cleanup失敗は「公開済み」と明示してexit 1となり、一時hardlinkが残る場合があります。完全な出力を確認し、権限回復後に残留linkを除去してください。 |
| NF-06 | 長い出力basenameは一時suffix追加でNAME_MAXを超える場合があります。短い名前を使用してください。 |
| NF-07 | Goが除外する`testdata`配下でもtracked nested `go.mod`はincompleteです。制約を受け入れるか、解析用checkoutでfixture moduleファイルをrename/untrackしてください。 |
| NEW-01の処理時間 | base embed解析はbaseの`.go`ファイルごとに`git cat-file blob`を1回起動します。Apple M1実測で100 / 1,000 / 10,000ファイルが約2.2秒 / 20.8秒 / 182秒です。大規模repositoryではCIの時間枠に考慮してください。v0.1.1候補は監査済みのstream型`cat-file --batch`読み取りです。 |

実測結果、互換性の範囲、対応候補は[VALIDATION.md](docs/VALIDATION.md)に記載します。

同じ正規化Snapshotから、同じ状態、Trace、世代数、SHA-256、JSONバイト列を生成します。入力収集はHEAD、ツールチェーン、ローカル依存関係、ビルド条件に依存します。エンジンは時刻、乱数、ネットワーク、浮動小数点Risk、並列更新を使用しません。

全Go解析子プロセスの環境を共通builderで固定します。`GOPROXY=off`、`GONOPROXY=none`、`GOVCS=*:off`、`GOSUMDB=off`、`GOTOOLCHAIN=local`、`GOWORK=off`、`GO111MODULE=on`、`GOENV=off`、`GONOSUMDB=none`と空の`GOFLAGS`/`GOEXPERIMENT`/`GOPRIVATE`/`GOINSECURE`を適用し、listに`-mod=readonly`を指定します。private設定によるproxy迂回、直接VCS取得、toolchain自動取得を防ぎ、保存済みGo設定やflags注入を無効にします。親GOFLAGS/GOEXPERIMENTが空でなければincompleteのwarningを残します。明示的なGOOS、GOARCH、CGO_ENABLED、cacheパスを保持し、親環境や永続設定は変更しません。cache済み依存は利用でき、欠落はエラーまたはincompleteです。これはGo取得ポリシーでありOSの通信隔離ではありません。インストール済みtool、wrapper、外部cache helperは信頼対象です。Snapshot上限とHashは[SCHEMA.md](docs/SCHEMA.md)に記載します。

静的importだけでは全実行時依存を捕捉できません。動的ロード、生成コード、異なるbuild tags、workspace、ネストしたモジュール、ローカルreplaceには保守的な制約があります。全プラットフォームではなく現在のターゲット環境と標準ビルド条件を解析します。削除ファイルや消えたテストの所有先を特定できない場合はincompleteです。テスト対象0件は検証不要の証明ではありません。他のソフトウェアより品質が高いという主張は行いません。

解析中のcheckoutと出力directoryは安定した信頼できる状態を保ってください。解析検査はfilesystem transactionではありません。Unix入力はnonblockingでidentityを再確認し、他環境には事前検査/openの競合が残ります。出力の親directoryを解決し公開前にidentityを再検査しますが、悪意ある親directoryや一時名の差し替えをportable path APIだけでは完全に排除しません。directoryと公開entryが維持される条件では同時writerの成功は高々1件です。公開後のcleanup失敗は「公開済み」と明示してexit 1になります。exit 1でも出力が存在する場合があります。cleanup失敗やcrashで一時hardlinkが残る可能性があり、directory fsync/crash durabilityは保証しません。macOS出力とLinuxコンテナのtmpfs/overlay出力について、競合とmetadata保持を含め実測しました。Windows実行時は未検証です。Go 1.23.12は、macOS arm64のローカルとLinux arm64コンテナで、ビルドと全テストに成功しました。GitHub ActionsのLinux amd64ランナーでも全テストに成功しています。クロスコンパイルとLinuxの検証範囲は[VALIDATION.md](docs/VALIDATION.md)で区別します。

## 開発

```sh
gofmt -l .
go test ./...
go test -race ./...
go test -shuffle=on -count=10 -timeout 30m ./...   # adaptersだけで約10分かかる
go vet ./...
go build ./cmd/stateripple
git status --short   # 未追跡ファイルも確認する（HEADがまだない場合がある）
```

### 継続的インテグレーション

[`.github/workflows/ci.yml`](.github/workflows/ci.yml)は、`main`へのpushとpull request、および手動実行で動きます。検証だけを行います。tokenは読み取り専用（`contents: read`）で、公開処理はなく、使うActionは公式の`actions/checkout`と`actions/setup-go`のみで、どちらもcommit SHAに固定しています。コマンド、timeout、オフライン検査の範囲、初回Release前の必須ゲートは[docs/CI.md](docs/CI.md)（英語）にあります。

- **Go。** サポート対象は1.26と1.27です。1.23はStateRippleがビルド・テストできる最低versionで、Goのセキュリティサポート対象外のため、後方互換性の確認にだけ使います。3つとも実際のtoolchainを使い、`GOTOOLCHAIN=local`で自動切り替えを防ぎます。
- **Git。** `analyze`を実行するもの（Gitを使うテストを含む）にはGit 2.45以上が必要です。CIは最初のステップで確認し、足りなければ明確に失敗します。versionは数値で比較し、skipはしません。
- **オフライン。** `GOPROXY=off`や`GIT_NO_LAZY_FETCH=1`などはGo/Gitの取得制御であり、ネットワーク遮断ではありません。ビルド済みバイナリは、空のLinux network namespaceの中で、GitもGoも見えない状態にして`simulate`、`help`、`version`、Goldenのhash、終了コードを検証します。実際の`analyze`は別のnamespaceで実行します。
- **実行テストとクロスコンパイル。** CIがテストを実行するのはLinuxだけです。他OS向けのビルドは、コンパイルできることの確認であり動作の確認ではありません。macOS、Windows、BSDでの実行はCIの対象外です。
- **状態。** workflowは、`main`へのpush、手動`workflow_dispatch`（long shuffleを含む）、`pull_request`の3経路すべてでGitHub Actions上で成功しています。証拠とRelease前チェックリストは[docs/CI.md](docs/CI.md)と[docs/VALIDATION.md](docs/VALIDATION.md)にあります。PR単位のconcurrency取消しは未観測です（追い越されたRunはありません）。

[設計](docs/ARCHITECTURE.md)、[スキーマ](docs/SCHEMA.md)、[検証証拠](docs/VALIDATION.md)を参照してください。Mandalaは開発上の宣言済み必須項目の追跡にだけ使用します。StateRipple本体の依存関係やドメインモデルには含めません。

## 将来の候補

- v0.1.1（信頼性と性能）：base embed解析のstream型`git cat-file --batch`読み取り、Git子プロセス数の削減、FIFOのpreflightと期限、tracked symlinkとnested fixtureの扱いの改善。
- v0.2.0（入力の拡張）：stdinからのSnapshot、Graph/Seedの明示入力、Changed Files Adapter、Git AdapterとGo Adapterの分離、構造化Diagnostics、明示的なcompletenessオブジェクトを持つReport v2、`test_targets`の投影とテスト向け文言を汎用CoreからAdapter側の結果へ移す責務分離。
- それ以降：`ruleset`による複数Ruleの選択、完了基準を宣言する非単調Rule・Conway型Rule、多状態セル、AIエージェント向けの構造化入出力。

いずれも候補であり、実装済み機能ではありません。v0.1.0のRuleは1つ（RiskPropagationV1）、入力形式は1つ（Snapshotファイル。stdinは未対応）、Adapterは1つ（Git + Go）です。コード修正、テスト実行、AI API接続、動的プラグインロードは行いません。

## ライセンス

[MIT](LICENSE)、copyright 2026 cottondesu。初期OSS向けの簡潔で許容的なライセンスとして選択しました。
