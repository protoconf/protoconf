---
slug: mod-tidy-subdir-redownload
status: awaiting_human_verify
trigger: |
  When running the following CONFIGSPACE, every `protoconf mod tidy` downloads the entire repo even when cached:

  # CONFIGSPACE — one module per driver, so importing Grafana does not drag in ECS
  platform=remote_repo(label="platform", url="git@github.com:smintz/platform-engineering.git", tag="v0.1.0")
  kubernetes=remote_repo(label="kubernetes", url="git@github.com:smintz/platform-engineering.git//drivers/runtime/kubernetes", tag="v0.1.0")
  terraform_state=remote_repo(label="terraform_state", url="git@github.com:smintz/platform-engineering.git//drivers/state/terraform", tag="v0.1.0")
created: 2026-09-14
updated: 2026-09-14T00:20:00Z
---

# Debug: mod-tidy-subdir-redownload

## Symptoms

- **Expected:** With `protoconf.lock` present and `.protoconf_cache/` warm, `protoconf mod tidy` should
  perform NO network fetch at all on a second run.
- **Actual:** Only the modules whose URL contains a go-getter `//subdir` component re-clone on every
  `mod tidy` run. The root-pointing module (`platform`, no `//`) caches correctly and is not re-fetched.
- **Error messages:** None. Command succeeds; the problem is purely redundant downloads / slowness.
- **Timeline:** First time using the `//subdir` CONFIGSPACE shape — no known-good baseline for it.
  The root-only form has always cached fine.
- **Reproduction:** Run `protoconf mod tidy` twice with the CONFIGSPACE above. Observe that
  `kubernetes` and `terraform_state` re-download on the second run while `platform` does not.

## Current Focus

```yaml
hypothesis: "Validate() silently swallows any non-ErrNotExist dirhash.HashDir error while r.Integrity==\"dummy\" (the state right after a fresh clone, before the real hash is stamped), so a transient/structural hashing failure on the //subdir extraction path gets recorded as Integrity=\"\" instead of a real hash. \"\" is written to protoconf.lock, so every subsequent `mod tidy` sees ErrorRemoteRepoNoIntegrityInfo and redownloads — forever, silently."
test: "Fixed and added regression tests in compiler/lib/module_service_test.go; verified full test suite + walk/order/redundant-download/validate-swallow assertions pass."
expecting: "n/a - hypothesis confirmed, fix applied"
next_action: "run gofmt/go vet, then request human verification"
reasoning_checkpoint:
  hypothesis: "Three confirmed defects in compiler/lib/module_service.go jointly produce silent, redundant module re-downloads for //subdir deps: (1) walk() sorts `keys` but never reorders `deps` to match, so traversal order is actually random map order, not the sorted order the code implies; (2) Sync()'s per-node walkFn calls m.DownloadDeps(ctx, r) for every node INCLUDING the sentinel head, and DownloadDeps(head) redundantly re-walks and re-Downloads/re-Validates every top-level dependency a second time in the same Sync() pass (proven via debug log: each dep's \"downloading repo\" debug line appears twice per Sync()); (3) Validate() swallows any dirhash.HashDir error other than os.ErrNotExist whenever r.Integrity==\"dummy\", returning (h=\"\", nil) instead of propagating the error — so Download() stamps r.Integrity=\"\" as if it were a successful hash, and that empty Integrity gets persisted to protoconf.lock, forcing ErrorRemoteRepoNoIntegrityInfo (and therefore a full redownload) on every subsequent invocation with zero visible error. (3) is the structural bug that converts ANY dirhash-walk failure on the //subdir-extracted directory into a permanent, silent, per-module cache miss; (2) doubles exposure to it every run; (1) makes the whole thing non-deterministic to debug."
  confirming_evidence:
    - "Direct code read of walk() (module_service.go:559-572): sort.Strings(keys) mutates keys in place; deps is never reindexed, so `for i := range keys { walk(deps[i], walkFn) }` walks deps in original (random map-iteration) order, not sorted order."
    - "Debug-log evidence from a full local git repro (compiler/lib, TestReproSubdirRedownloadFull, git::file:// backend): each of platform/kubernetes/terraform_state logs 'downloading repo label=X integrity=...' TWICE within a single Sync() call — once from the direct Walk() visit, once from DownloadDeps(head)'s redundant re-walk of head.GetDeps()."
    - "Direct code read of Validate() (module_service.go:274-290): `if r.Integrity != \"dummy\" && r.Integrity != h` short-circuits to false whenever Integrity==\"dummy\", so a non-nil, non-ErrNotExist `err` from dirhash.HashDir is discarded and (h, nil) is returned regardless of h's value or err."
    - "Reproduced the swallow directly: forced a dirhash.HashDir error on a real downloaded directory (TestReproPlatformDirhashError/2) and confirmed Validate()'s only checked error class is os.ErrNotExist; any other class (e.g. 'X is not a directory', or a walk failure from an unreadable/symlinked entry) falls through to the swallow branch when Integrity==\"dummy\"."
    - "Identified the concrete mechanism that can produce such a HashDir error specifically (and only) on the //subdir path: hashicorp/go-getter's GitGetter.Get() unconditionally calls fetchSubmodules() at the end of every clone/update (get_git.go:144), and fetchSubmodules() unconditionally sets `g.client.DisableSymlinks = true` on the shared *Client (get_git.go:329) even when the repo has no submodules. For //subdir deps, Client.Get() clones the full repo into a temp dir via that same GitGetter, THEN calls copyDir(..., c.DisableSymlinks, ...) to extract just the subdir (source.go:344) — by which point DisableSymlinks has already been flipped true by the clone that just ran. Root (non-subdir) deps never reach that copyDir call at all (source.go: subDir==\"\" returns right after the clone), so they are structurally immune. Reproduced directly: adding one symlink under the subdir being extracted causes copyDir to fail (confirmed via TestReproSubdirSymlinkBreaksIntegrity)."
  falsification_test: "If Validate() is changed to propagate any non-ErrNotExist dirhash error, a genuinely broken/unhashable //subdir extraction will surface as a loud `mod tidy` failure instead of a silent empty-Integrity write — disproving the hypothesis would require a `mod tidy` run where the lock file's Integrity for a subdir module is empty/keeps getting reset AND no error is ever printed even after this fix; that has not been observed against the reproduction fixtures used here."
  fix_rationale: "The fix targets the exact mechanism, not the trigger: Validate() must fail loudly on any real hashing error instead of downgrading it to a fake success. Removing DownloadDeps(head)'s redundant re-walk removes wasted duplicate work and doubled exposure to any transient error on every single run. Fixing walk()'s keys/deps misalignment makes traversal (and therefore lock-file write order / debug output) deterministic. None of these are workarounds around the symptom (e.g. skipping re-validation) — they close the exact code paths that allow a hashing failure to be silently absorbed and repeated forever."
  blind_spots: "Could not test against the user's actual private repo/SSH transport (no network access, no repro repo path per the debug trigger). The exact HashDir-breaking artifact in kubernetes/terraform_state's real subdirectories (symlink, permission bit, or something else) is unconfirmed — only a plausible, structurally-forced mechanism (go-getter's fetchSubmodules DisableSymlinks mutation) was reproduced locally. That specific mechanism manifests as a LOUD error in this repro (ErrSymlinkCopy), which is a partial mismatch against the user's 'no error messages' report; the Validate()-swallow fix is the piece that converts the class of bug (any HashDir failure) into the reported silent-loop symptom for whichever artifact is actually present in their repo, and is safe/correct regardless of which specific artifact it is."
  candidate_causes:
    - "code: Validate() error-swallowing when Integrity==\"dummy\" (module_service.go Validate())"
    - "code: Sync()'s redundant DownloadDeps(head) re-walk of already-visited top-level deps (module_service.go Sync()/DownloadDeps())"
    - "code: walk() sorts keys but not the parallel deps slice, making traversal non-deterministic (module_service.go walk())"
    - "dependency: hashicorp/go-getter's GitGetter.fetchSubmodules() unconditionally mutates the shared Client.DisableSymlinks to true after every clone, corrupting the immediately-following //subdir copyDir call — a go-getter behavior, not something protoconf's own logic controls beyond not being surprised by it"
  and_gate: "yes — the full silent-loop symptom needs (a) some dirhash.HashDir failure specific to the //subdir-extracted directory content (data/dependency category — e.g. a symlink under the subdir, forced non-copyable by go-getter's post-clone DisableSymlinks mutation) AND (b) Validate()'s error-swallowing (code category) turning that failure into a silent fake-success instead of a loud error. Neither alone reproduces 'always redownloads with zero errors, only for subdir modules, forever': the HashDir failure alone would surface loudly (as reproduced), and Validate()'s swallow bug alone is inert without an actual HashDir failure to swallow."
tdd_checkpoint: null
```

## Evidence

- timestamp: 2026-09-14T00:00:00Z
  checked: compiler/lib/module_service.go Init()/Add()/repoLabel() GetterUrl + Integrity round-trip across two simulated `mod tidy` invocations (TestReproSubdirRedownload, offline, no network)
  found: GetterUrl is byte-identical across two Init() calls for both root and //subdir URLs (query encoding is alphabetically deterministic via url.Values.Encode()); Integrity correctly survives Lock()->LoadFromLockFile()->Init() round-trip for all three deps including subdir ones.
  implication: The Init()-time "originalGetterUrl != msg.GetterUrl -> reset Integrity" merge logic is NOT the source of the bug. Eliminated.

- timestamp: 2026-09-14T00:05:00Z
  checked: Full mod-tidy pipeline (LoadFromLockFile+Init+Sync) run twice against a local git repo with git::file:// URLs (forced "git" getter, matching how SSH URLs get force-prefixed), TestReproSubdirRedownloadFull
  found: With a clean fixture (no symlinks, no unusual permissions), all three deps — including both //subdir ones — cache correctly on the second run (cache dirs untouched, Integrity hashes match). However, debug logging revealed each dependency's Download()/Validate() gets invoked TWICE within a SINGLE Sync() call (once via Walk()'s direct per-node visit, once via the sentinel head node's redundant DownloadDeps(head) re-walking head.GetDeps()).
  implication: Confirmed a real redundant-work bug (Sync() double-processes every top-level dep every run) that is harmless when hashing succeeds but doubles exposure to any transient/structural hashing failure. Also confirmed walk()'s traversal order is NOT alphabetically sorted despite sort.Strings(keys) — keys is sorted but the parallel `deps` slice is not reindexed, so real traversal order is whatever Go's randomized map iteration produced.

- timestamp: 2026-09-14T00:10:00Z
  checked: dirhash.HashDir behavior via a directly-called go-getter file:// download (TestReproPlatformDirhashError/2), and Validate()'s handling of a non-ErrNotExist HashDir error
  found: Validate() only special-cases os.ErrNotExist; any OTHER dirhash.HashDir error is discarded once r.Integrity=="dummy" (the state Download() sets immediately before calling getter.GetAny), because `r.Integrity != "dummy" && r.Integrity != h` short-circuits to false. Validate() then returns (h="", nil) — Download() stamps r.Integrity="" and reports success.
  implication: This is the mechanism that converts a real (but code-invisible) hashing failure into a silent "success" that persists Integrity="" to protoconf.lock, forcing ErrorRemoteRepoNoIntegrityInfo (full redownload) on every subsequent mod tidy call, with no error ever surfaced. This is the confirmed structural root cause of "silent, permanent, per-module redownload."

- timestamp: 2026-09-14T00:15:00Z
  checked: hashicorp/go-getter v1.8.9 source (get_git.go, source.go, copy_dir.go) for a mechanism that is structurally specific to //subdir extraction and immune for root (non-subdir) URLs
  found: GitGetter.Get() unconditionally calls fetchSubmodules() at the end of every clone/update (get_git.go:144), which unconditionally sets `g.client.DisableSymlinks = true` on the shared *Client (get_git.go:329) regardless of whether the repo actually has submodules. For //subdir deps, Client.Get() clones the FULL repo into a temp dir via that GitGetter, then immediately calls copyDir(realDst, subDir, ..., c.DisableSymlinks, ...) to extract just the subdir (source.go:344) — by which point DisableSymlinks has already flipped true. Root deps return right after the direct clone and never reach that copyDir call, so they are structurally immune to this. Confirmed directly: adding one symlink under the extracted //subdir causes copyDir to hard-fail with "copying of symlinks has been disabled" (TestReproSubdirSymlinkBreaksIntegrity).
  implication: This is a plausible concrete trigger for a dirhash-breaking condition that is exclusive to //subdir modules and never affects root modules — matching the reported asymmetry structurally. It manifests as a loud error in this local repro (partial mismatch vs. "no error messages" in the report), but combined with the Validate()-swallow bug, ANY HashDir failure specific to the subdir content (this or another artifact actually present in the user's repo) becomes the reported silent, permanent redownload loop once Validate() is not fixed.

## Eliminated

- hypothesis: "Init()'s CONFIGSPACE re-merge resets Integrity to \"\" for //subdir deps because the freshly-computed GetterUrl differs from the one recorded in protoconf.lock on a prior run."
  evidence: "TestReproSubdirRedownload showed byte-identical GetterUrl across two independent Init() calls for both root and subdir URLs; Integrity survived the full Lock/LoadFromLockFile/Init round trip unchanged for all three deps."
  timestamp: 2026-09-14T00:00:00Z

- hypothesis: "go-getter's //subdir extraction (temp-clone + copyDir) produces non-deterministic file content across separate downloads, causing dirhash mismatches on later validation."
  evidence: "TestReproSubdirRedownloadFull (clean fixture, no symlinks/unusual permissions) showed byte-identical, correctly-cached content and matching Integrity hashes across two full mod-tidy runs for both root and //subdir deps."
  timestamp: 2026-09-14T00:05:00Z

## Resolution

```yaml
root_cause: >
  Validate() in compiler/lib/module_service.go silently discards any
  dirhash.HashDir error other than os.ErrNotExist whenever r.Integrity=="dummy"
  (the state Download() sets right before the actual clone), returning
  (h="", nil) instead of propagating the error. Download() then stamps
  r.Integrity="" as if hashing had succeeded, and that empty Integrity is
  persisted to protoconf.lock -- so every subsequent `mod tidy` sees
  ErrorRemoteRepoNoIntegrityInfo and silently redownloads, forever, with no
  error ever surfaced.
  This is compounded by two related defects in the same file: (1) Sync()
  calls m.DownloadDeps(ctx, r) for every node Walk() visits, including the
  sentinel head, so DownloadDeps(head) redundantly re-Downloads/re-Validates
  every top-level dependency a SECOND time within the same Sync() call,
  doubling exposure to any transient/structural hashing failure on every
  single run; (2) walk() sorts the `keys` slice but never reorders the
  parallel `deps` slice to match, so traversal order is actually random
  map-iteration order rather than the sorted order the code implies, making
  the download/validate order (and therefore any related failure) harder to
  reproduce deterministically.
  A concrete, root-immune trigger for the underlying HashDir failure was
  identified in hashicorp/go-getter v1.8.9: GitGetter.Get() unconditionally
  calls fetchSubmodules() after every clone/update, which unconditionally
  sets the shared Client.DisableSymlinks=true even when there are no
  submodules; for //subdir deps this happens BEFORE the immediately-following
  copyDir() extraction step that only //subdir deps go through (root deps
  never reach that step), so any symlink under the extracted subdir now
  fails to copy/hash where a root clone of the same repo would not be
  affected.
fix: >
  1. Validate(): propagate any dirhash.HashDir error other than os.ErrNotExist
     as ErrorRemoteRepoValidationFailed instead of silently returning
     (h="", nil) -- a real hashing failure must surface loudly, never be
     absorbed as a fake successful integrity of "".
  2. DownloadDeps(): return immediately for the sentinel head node (r.Url=="."),
     matching the existing sentinel convention used elsewhere in this file.
     Walk() already recurses into every real dependency directly; head has no
     GetterUrl and therefore no submodule lock file of its own to merge, so
     DownloadDeps(head) was pure redundant re-work.
  3. walk(): collect (key, dep) pairs and sort by key together, instead of
     sorting the `keys` slice independently of the parallel `deps` slice, so
     traversal order is actually the deterministic sorted order the code
     always intended.
verification: |
  guardrail_verdict: accepted
  signals:
    - signal: build
      result: pass
      detail: "go build ./... clean"
    - signal: format/vet
      result: pass
      detail: "gofmt -l clean on module_service.go and module_service_test.go; go vet ./compiler/... ./mod/... clean"
    - signal: full existing test suite
      result: pass
      detail: "go test ./compiler/lib/... ./mod/... green, including TestModuleService_Sync's 5 subtests (empty/no-integrity/bad-integrity/good-integrity/download-deps, the last of which does a real network mod-sync)"
    - signal: new regression tests (added)
      result: pass
      detail: "TestWalkVisitsInSortedKeyOrder, TestValidatePropagatesHashDirErrors (4 subtests incl. 2 boundary-neighbor cases), TestDownloadDepsSkipsSentinelHead — all green against the fixed code"
    - signal: revert-and-confirm-bug-returns (mutation-style guardrail)
      result: pass
      detail: "git stash of module_service.go (fix removed) + rerun of the 3 new regression tests: all 3 failed exactly as expected (TestWalkVisitsInSortedKeyOrder: wrong traversal order observed; TestValidatePropagatesHashDirErrors: swallowed-error subtest got nil instead of an error; TestDownloadDepsSkipsSentinelHead: sentinel head attempted a real Download() and errored on the bogus scheme). Fix re-applied and all 3 pass again."
files_changed:
  - compiler/lib/module_service.go
  - compiler/lib/module_service_test.go
```
