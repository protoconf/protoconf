---
phase: quick-260901-vaj
plan: 01
type: execute
wave: 1
depends_on: []
autonomous: true
requirements: [QUICK-260901-vaj]

files_modified:
  # Certain
  - go.mod
  - go.sum
  - .github/workflows/go.yml
  - .github/workflows/codeql-analysis.yml
  - .github/workflows/release.yml
  - .planning/quick/260901-vaj-merge-5-stale-security-dependency-bumps-/baseline-tests.txt
  - .planning/quick/260901-vaj-merge-5-stale-security-dependency-bumps-/after-tests.txt
  # CONDITIONAL — authorized ONLY for files the Go toolchain names in a compile
  # error or a NEW test failure. Not enumerable before Task 2 runs. See <scope_authority>.

estimate:
  tokens: 55000
  raw_tokens: 40000
  tasks: 3
  confidence: low

must_haves:
  truths:
    - "`go build ./...` succeeds with all five target module versions present in go.mod"
    - "`go test ./...` produces no failing package that is absent from the pre-upgrade baseline"
    - "Every CI job installs a Go toolchain that satisfies the go.mod go directive"
    - "No dependency is bumped beyond what the five targets require transitively"
  artifacts:
    - go.mod
    - go.sum
    - .planning/quick/260901-vaj-merge-5-stale-security-dependency-bumps-/baseline-tests.txt
    - .planning/quick/260901-vaj-merge-5-stale-security-dependency-bumps-/after-tests.txt
  key_links:
    - "otel core / sdk / sdk-metric / metric / trace / otlp exporters must ALL land on v1.43.0 together — a split version set does not compile"
    - "go.opentelemetry.io/contrib otelgrpc is version-paired to otel core; v0.68.0 is the release whose go.mod requires otel v1.43.0"
    - "grpc v1.82.1 forces google.golang.org/protobuf to >= v1.36.11, which the compiler stack (jhump/protoreflect, bufbuild/protocompile, bufbuild/protovalidate-go) must tolerate"
    - "go.mod go directive rises to >= 1.25.8 (go-getter v1.8.6 floor); every CI setup-go step must satisfy it"
---

<objective>
Land the five stale security dependency bumps (renovate/snyk PRs #490, #493, #494, #489, #507) on the existing `deps/security-bumps` branch, with a green build and a test suite no worse than before the bump.

Purpose: five security advisories have sat unmerged because each PR bumps one module in isolation and they conflict on the shared transitive graph (protobuf, golang.org/x/*, the otel family). Merging them as one coordinated `go get` resolves that graph once instead of five times.

Output: updated `go.mod`/`go.sum` at the five target versions, CI workflows pinned to a toolchain satisfying the new go directive, and a before/after test record proving no regression.
</objective>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.claude/gsd-core/workflows/execute-plan.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@CLAUDE.md
@go.mod
</context>

<scope_authority>
Concrete edit scope CANNOT be known before Task 2 runs `go build ./...`. Therefore:

**Authorized outright:** `go.mod`, `go.sum`, the three `.github/workflows/*.yml` files, and the two test-record files in the quick directory.

**Authorized conditionally:** any source file the Go toolchain *names* in a compile error, or that owns a test which newly fails. Fix the minimum in that file to restore pre-upgrade behavior.

**Not authorized:** touching a file because it looks related, refactoring while in there, or upgrading a module the toolchain did not demand. If `go build ./...` and `go test ./...` never print a file's path, that file is out of scope.

Branch discipline: the tree is already on `deps/security-bumps`, forked from origin/main at 7fdffc4. Do NOT create another branch.
</scope_authority>

<tasks>

<task type="auto">
  <name>Task 1: Capture the pre-upgrade baseline</name>
  <files>.planning/quick/260901-vaj-merge-5-stale-security-dependency-bumps-/baseline-tests.txt</files>
  <precondition>Working tree is on branch `deps/security-bumps` and `go.mod`/`go.sum` are unmodified relative to HEAD.</precondition>
  <action>
BEFORE changing any dependency version, record the current state of the suite so upgrade-induced failures can be told apart from pre-existing ones.

Confirm the branch with `git branch --show-current` and confirm `git status --porcelain go.mod go.sum` is empty. If either check fails, halt and report — dirty module files make the baseline meaningless.

Run `go build ./...`, then `go test ./... 2>&1 | tee '.planning/quick/260901-vaj-merge-5-stale-security-dependency-bumps-/baseline-tests.txt'`. Skip `-race` here; the goal is a fast, comparable per-package pass/fail record, and CI's race run is a separate concern.

Append to the end of that same file a section headed `## Pre-existing failures` listing every output line beginning with FAIL, one per line, plus the output of `go version`. If the list is empty, write `none` under that heading. One skip is already expected upstream (`load_remote_with_load_local`, stale vizceral_repo module pin) — skips are not failures and do not belong in the list.

Change no other file. Do not run `go get`, do not run `go mod tidy`, do not edit `go.mod` in this task.
  </action>
  <verify>
    <automated>test -s '.planning/quick/260901-vaj-merge-5-stale-security-dependency-bumps-/baseline-tests.txt' && grep -q '## Pre-existing failures' '.planning/quick/260901-vaj-merge-5-stale-security-dependency-bumps-/baseline-tests.txt' && git diff --quiet go.mod go.sum</automated>
  </verify>
  <done>baseline-tests.txt holds the full `go test ./...` output plus an explicit `## Pre-existing failures` list, and go.mod/go.sum remain untouched.</done>
</task>

<task type="auto">
  <name>Task 2: Apply the five bumps, tidy, and restore a clean build</name>
  <files>go.mod, go.sum (plus CONDITIONAL source files per scope_authority)</files>
  <precondition>`go env GOTOOLCHAIN` is not `local` — the new go directive requires Go >= 1.25.8 while the local toolchain is 1.25.1, so Go must be allowed to auto-download the newer toolchain.</precondition>
  <action>
Bump all five targets in ONE `go get` invocation so version selection resolves the shared transitive graph a single time rather than five conflicting times:

`go get google.golang.org/grpc@v1.82.1 golang.org/x/net@v0.55.0 github.com/go-git/go-git/v5@v5.19.2 github.com/hashicorp/go-getter@v1.8.6 go.opentelemetry.io/otel@v1.43.0 go.opentelemetry.io/otel/sdk@v1.43.0 go.opentelemetry.io/otel/sdk/metric@v1.43.0 go.opentelemetry.io/otel/metric@v1.43.0 go.opentelemetry.io/otel/trace@v1.43.0 go.opentelemetry.io/otel/exporters/otlp/otlptrace@v1.43.0 go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc@v1.43.0 go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc@v1.43.0`

The otel siblings are listed explicitly and are NOT opportunistic extras: otel is a multi-module repo whose packages are version-locked to each other, and the two otlp exporter modules are distinct module paths that a bare `go get otel/sdk` leaves stranded at v1.27.0 where they will not compile.

Then run `go mod tidy`, then `go build ./...`.

Expected and accepted side effects — do not fight them:
- The go directive rises from 1.22.4 to at least 1.25.8 (go-getter v1.8.6's floor) and a `toolchain` line may appear. Both are hard requirements of the target versions, not scope creep.
- `google.golang.org/protobuf` is pulled from v1.34.1 to at least v1.36.11 by grpc v1.82.1, dragging `golang.org/x/*` and `genproto` forward with it.

Risk areas in likely order of biting. Address an item ONLY if the build actually reports it:
- `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` sits at v0.52.0, paired to otel core v1.27. If it fails against otel v1.43.0, move it to v0.68.0 — the contrib release whose go.mod requires exactly otel v1.43.0. Apply the same pairing to the indirect `contrib/instrumentation/net/http/otelhttp` if it breaks.
- The compiler stack (`github.com/jhump/protoreflect` v1.16.0, `github.com/bufbuild/protocompile` v0.10.0, `github.com/bufbuild/protovalidate-go` v0.6.2) all predate protobuf v1.36. If one fails to compile, bump only that module, to the lowest release that compiles.
- `github.com/grpc-ecosystem/go-grpc-prometheus` v1.2.0 is unmaintained and used by `agent/agent.go`. If it no longer compiles against grpc v1.82.1, report the breakage and STOP rather than swapping in a replacement library — that is a design change, not a version bump, and belongs in its own task.
- `github.com/fullstorydev/grpcui` v1.4.1 and `grpcurl` v1.9.1 pin grpc reflection behavior used by the mutation server.

Two items were pre-verified during planning and need NO edit:
- `go.opentelemetry.io/otel/semconv/v1.4.0`, imported by `observability/observability.go`, still ships inside otel v1.43.0. Leave that import as it is.
- Every gRPC client site already uses `grpc.NewClient`; there is no `grpc.Dial` call anywhere in the tree. Leave those call sites as they are.

Do not touch test files here; Task 3 owns test breakage. Do not upgrade any module the toolchain did not demand.
  </action>
  <verify>
    <automated>go build ./... && go vet ./... && test "$(go list -m -f '{{.Path}}@{{.Version}}' google.golang.org/grpc golang.org/x/net github.com/go-git/go-git/v5 github.com/hashicorp/go-getter go.opentelemetry.io/otel/sdk | grep -cE 'grpc@v1\.82\.1|net@v0\.55\.0|go-git/v5@v5\.19\.2|go-getter@v1\.8\.6|otel/sdk@v1\.43\.0')" = 5</automated>
  </verify>
  <done>`go build ./...` and `go vet ./...` both pass, and `go list -m` reports all five target modules at exactly their target versions.</done>
</task>

<task type="auto">
  <name>Task 3: Restore a green suite and align the CI toolchain floor</name>
  <files>.github/workflows/go.yml, .github/workflows/codeql-analysis.yml, .github/workflows/release.yml (plus CONDITIONAL test/source files per scope_authority)</files>
  <action>
Run `go test ./... 2>&1 | tee '.planning/quick/260901-vaj-merge-5-stale-security-dependency-bumps-/after-tests.txt'` and diff the set of failing packages against the `## Pre-existing failures` list written in Task 1.

Fix ONLY packages that fail now and did not fail in the baseline. A package that failed in the baseline stays failing and is reported, not repaired — pre-existing breakage is out of scope for this task. For each new failure, read the actual assertion and fix the smallest thing that restores the pre-upgrade behavior; if a failure turns out to be a deliberate upstream behavior change rather than a bug, note it in the summary instead of forcing the old behavior back.

Then align CI. The three workflow files each pin an explicit Go version through `actions/setup-go` (go.yml at 1.22, codeql-analysis.yml and release.yml at 1.21), all of which now sit below the go.mod floor. In each of the three files, replace that pinned-version input with `go-version-file: go.mod` so setup-go reads the floor straight from the module and cannot drift out of sync again on the next bump. This is one line per file — do not otherwise restructure the workflows.

Finally confirm the race detector run that CI actually performs still works: `go test -race ./...` on at least the packages that changed. If `-race` surfaces a failure that the plain run did not, treat it as a new failure and fix it.

Commit go.mod, go.sum, the workflow files, the two test-record files, and any conditional source fixes as one atomic commit.
  </action>
  <verify>
    <automated>test "$(grep -l 'go-version-file: go.mod' .github/workflows/go.yml .github/workflows/codeql-analysis.yml .github/workflows/release.yml | wc -l | tr -d ' ')" = 3 && go test ./... > /dev/null</automated>
  </verify>
  <done>after-tests.txt exists; every failing package in it also appears in the baseline's `## Pre-existing failures` list; all three workflow files resolve their Go version from go.mod.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| module proxy -> build | Third-party code enters the binary from proxy.golang.org |
| KV store / gRPC peer -> agent & mutation server | Untrusted network input parsed by the bumped grpc and x/net stacks |

## STRIDE Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation Plan |
|-----------|----------|-----------|----------|-------------|-----------------|
| T-vaj-01 | Denial of Service | grpc / golang.org/x/net HTTP2 handling in agent + mutation server | high | mitigate | The bump itself is the mitigation: grpc v1.82.1 and x/net v0.55.0 carry the advisory fixes these PRs were opened for |
| T-vaj-02 | Tampering | go.sum integrity across a large transitive bump | high | mitigate | `go mod tidy` regenerates go.sum against the checksum database; GONOSUMDB/GONOSUMCHECK must not be set, and GOFLAGS must not contain -mod=mod bypasses |
| T-vaj-03 | Elevation of Privilege | go-getter remote module fetch (compiler/lib/module_service.go) | high | mitigate | go-getter v1.8.6 carries the path-traversal/symlink advisory fixes; no code change needed, version floor is the control |
| T-vaj-04 | Tampering | go-git v5.19.2 repo operations in inserter/inserter.go | medium | mitigate | Version bump only; verify inserter git-metadata tests still pass in Task 3 |
| T-vaj-SC | Tampering | package-manager installs | low | accept | No NEW third-party module is introduced — every module here already exists in go.mod and only its version moves, so the package legitimacy gate does not apply |
| T-vaj-05 | Denial of Service | unmaintained go-grpc-prometheus against grpc v1.82.1 | medium | accept | Task 2 halts and reports rather than silently swapping the library; replacement is a separate design decision |
</threat_model>

<verification>
- `go build ./...` succeeds
- `go vet ./...` succeeds
- `go test ./...` has no failing package absent from baseline-tests.txt
- `go test -race ./...` agrees with the plain run
- `go list -m` shows grpc v1.82.1, x/net v0.55.0, go-git/v5 v5.19.2, go-getter v1.8.6, otel/sdk v1.43.0
- All three CI workflows read their Go version from go.mod
</verification>

<success_criteria>
The five target modules sit at their target versions on `deps/security-bumps`, the build and test suite are no worse than the recorded baseline, CI installs a toolchain that satisfies the new go directive, and no module outside the five targets moved except where the toolchain required it.
</success_criteria>

<output>
Create `.planning/quick/260901-vaj-merge-5-stale-security-dependency-bumps-/260901-vaj-SUMMARY.md` when done. Record: the final version of every module that moved (targets and transitives), any conditional source file edited and why, any pre-existing failure still failing, and any risk-area item from Task 2 that fired.
</output>
