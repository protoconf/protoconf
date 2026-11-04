---
phase: quick-261104-ikr
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - compiler/starproto/any.go
  - compiler/starproto/message.go
  - compiler/starproto/any_test.go
  - .planning/quick/261104-ikr-skip-decode-in-any-unpack-for-anys-packe/bench-before.txt
  - .planning/quick/261104-ikr-skip-decode-in-any-unpack-for-anys-packe/bench-after.txt
autonomous: true
requirements: [QUICK-261104-ikr]

estimate:
  tokens: 60000
  raw_tokens: 60000
  tasks: 2
  confidence: low

must_haves:
  truths:
    - "any.unpack of an Any made by any.new on the same Starlark thread gives a target that dynamic.Equal-matches decoding the Any's current value bytes into the same target, whether the target starts empty or pre-populated"
    - "Mutating the source message, including nested messages reached through repeated fields, after any.new does not change what any.unpack returns"
    - "An Any whose value bytes were replaced, an Any made on another thread, or an unpack into a different message type takes the existing UnmarshalMerge path, with the same result or error as before"
    - "An Any stored in a parent message field and read back, or carried inside another Any through a pack/unpack round trip, still hits the memo"
    - "No package-level memo map exists. The memo lives in starlark.Thread locals and is collected with the thread"
    - "BenchmarkAnyRoundTrip before/after ns/op, B/op, allocs/op are recorded in SUMMARY, and the production change is kept only if ns/op drops by at least 25% with allocs/op not higher"
  artifacts:
    - path: compiler/starproto/any.go
      provides: "per-thread content-keyed memo, cloneDynamic, memo-aware newAny/unpackAny"
    - path: compiler/starproto/message.go
      provides: "toDynamicPb returning the ConvertTo error; ToDynamicPb signature unchanged"
    - path: compiler/starproto/any_test.go
      provides: "Starlark harness, equivalence tests, memo hit/miss assertions, BenchmarkAnyRoundTrip"
  key_links:
    - from: "newAny"
      to: "thread.SetLocal(anyMemoKey)"
      via: "snapshot stored under string(any.GetValue()) only when ConvertTo and anypb.New both succeeded"
      pattern: "anyMemoKey"
    - from: "unpackAny"
      to: "target.msg.MergeFrom(snapshot)"
      via: "lookup by current value bytes plus descriptor pointer equality, entry deleted on use"
      pattern: "MergeFrom"
---

<objective>
Make `any.unpack` skip the jhump wire decode when the Any's bytes came from `any.new` on the same Starlark thread. It merges a remembered snapshot into the target instead of calling `target.msg.UnmarshalMerge(bytes)`. Measure before and after, and ship only on a clear win.

Purpose: in the user's real repo, hook chains (Component, chain, run_next, wrapper, mutate_config) pack and unpack the config at every step. In the Starlark profile `unpack` is 26% of flat time, and the cost is jhump decode: one golang/protobuf legacy `proto.Unmarshal` hop per nested message (codec/decode_fields.go:196), plus the allocation and GC that come with it.

Output: memo-aware `newAny`/`unpackAny` in compiler/starproto/any.go, an error-returning `toDynamicPb` in message.go, tests and a benchmark in any_test.go, and before/after benchmark files.
</objective>

<design_notes>
These notes record what the planner verified in the code. The executor should not re-derive them.

- Equivalence: jhump `(*Message).unmarshal` (dynamic/binary.go:185) calls `mergeField(m, fd, val)` once per decoded field. `MergeFrom(*dynamic.Message)` (dynamic_message.go:2448) calls the same `mergeField(m, fd, v)` once per stored value of the source. So if the snapshot's stored values equal what decoding the bytes would produce, `target.MergeFrom(snapshot)` and `target.UnmarshalMerge(bytes)` give the same target. That includes merge semantics: scalars overwrite, repeated fields append, map keys replace, singular nested messages merge through TryMerge, and setting one oneof member clears the others.
- Aliasing: jhump's dynamic-to-dynamic merge does not deep-copy. When the target has no existing value it stores the source's nested `*dynamic.Message` pointers and repeated elements as they are (merge.go `internalSetField`/`addRepeatedField`). So a snapshot is handed to exactly one target and is deleted from the memo when used. A second unpack of the same bytes decodes as it does today.
- Source isolation: `Attr` wraps the same nested `*dynamic.Message` pointers (field.go:94-95), so Starlark can mutate the source tree in place after `any.new`. The snapshot must be a deep clone taken inside `newAny`.
- Memo key: the Any's value bytes (`string(any.GetValue())`), not the Any's pointer. The key validates itself, because a hit means the current bytes are equal. It also survives the Any being stored in a parent field, read back, or carried inside another packed Any, since the inner bytes are kept verbatim. type_url is not part of the key because decode ignores it. Only the target descriptor matters, and that is checked by pointer equality.
- Lifetime and concurrency: the memo is stored with `thread.SetLocal`. The compiler creates a fresh thread for each `main()` call (compiler/lib/config.go:43), so the memo is collected with that thread. A Starlark thread runs on one goroutine and parallel compiles use separate threads, so no lock is needed.
- Why the message factory matches: every protoconf message is built with `dynamic.NewMessage(md)` (message_type.go:106, nil factory), and decode creates nested messages the same way. The clone must also use `dynamic.NewMessage(fd.GetMessageType())`.
- The existing `attrCache` staleness on a target that was read before unpack is the same in both paths and is out of scope.
</design_notes>

<execution_context>
@/Users/smintz/go/src/github.com/protoconf/protoconf/.claude/gsd-core/workflows/execute-plan.md
@/Users/smintz/go/src/github.com/protoconf/protoconf/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@./CLAUDE.md
@.planning/STATE.md
@compiler/starproto/any.go
@compiler/starproto/message.go
@compiler/starproto/any_test.go
@compiler/starproto/message_type.go
</context>

<tasks>

<task type="tracer">
  <name>Task 1: Starlark any.new to any.unpack harness, equivalence tests, BenchmarkAnyRoundTrip, baseline numbers</name>
  <files>compiler/starproto/any_test.go, .planning/quick/261104-ikr-skip-decode-in-any-unpack-for-anys-packe/bench-before.txt</files>
  <read_first>compiler/starproto/any_test.go (callNewAny/callUnpackAny each make a fresh thread, so existing tests keep exercising the decode path), compiler/starproto/field_test.go lines 33-60 (desc.LoadFileDescriptor("google/protobuf/descriptor.proto") pattern), compiler/starproto/message_type.go (NewMessageType gives a Starlark constructor)</read_first>
  <action>
Tracer: this harness is the end-to-end path (Starlark source, then the any builtins, then the jhump messages) that every later change is measured and verified through. It must pass on the current, unmodified production code. Do not touch any.go or message.go in this task.

Append the following to compiler/starproto/any_test.go. Do not create a new file.

1. A helper that runs Starlark source on a caller-supplied `*starlark.Thread` with `starlark.ExecFile` and returns the globals. Its predeclared dict holds `any` (AnyModule) and `NewMessageType` constructors for google.protobuf FileDescriptorProto, DescriptorProto, FieldDescriptorProto, Struct, Value, Duration, and Option. Option is the type.proto message whose `value` field is an Any. Load each descriptor with `desc.LoadFileDescriptor` plus FindMessage. If type.proto is not registered, add a blank import of google.golang.org/protobuf/types/known/typepb to the test file. Starlark forbids top-level for/if, so put loops inside def or comprehensions.
2. A shared Starlark prelude string. Its `build()` returns FileDescriptorProto(name="bench.proto", package="bench", dependency=["a.proto","b.proto"], message_type=[20 DescriptorProto, each with 10 FieldDescriptorProto (name, number, json_name) and one nested_type DescriptorProto with one field]). Its `chain(cfg, steps)` loops `steps` times doing `a = any.new(cfg)`, then `cfg = any.unpack(a, FileDescriptorProto())`, then `cfg.package = "bench%d" % i`, and returns cfg. This mirrors the hook-chain pack, unpack, mutate pattern.
3. A Go helper that takes a Starlark Any value (`*starProtoMessage`), reads its "value" bytes with TryGetFieldByName, and returns them. Also a reference decoder that UnmarshalMerges those bytes into a given base `*dynamic.Message` and returns the error.
4. Behavioural tests, all named TestAnyUnpackMemo_*. Each runs new and unpack on ONE shared thread so a memo can hit later, and each compares against the reference decoder with `dynamic.Equal`:
   - HitMatchesDecode: `out = any.unpack(any.new(build()), FileDescriptorProto())` equals the reference decode of the Any's bytes into a fresh FileDescriptorProto.
   - SourceMutatedAfterNew: after `a = any.new(src)`, run `src.package = "mutated"` and `src.message_type[0].name = "X"`. The second is an in-place nested mutation through a repeated element. Then unpack. Assert package == "bench" and message_type[0].name == "M0", and that the result equals the reference decode.
   - ValueBytesReplaced: on a second thread, create an Any of build() with package set to "other". On the main thread, run `a = any.new(build())`, then `a.value = other.value`, then unpack. Assert package == "other" and equality with the reference decode of other's bytes.
   - MergeIntoPopulatedTarget, two cases. (a) The target is FileDescriptorProto(name="keep.proto", dependency=["pre.proto"], message_type=[DescriptorProto(name="Pre")]). Expect name "bench.proto", dependency ["pre.proto","a.proto","b.proto"], and 21 message_type entries starting with "Pre". (b) Singular nested-message merge: the target is Value(struct_value=Struct(fields={"pre": Value(string_value="x")})) and the source is Value(struct_value=Struct(fields={"a": Value(number_value=1)})). Expect both keys. In each case, build a second identical pre-populated message in the same snippet as the reference base, and assert dynamic.Equal(reference, out).
   - TypeMismatch: unpack a FileDescriptorProto Any into Duration(). Assert that the builtin's error-or-nil matches the reference decoder's error-or-nil for the same bytes into a fresh Duration. If both are nil, also assert dynamic.Equal.
   - AnyInParentField: run `h = Option(name="o", value=any.new(build()))`, then `out = any.unpack(h.value, FileDescriptorProto())`. Also do the nested round trip `h2 = any.unpack(any.new(h), Option())` followed by `out2 = any.unpack(h2.value, FileDescriptorProto())`. Both must equal the reference decode.
5. BenchmarkAnyRoundTrip. Before timing, ExecFile the prelude once on a setup thread, call build() once, and take `chain`. Call b.ReportAllocs, then loop with `for b.Loop()`. Each iteration creates a fresh `&starlark.Thread{}`, which mirrors the per-main() thread lifetime, and calls chain(cfg, 5). Use b.Fatal on error.

Then record the baseline on the unmodified production code: run the benchmark with -count=5 and tee the output into bench-before.txt in the quick directory.
  </action>
  <verify>
    <automated>cd /Users/smintz/go/src/github.com/protoconf/protoconf && git diff --quiet -- compiler/starproto/any.go compiler/starproto/message.go && go test -race -timeout 300s -run 'TestAnyUnpackMemo|TestAny' ./compiler/starproto/ && go test -run '^$' -bench BenchmarkAnyRoundTrip -benchmem -count=5 ./compiler/starproto/ | tee .planning/quick/261104-ikr-skip-decode-in-any-unpack-for-anys-packe/bench-before.txt && grep -c 'BenchmarkAnyRoundTrip' .planning/quick/261104-ikr-skip-decode-in-any-unpack-for-anys-packe/bench-before.txt</automated>
  </verify>
  <done>Every TestAnyUnpackMemo_* test passes on unmodified production code. bench-before.txt holds 5 BenchmarkAnyRoundTrip lines with ns/op, B/op, allocs/op. any.go and message.go are unchanged. The work is committed as a test commit.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Per-thread content-keyed memo in any.new/any.unpack, measure, ship-or-revert gate</name>
  <files>compiler/starproto/any.go, compiler/starproto/message.go, compiler/starproto/any_test.go, .planning/quick/261104-ikr-skip-decode-in-any-unpack-for-anys-packe/bench-after.txt</files>
  <behavior>
    - After `a = any.new(x)` on thread T, T's memo holds 1 entry. After `any.unpack(a, X())` on T it holds 0 (hit and consumed).
    - SourceMutatedAfterNew and MergeIntoPopulatedTarget still equal the reference decode with the memo active.
    - ValueBytesReplaced: the entry for a's original bytes is still there after unpack (miss, so decode ran).
    - TypeMismatch: the FileDescriptorProto entry is still there (the descriptor-pointer check refused it), and the error or nil matches the reference as before.
    - AnyInParentField: both out and out2 are memo hits. After the nested round trip, the memo holds only entries that were never unpacked.
    - Unpacking the same Any twice: the first unpack hits, the second decodes, and both results are dynamic.Equal.
  </behavior>
  <action>
RED: in any_test.go, add a test-only helper that reads `thread.Local(anyMemoKey)` and returns the map length, or 0 if none. Extend the Task 1 tests with the memo assertions from the behavior block, and add a TestAnyUnpackMemo_DoubleUnpack test. Run them and confirm they fail because anyMemoKey does not exist yet.

GREEN, message.go: add an unexported `toDynamicPb(msg *dynamic.Message) (*dynamicpb.Message, error)` that does today's ConvertTo, keeps the existing slog.Error line as the single logging site, and returns the error. The exported ToDynamicPb keeps its signature and behaviour by calling it and dropping the error. mutate/mutate.go and compiler/lib/compiler.go call ToDynamicPb and must not change. AsDynamicPb stays as it is.

GREEN, any.go:
- Add an unexported const anyMemoKey (a namespaced string such as "protoconf.starproto.anyMemo"). The memo value stored in thread locals is a `map[string]*dynamic.Message` keyed by Any value bytes.
- newAny: replace `msg.AsDynamicPb()` with `toDynamicPb(msg.msg)`, keeping the converted message and the error. On error, carry on exactly as today: build the Any and return it. Only after anypb.New and dynamic.AsDynamicMessage succeed, and only if the conversion error was nil and the thread is non-nil, call `cloneDynamic(msg.msg, msg.msg.GetMessageDescriptor())`. If that returns ok, store the clone at memo[string(any.GetValue())], creating the map with SetLocal on first use. A later identical-bytes entry overwriting an earlier one is fine, because equal bytes mean equal snapshots.
- unpackAny: after the existing extraction of the value bytes `s`, if the thread is non-nil and its memo has an entry for string(s) whose GetMessageDescriptor() is pointer-equal to target.msg.GetMessageDescriptor(), delete the entry first, then call target.msg.MergeFrom(snapshot). Wrap any error with the same "failed to unmarshal any: %v" format and return target. Otherwise run the existing UnmarshalMerge path unchanged. That covers a missing entry, a descriptor mismatch, and a nil thread, and keeps type-mismatch errors as they are.
- `cloneDynamic(src *dynamic.Message, md *desc.MessageDescriptor) (*dynamic.Message, bool)` returns ok=false, which means don't memo and let decode run, when any of these hold:
  - src.GetMessageDescriptor() != md (pointer).
  - md.IsExtendable(). Extensions are not reachable through regular fields.
  - len(src.GetUnknownFields()) > 0. jhump MergeFrom drops unknown fields.
  - Any field in src.GetKnownFields() has IsRequired(). A nested decode runs Validate(), and the fast path would not.
  - A TryGetField or TrySetField call errors.
  - A nested message value is not a `*dynamic.Message`.

  Otherwise it builds `out := dynamic.NewMessage(md)`. For each known field where src.HasField(fd), it gets the value (GetField already returns defensive copies of maps and slices), deep-copies it, and TrySetFields it on out:
  - Maps: clone message-typed map values through fd.GetMapValueType().GetMessageType().
  - Repeated: clone each message element through fd.GetMessageType().
  - Singular message: recurse with fd.GetMessageType(). The pointer check above then rejects descriptor drift.
  - []byte: copy.
  - Other scalars: keep as they are.
- Doc comment on the memo, at most 6 lines. State the invariant: a snapshot equals decoding its key bytes into its descriptor. State why it is one-shot: MergeFrom aliases nested messages into the target. Point to the jhump mergeField equivalence. Add a `ponytail:` comment that the map has no entry cap, because it is bounded by one thread (one main() call), and that a cap should be added if a Starlark thread ever becomes long-lived.

MEASURE: run the full compiler suite with -race, then the benchmark with -count=5, teeing into bench-after.txt. Compare the mean ns/op and allocs/op from bench-before.txt and bench-after.txt.

GATE: keep the change only if mean ns/op is at least 25% lower and allocs/op is not higher. If the gate fails, run `git checkout -- compiler/starproto/any.go compiler/starproto/message.go`. Then remove only the memo-internal assertions and the anyMemoKey helper from any_test.go, so the equivalence tests and the benchmark stay. Rerun the suite, and record "not shipped" with the numbers.

SUMMARY: include a before/after table of mean ns/op, B/op, allocs/op and the ship decision.

Run gofmt -l and go vet on ./compiler/starproto/ before committing as a perf commit.
  </action>
  <verify>
    <automated>cd /Users/smintz/go/src/github.com/protoconf/protoconf && test -z "$(gofmt -l compiler/starproto/)" && go vet ./compiler/starproto/ && go test -race -timeout 300s -skip 'Test_cliCommand_Run' ./compiler/... && go test -run '^$' -bench BenchmarkAnyRoundTrip -benchmem -count=5 ./compiler/starproto/ | tee .planning/quick/261104-ikr-skip-decode-in-any-unpack-for-anys-packe/bench-after.txt && grep -c 'BenchmarkAnyRoundTrip' .planning/quick/261104-ikr-skip-decode-in-any-unpack-for-anys-packe/bench-after.txt && ! grep -nE '^var .*map\[string\]\*dynamic\.Message' compiler/starproto/any.go</automated>
  </verify>
  <done>The ./compiler/... suite is green under -race. Every TestAnyUnpackMemo_* test passes, including the memo hit/miss assertions if the change shipped. bench-after.txt is recorded. SUMMARY holds the before/after table and the ship decision against the 25% gate. No package-level memo map exists. ToDynamicPb's exported signature is unchanged.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Starlark config code to compiler builtins | User-authored .pconf/.mpconf/.pinc code calls any.new/any.unpack and can mutate sources, rewrite Any value/type_url, and unpack into any type |

## STRIDE Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation Plan |
|-----------|----------|-----------|----------|-------------|-----------------|
| T-q261104-01 | Tampering | unpackAny memo hit returns stale or wrong config | high | mitigate | Key on the current value bytes, require descriptor-pointer equality, take a deep clone at any.new, delete the entry on use, and apply the cloneDynamic refusal guards (unknown fields, extendable, required, descriptor drift, ConvertTo error). Covered by the SourceMutatedAfterNew, ValueBytesReplaced, TypeMismatch, MergeIntoPopulatedTarget, and DoubleUnpack tests |
| T-q261104-02 | Denial of Service | Snapshots for Anys that are never unpacked stay in memory | low | accept | The memo is collected with the per-main() Starlark thread. Entries are deleted on hit. A ponytail comment names the cap upgrade path |
| T-q261104-03 | Tampering | Data race on a shared memo during parallel errgroup compiles | medium | mitigate | No package-level map. The memo is stored in thread.Local, and each Starlark thread runs on one goroutine. Verified with the -race suite and the negative grep for a package-level map |
</threat_model>

<verification>
- `go test -race -timeout 300s -skip 'Test_cliCommand_Run' ./compiler/...` is green. Do not run a bare `./...`, which hangs on Consul.
- bench-before.txt and bench-after.txt both hold 5 BenchmarkAnyRoundTrip lines.
- The ship decision in SUMMARY follows the 25% ns/op gate with allocs/op not higher.
</verification>

<success_criteria>
- If shipped: hook-chain-style pack, unpack, mutate cycles skip the jhump decode for same-thread Anys, with results provably equal to decoding, and BenchmarkAnyRoundTrip at least 25% faster.
- If not shipped: production files are reverted, and the equivalence tests and benchmark stay as regression and measurement tooling, with the numbers recorded.
</success_criteria>

<output>
Create `.planning/quick/261104-ikr-skip-decode-in-any-unpack-for-anys-packe/261104-ikr-SUMMARY.md` when done, with the before/after benchmark table and the ship decision.
</output>
