"use strict";
/**
 * Verification Status — single queryable home for verification-status routing.
 *
 * Issue #651: consolidate the pass/gaps_found/human_needed routing that was
 * previously scattered across ship.md and execute-phase.md into a single
 * tested module. Both workflow files will later consume this module's routing
 * table as the single source of truth.
 *
 * ADR-457 build-at-publish: source in src/verification.cts, compiled to
 * gsd-core/bin/lib/verification.cjs (gitignored).
 *
 * DEFECT.FRONTMATTER-SCALAR-BROAD-GREP fix: status extraction is scoped to
 * the leading YAML frontmatter block only. A `status:` line in the body (e.g.
 * inside a fenced code block) is ignored — this is the exact failure mode that
 * issue #586 / PR #650 identified. The shared extractFrontmatter parser anchors
 * its regex at byte 0 of the document, which provides this guarantee.
 *
 * #2348 staleness signal: whether a *-VERIFICATION.md is stale (a summary newer
 * than it) is decided from git commit time when a file is committed AND clean,
 * and from filesystem mtime otherwise. mtimes are assigned at checkout time and
 * are not preserved by `git clone` / `cp -R`, and any unrelated `touch` /
 * reformat / editor-save re-stales a valid report — so a committed phase could
 * read `passed` on one machine and `stale` on a fresh clone purely from checkout
 * order. Git commit time is content-tied and clone-stable; mtime is retained
 * only for uncommitted or working-tree-dirty files, where it is the true
 * last-changed signal. Both are real wall-clock change times, so the comparison
 * is sound even when one file uses each.
 */
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
const node_fs_1 = __importDefault(require("node:fs"));
const node_path_1 = __importDefault(require("node:path"));
const node_crypto_1 = __importDefault(require("node:crypto"));
const project_root_cjs_1 = require("./project-root.cjs");
// eslint-disable-next-line @typescript-eslint/no-require-imports -- io.cjs is an export= CommonJS module
const io = require("./io.cjs");
// eslint-disable-next-line @typescript-eslint/no-require-imports -- phase-id.cjs is an export= CommonJS module
const phaseId = require("./phase-id.cjs");
// eslint-disable-next-line @typescript-eslint/no-require-imports -- frontmatter.cjs is an export= CommonJS module
const frontmatterMod = require("./frontmatter.cjs");
// eslint-disable-next-line @typescript-eslint/no-require-imports -- plan-scan.cjs is an export= CommonJS module
const scanPhasePlans = require("./plan-scan.cjs");
// eslint-disable-next-line @typescript-eslint/no-require-imports -- core-utils.cjs is an export= CommonJS module
const coreUtilsMod = require("./core-utils.cjs");
// eslint-disable-next-line @typescript-eslint/no-require-imports -- planning-scope.cjs is an export= CommonJS module
const planningScopeMod = require("./planning-scope.cjs");
const shell_command_projection_cjs_1 = require("./shell-command-projection.cjs");
const runtime_slash_cjs_1 = require("./runtime-slash.cjs");
const { output, error } = io;
const { extractPhaseToken, scopeToPhase } = phaseId;
const { extractFrontmatter } = frontmatterMod;
const { normalizeLineEndings } = coreUtilsMod;
const { SCOPE } = planningScopeMod;
// ─── Constants ────────────────────────────────────────────────────────────────
/** The set of status values that the gsd-verifier agent emits. */
const VERIFIER_STATUSES = ['passed', 'gaps_found', 'human_needed'];
/**
 * Canonical routing table for verification statuses.
 *
 * This is the single source of truth — ship.md and execute-phase.md will
 * later import from here instead of embedding their own message strings.
 *
 * INTERNAL SENTINELS: 'missing' and 'unknown' are operational states constructed
 * internally — the verifier (gsd-verifier.md) never emits them. The verifier only
 * emits values in VERIFIER_STATUSES (passed|gaps_found|human_needed). The guard in
 * readVerificationStatus excludes 'missing' and 'unknown' from raw-status table
 * lookup so they can only be reached via internal construction paths.
 *
 * For 'gaps_found', next_command is built at call time in readVerificationStatus
 * by substituting the phase number — it is NOT stored as a function in the table.
 *
 * #2617: `next_command` here holds a BARE command name (`execute-phase`), never a
 * prefixed one. Every return path projects it through `formatGsdSlash` with the
 * caller's runtime, so Codex sees `$gsd-execute-phase` and slash-hyphen runtimes
 * see `/gsd-execute-phase`. Storing a prefixed literal is what leaked the
 * hard-coded (and deprecated) `/gsd:` colon form to every runtime.
 */
const VERIFICATION_ROUTING_TABLE = {
    passed: {
        status: 'passed',
        next_action: 'Verification passed — continue.',
        next_command: '',
    },
    gaps_found: {
        status: 'gaps_found',
        next_action: 'Gaps found. Plan the fixes, then re-run execute-phase before shipping.',
        // next_command is computed at call time; this entry is never returned directly.
        next_command: '',
    },
    human_needed: {
        status: 'human_needed',
        next_action: "Human verification required. Complete the manual tests in the phase's *-UAT.md, then re-run the verify step until status is passed.",
        // #2617: was '' — next_action told the user to "re-run the verify step" but
        // named no command, while init.cts's parallel projector emitted
        // `verify-work <N>` for this same state. The two surfaces disagreed on
        // whether a next command existed at all; init's answer was the useful one,
        // and init now delegates here rather than re-deriving it.
        next_command: 'verify-work',
    },
    stale: {
        status: 'stale',
        next_action: 'Verification is stale. Re-run verify-work before transition.',
        next_command: '',
    },
    // INTERNAL SENTINEL: constructed when no *-VERIFICATION.md file exists or when
    // the file has no parseable frontmatter status. Never emitted by the verifier.
    missing: {
        status: 'missing',
        next_action: 'No verification report found — the verify step never completed. Running execute-phase is safe here: it resumes at the verification gates and does not re-run plans that already have a SUMMARY.md (see #2868).',
        next_command: 'execute-phase',
    },
    // INTERNAL SENTINEL: constructed when the file has a status value not in
    // VERIFIER_STATUSES. Never emitted by the verifier.
    unknown: {
        status: 'unknown',
        next_action: '', // filled in dynamically with the raw value
        next_command: 'execute-phase',
    },
};
/**
 * Project a BARE command name (plus optional argument tail) into the surface the
 * given runtime actually installs (#2617).
 *
 * `formatGsdSlash` owns the per-runtime shape (`$gsd-<cmd>` for shell-var
 * runtimes like Codex, `/gsd-<cmd>` otherwise) and is idempotent, so passing an
 * already-prefixed string is safe. An empty command stays empty — "no next
 * command" must not become a bare prefix.
 */
function projectNextCommand(bare, runtime, tail = '') {
    if (!bare)
        return '';
    return `${(0, runtime_slash_cjs_1.formatGsdSlash)(bare, runtime)}${tail}`;
}
/**
 * Real `node:fs`-backed default satisfying FsLike. Every method wraps a call
 * to `fs.<method>` rather than capturing the function reference — existing
 * tests mock individual `fs` methods in place (`t.mock.method(fs, 'statSync', …)`),
 * and a captured reference taken at module-load time would be invisible to
 * that late mock, silently un-mocking this seam's "default" path.
 */
const defaultFsImpl = {
    readdirSync: (dir) => node_fs_1.default.readdirSync(dir),
    readFileSync: (filePath, encoding) => node_fs_1.default.readFileSync(filePath, encoding),
    statSync: (filePath) => node_fs_1.default.statSync(filePath),
};
/** Normalize separators to posix (git emits `/`; callers may pass `\` on Windows). */
function toPosix(p) {
    return p.replace(/\\/g, '/');
}
/**
 * #4155: canonicalize a covered-input path before it becomes either a
 * dedup/sort/hash key or a confinement-check subject. `path.posix.normalize`
 * collapses `./`, redundant slashes, and internal `..` segments (`a/../../b`
 * → `../b`) — without this, two spellings of the SAME file (`src/x.cts` vs
 * `./src/x.cts`) hash as different covered inputs (spurious `stale`, or a
 * file double-counted into the digest under two keys), and an escape
 * disguised by an internal `..` segment slips past a check that only looks
 * at the string's start.
 */
function normalizeRel(p) {
    return node_path_1.default.posix.normalize(toPosix(p));
}
/** Canonicalize a covered-files list: normalize, de-duplicate, sort — the SAME
 *  transform computeCoveredDigest and cmdVerificationFingerprint both need
 *  (the digest's own key order; the CLI's own `covered_files` JSON output). */
function canonicalizeCoveredFiles(files) {
    return Array.from(new Set(files.map(normalizeRel))).sort();
}
// ─── #4155: covered-input fingerprint ──────────────────────────────────────────
/**
 * Bump on any change to the digest's input shape (path list, hashing order,
 * per-file hash algorithm) so an old stored digest can never collide with a
 * differently-computed new one — a version mismatch is just a mismatch.
 */
const FINGERPRINT_VERSION = 1;
/**
 * #4155: recompute the deterministic content fingerprint over a verifier's
 * declared covered-input set (phase PLAN/SUMMARY, mapped requirements,
 * implementation files in the change set) and return the versioned digest
 * string, or `null` if the set cannot be resolved.
 *
 * Determinism: paths are de-duplicated and SORTED before hashing (directory
 * enumeration order is irrelevant), each path is resolved relative to
 * `projectRoot` (the absolute checkout path never enters the digest), and
 * file BYTES are hashed (mtime never enters the digest).
 *
 * NOT normalized: line endings. Unlike the report-frontmatter read (which
 * runs every VERIFICATION.md through `normalizeLineEndings`), covered-file
 * bytes are hashed exactly as they sit on disk. A covered text file checked
 * out with CRLF line endings (e.g. a Windows checkout without a `.gitattributes
 * eol=lf` rule pinning it to LF) hashes differently than the same file on an
 * LF checkout — a real cross-platform digest mismatch, not a bug, since GSD
 * installs into arbitrary user projects with no guaranteed line-ending policy.
 *

 * Fail closed: a covered path that is empty, absolute, escapes
 * `projectRoot` (`..` traversal), or cannot be read (missing, unreadable,
 * not a regular file) makes the WHOLE fingerprint unresolvable — returns
 * `null` — rather than silently hashing a partial set. Callers treat `null`
 * as stale (#4155), the same fail-closed shape #3057 B3 established for the
 * legacy mtime staleness check.
 *
 * Always reads through the REAL `node:fs`, never a caller-injected `FsLike`
 * seam — same reasoning as the root canonicalization below, extended to
 * every covered file: `covered_files` is expected to span the whole
 * `projectRoot` (implementation files under `src/`, not just `.planning/`
 * artifacts), so a caller-scoped containment wrapper narrower than
 * `projectRoot` (e.g. `planning-inspect.cts`'s `containmentEnforcingVerificationFs`,
 * confined to `.planning/`) would reject every implementation-file read and
 * report EVERY fingerprinted phase permanently `stale` regardless of actual
 * drift — the bug this comment now documents against regressing. The
 * `realRel`-vs-`realRoot` re-check a few lines below already does the real
 * confinement work (against `projectRoot`, the correct boundary for this
 * data), so no security property is lost by bypassing a narrower seam here.
 */
function computeCoveredDigest(projectRoot, coveredFiles) {
    const uniqueSorted = canonicalizeCoveredFiles(coveredFiles);
    if (uniqueSorted.length === 0)
        return null;
    // Canonicalize the root ONCE — every candidate's realpath is checked against
    // this, not the possibly-symlinked `projectRoot` argument itself. Always via
    // the REAL fs, never fsImpl: `projectRoot` is a trusted anchor the CALLER
    // derived (findProjectRoot), not attacker-influenced covered-input data —
    // routing it through a caller-scoped containment seam (e.g. #4155's
    // containmentEnforcingVerificationFs, confined to `.planning/`, a proper
    // SUBSET of `projectRoot`) would reject the root itself and fail every
    // lookup regardless of whether the covered files are legitimate.
    let realRoot;
    try {
        realRoot = node_fs_1.default.realpathSync(projectRoot);
    }
    catch {
        return null;
    }
    const parts = [];
    for (const rel of uniqueSorted) {
        // `normalizeRel` (already applied by `canonicalizeCoveredFiles` above)
        // collapses internal `..` segments before `rel` ever reaches here
        // (`a/../../b` → `../b`), so this start-of-string check is already the
        // full lexical confinement test — no separate post-`path.resolve`
        // re-check can observe a different answer.
        if (rel === '' || rel === '..' || rel.startsWith('../') || node_path_1.default.isAbsolute(rel))
            return null;
        const resolved = node_path_1.default.resolve(projectRoot, rel);
        let bytes;
        try {
            // A regular file INSIDE projectRoot can still be a symlink whose TARGET
            // escapes it — statSync/readFileSync follow symlinks, so the lexical
            // confinement check above is not enough. realpathSync resolves the
            // actual target; re-confining against realRoot closes that gap.
            const real = node_fs_1.default.realpathSync(resolved);
            const realRel = node_path_1.default.relative(realRoot, real);
            if (realRel === '' || realRel === '..' || realRel.startsWith(`..${node_path_1.default.sep}`) || node_path_1.default.isAbsolute(realRel)) {
                return null;
            }
            const st = node_fs_1.default.statSync(real);
            if (!st.isFile())
                return null;
            bytes = node_fs_1.default.readFileSync(real);
        }
        catch {
            return null;
        }
        const fileHash = node_crypto_1.default.createHash('sha256').update(bytes).digest('hex');
        parts.push(`${rel}\n${fileHash}\n`);
    }
    const aggregate = node_crypto_1.default
        .createHash('sha256')
        .update(`v${FINGERPRINT_VERSION}\n${parts.join('')}`, 'utf-8')
        .digest('hex');
    return `v${FINGERPRINT_VERSION}:sha256:${aggregate}`;
}
/**
 * #4155: the content fingerprint only recomputes digests for paths the
 * verifier actually DECLARED in `covered_files` — it has no way to notice a
 * plan or summary added to the phase directory AFTER verification if that
 * new file was never declared. This closes that gap the same way the
 * legacy mtime check always did: by re-scanning the LIVE directory (not the
 * declared list) for every current `*-PLAN.md`/`*-SUMMARY.md` and checking
 * each is represented in `coveredFiles` — matched by suffix (mirrors
 * `matchRequestedFile`'s convention) since `coveredFiles` holds
 * project-root-relative paths while the scan returns phase-relative
 * filenames. Returns `true` if every current plan/summary is covered,
 * `false` otherwise — callers only ever branch on this pass/fail, so no
 * caller needs which artifact was uncovered.
 *
 * Fails CLOSED on an incomplete scan: `scanPhasePlans` never throws on a
 * readdir failure — it reports it via `scope` (`SCOPE.UNREADABLE` for the
 * phase dir itself, `SCOPE.TRUNCATED` for an unreadable nested `plans/`)
 * with whatever files it DID manage to enumerate, per `SCOPE`'s own
 * contract (`planning-scope.cts`): zero items under a non-`COMPLETE` scope
 * is a NON-answer, never "this phase has no plans." Branching on `scope`
 * here (rather than a try/catch, which this scan never triggers) is what
 * makes an unreadable `plans/` dir report `false` instead of silently
 * treating its invisible contents as vacuously covered — the same
 * fail-open regression #3057 B3 fixed for the legacy path.
 */
function allCurrentArtifactsCovered(phaseDir, coveredFiles) {
    const scan = scanPhasePlans(phaseDir);
    if (scan.scope !== SCOPE.COMPLETE)
        return false;
    const coveredPosix = canonicalizeCoveredFiles(coveredFiles);
    return [...scan.allPlanFiles, ...scan.summaryFiles].every((artifact) => {
        const artifactPosix = toPosix(artifact);
        return coveredPosix.some((c) => c === artifactPosix || c.endsWith(`/${artifactPosix}`));
    });
}
/**
 * Match a git-emitted (repo-root-relative) path back to the caller's
 * phaseDir-relative request by exact match or `/`-bounded suffix — precise
 * enough that a root file and a nested `plans/` file can never collide (a plain
 * basename match could). Returns the original caller-form file string, or null.
 */
function matchRequestedFile(gitPath, requested, requestedPosix) {
    const g = toPosix(gitPath);
    for (let i = 0; i < requested.length; i++) {
        const want = requestedPosix[i];
        if (g === want || g.endsWith('/' + want))
            return requested[i];
    }
    return null;
}
/**
 * Parse `git log --format=%ct --name-only` output into file → most-recent commit
 * time (ms). Output is reverse-chronological, so a file's FIRST appearance
 * top-down is its latest commit. `%ct` headers are pure digits; path lines
 * contain a `.` (the `.md` extension) — so the two are unambiguous.
 */
function parseCommitTimes(stdout, requested, requestedPosix) {
    const out = new Map();
    let currentCt = null;
    for (const line of stdout.split('\n')) {
        if (line.length === 0)
            continue;
        if (/^\d+$/.test(line)) {
            currentCt = Number.parseInt(line, 10);
            continue;
        }
        if (currentCt === null)
            continue;
        const rel = matchRequestedFile(line, requested, requestedPosix);
        if (rel !== null && !out.has(rel))
            out.set(rel, currentCt * 1000);
    }
    return out;
}
function defaultPhaseCleanCommitTimesMs(phaseDir, files, execGitFn = shell_command_projection_cjs_1.execGit) {
    if (files.length === 0)
        return new Map();
    const requestedPosix = files.map(toPosix);
    const logRes = execGitFn(['log', '--first-parent', '--format=%ct', '--name-only', '--', ...files], {
        cwd: phaseDir,
    });
    if (logRes.error || logRes.exitCode !== 0 || logRes.stdout.length === 0)
        return new Map();
    const commitTimes = parseCommitTimes(logRes.stdout, files, requestedPosix);
    if (commitTimes.size === 0)
        return commitTimes;
    // Drop dirty files (working tree ≠ HEAD) so their mtime is used instead. If the
    // dirty-check itself is INCONCLUSIVE (git diff errored / non-zero — as opposed
    // to "ran and reported no dirty files"), we cannot prove any file is clean, so
    // fail SAFE: discard the commit times and let every file fall back to mtime,
    // the same direction as a git-log failure. Trusting possibly-stale commit times
    // here would silently mask a real edit (false "not stale"). (#2348)
    const diffRes = execGitFn(['diff', '--name-only', 'HEAD', '--', ...files], { cwd: phaseDir });
    if (diffRes.error || diffRes.exitCode !== 0)
        return new Map();
    for (const line of diffRes.stdout.split('\n')) {
        if (line.length === 0)
            continue;
        const rel = matchRequestedFile(line, files, requestedPosix);
        if (rel !== null)
            commitTimes.delete(rel);
    }
    return commitTimes;
}
/**
 * Build a 'missing' result from the routing table.
 * Used for two early-return paths: no *-VERIFICATION.md file found, and
 * file present but no parseable frontmatter status.
 */
function missingResult(runtime, phaseArg) {
    const route = VERIFICATION_ROUTING_TABLE['missing'];
    return {
        status: route.status,
        next_action: route.next_action,
        next_command: projectNextCommand(route.next_command, runtime, phaseArg),
    };
}
/**
 * #3518: the shared phase-pinned artifact-selection core BOTH single-pick
 * resolvers (`resolveVerificationFile` for `*-VERIFICATION.md`,
 * `resolveUatFile` for `*-UAT.md`) delegate to — one rule, not two grammars
 * that agree today and drift tomorrow (epic #3473 F2's defect class).
 *
 * `bareName` is the artifact filename WITHOUT the leading dash (`'UAT.md'`);
 * a "dashed" candidate is any entry ending `-${bareName}`.
 *
 * Selection order:
 *   1. `options.phaseToken` given and `<phaseToken>-${bareName}` is among
 *      the candidates — that exact file always wins: it is THIS phase's own
 *      artifact, and no other candidate (whichever phase's token it carries)
 *      can outrank it (#3492 / #3518).
 *   2. Fallback — no exact phase-token match (or no token given): alphabetically
 *      first of the dashed candidates that are THIS phase's own, per
 *      `scopeToPhase(candidates, options.phaseDirName)` (#3511 reconciliation,
 *      below). Load-bearing: a phase whose only artifact is non-canonically
 *      named must keep resolving to it, not to null — this fix must not turn
 *      "found an artifact" into "found nothing" for anyone. A
 *      non-canonically-named artifact of THIS phase (e.g.
 *      `03-CORRECTION-VERIFICATION.md` in `03-foo`) still passes
 *      `isPhaseArtifact` (it names phase 03, same as the directory), so it
 *      is still returned here.
 *   3. `options.allowBare` only — a bare `${bareName}`, ranked BELOW both
 *      of the above. Rationale: a dashed file names its phase, a bare one
 *      does not, so a dashed file (canonical or not) is always the better
 *      answer when both exist. Reached when neither (1) nor (2) found any
 *      candidate — including when (2)'s scoping filtered every dashed
 *      candidate out as belonging to some OTHER phase.
 *
 * #3511 RECONCILIATION with `isPhaseArtifact` (`src/phase-id.cts`): that
 * predicate's own docblock used to flag this fallback as an open gap — its
 * aggregate scans exclude a cross-phase stray, but this single-pick resolver
 * did not, so it could return a stray as THE artifact while the aggregate
 * scans correctly ignored it. Closed by scoping step (2) above through
 * `scopeToPhase` (`src/phase-id.cts`, itself built on `isPhaseArtifact`):
 * `options.phaseDirName` threads the phase directory's basename in, and the
 * fallback now filters candidates through `scopeToPhase(candidates,
 * phaseDirName)` before picking alphabetically-first. This does NOT reopen
 * the #3357 guarantee — that guarantee is "a phase whose only report is
 * non-canonically named must keep working", and a non-canonically-named
 * artifact of THIS phase still passes `isPhaseArtifact` (it is membership by
 * phase number, not by canonical shape), so it is still returned. Only a
 * file belonging to a DIFFERENT phase is now excluded — and excluding it is
 * correct: returning another phase's artifact as this phase's own is worse
 * than reporting none (confidently wrong beats honestly empty).
 * The fail-safe now lives entirely inside `isPhaseArtifact`, not in
 * `scopeToPhase` (which is a plain filter with no unfiltered fallback):
 * (a) when phase-number membership cannot be determined for `phaseDirName` at
 * all (no reliable token — the zero-token directory case), every candidate is
 * treated as belonging to the phase; (b) the `firstLetterPrefixed`
 * bracket-ambiguity case, where a letter-prefixed-decimal dir is
 * string-indistinguishable from a bracket-dir token, also includes
 * everything rather than guess; (c) a token-less filename (bare
 * `${bareName}`) is accepted by directory containment alone. Outside
 * those cases, when scoping DOES remove every dashed candidate — a real
 * cross-phase stray, or a phase whose own artifact is genuinely absent — the
 * fallback below correctly falls through to `allowBare`/`null`: reporting no
 * artifact, not another phase's. `options.phaseDirName` omitted entirely skips
 * the filter outright (the ternary below), which is unscoped, pre-#3511
 * behavior.
 *
 * Pure — takes an already-read directory listing and does no I/O of its own,
 * so every call site keeps its existing `fsImpl` seam and no-throw contract
 * untouched.
 */
function resolvePhaseArtifactFile(entries, bareName, options = {}) {
    const candidates = entries.filter((f) => f.endsWith(`-${bareName}`)).sort();
    if (candidates.length > 0) {
        if (options.phaseToken) {
            const thisPhaseFile = `${options.phaseToken}-${bareName}`;
            if (candidates.includes(thisPhaseFile))
                return thisPhaseFile;
        }
        // #3511: scope the fallback to files that belong to THIS phase, so a
        // stray cross-phase file can no longer outrank a return of null.
        // `phaseDirName` omitted, or membership undeterminable for it, →
        // unscoped `candidates` (pre-#3511 behavior); otherwise strays are
        // filtered out, and if that leaves nothing the code falls through to
        // `allowBare`/`null` deliberately.
        const scoped = options.phaseDirName
            ? scopeToPhase(candidates, options.phaseDirName)
            : candidates;
        if (scoped.length > 0)
            return scoped[0];
    }
    if (options.allowBare && entries.includes(bareName))
        return bareName;
    return null;
}
/**
 * Resolve which `*-VERIFICATION.md` entry in a phase directory's listing IS
 * the phase's verification report, when more than one such file exists.
 *
 * #3357: a phase dir can legitimately hold more than one `*-VERIFICATION.md`
 * — the real per-phase report (`03-VERIFICATION.md`) alongside an ad-hoc plan
 * worksheet (`03-CORRECTION-VERIFICATION.md`). Picking "alphabetically first"
 * (`'C' < 'V'`) silently chose the worksheet, which usually has no
 * frontmatter `status:`, so a phase with a PASSING report read as `missing`.
 * This was two independent hand-rolled `.sort()[0]` picks
 * (findStaleVerificationSummary and readVerificationStatus) — this is the
 * single resolver both now call (#3473 F2).
 *
 * Selection order: see `resolvePhaseArtifactFile` (the shared core this
 * delegates to since #3518, itself phase-scoped since #3511) —
 * phase-token-pinned, then phase-scoped alphabetically-first dashed
 * fallback, then (allowBare only) a bare `VERIFICATION.md`. #3518 extracted
 * this into the shared core without changing behavior; #3511's
 * `phaseDirName` scoping now lives inside that shared core rather than here.
 */
function resolveVerificationFile(entries, options = {}) {
    return resolvePhaseArtifactFile(entries, 'VERIFICATION.md', options);
}
/**
 * #3518: resolve which `*-UAT.md` entry in a phase directory's listing IS
 * the phase's UAT artifact, when more than one such file exists — the UAT
 * counterpart of `resolveVerificationFile`, sharing its exact selection rule
 * via `resolvePhaseArtifactFile`.
 *
 * The bug this closes: both `uat_path` projectors in `src/init.cts` picked
 * with a bare `.find((f) => f.endsWith('-UAT.md') || f === 'UAT.md')` over an
 * unsorted `readdir` listing — no phase-membership check and no ordering — so
 * a stray or cross-phase `04-UAT.md` sitting in phase 03's directory could
 * become phase 03's `uat_path`, and WHICH file won was filesystem-dependent
 * (creation order on APFS, hash order on ext4/XFS): two machines on the same
 * commit could emit different `uat_path` values for the same phase. `uat_path`
 * is consumed downstream by workflows that then read the named file, so a
 * wrong path routes UAT state from another phase.
 *
 * Deterministic by construction: same answer on every machine. Phase-scoped
 * (#3511): passing `options.phaseDirName` filters the alphabetically-first
 * fallback (tier 2) to artifacts that belong to THIS phase — see
 * `resolvePhaseArtifactFile` for the full selection order and scoping
 * rationale.
 */
function resolveUatFile(entries, options = {}) {
    return resolvePhaseArtifactFile(entries, 'UAT.md', options);
}
function findStaleVerificationSummary(phaseDir, fsImpl = defaultFsImpl, phaseCleanCommitTimesMs = defaultPhaseCleanCommitTimesMs) {
    // FS errors (TOCTOU: a SUMMARY listed by scanPhasePlans then removed before statSync;
    // unreadable dir; broken symlink; file->dir swap) must degrade rather than throw
    // uncaught into callers that are NOT under the planning lock (init.manager /
    // init.progress / uat-predicate). Mirrors readVerificationStatus's no-throw
    // contract; `fsImpl` threads the same injectable-fs seam for parity/testing.
    // (Review B1 on #1548.) The degraded result is `{determined:false}`, NOT the
    // same value as a completed "nothing is stale" check — see StaleCheckResult
    // doc and #3057 B3. The caller decides how to route an indeterminate result;
    // this function only reports what it actually knows.
    try {
        const phaseFiles = fsImpl.readdirSync(phaseDir);
        // #3492: pin selection to THIS phase's own token so a stray cross-phase
        // or sentinel-numbered canonically-shaped file cannot outrank this
        // phase's own (possibly non-canonical) report. #3511: phaseDirName scopes
        // the fallback path to this same phase (see resolveVerificationFile docs).
        const phaseDirName = node_path_1.default.basename(phaseDir);
        const phaseToken = extractPhaseToken(phaseDirName);
        const verificationFile = resolveVerificationFile(phaseFiles, { phaseToken, phaseDirName });
        if (!verificationFile)
            return { determined: true, stale: false };
        const summaryFiles = scanPhasePlans(phaseDir).summaryFiles
            .slice()
            .sort();
        // No summary can be newer than the verification → never stale. Return before
        // touching git so a phase with no summaries costs zero subprocesses. (#2348)
        if (summaryFiles.length === 0)
            return { determined: true, stale: false };
        // Each file's effective "last changed" time = its commit time when committed
        // AND clean (content-tied and clone-stable), else its filesystem mtime (the
        // uncommitted working-tree edit). Both are real wall-clock change times, so
        // comparing a clean file's commit time against a dirty file's mtime is sound.
        // One resolver call = two git subprocesses for the whole phase. (#2348)
        const cleanCommitMs = phaseCleanCommitTimesMs(phaseDir, [verificationFile, ...summaryFiles]);
        const effectiveTimeMs = (file) => cleanCommitMs.has(file)
            ? cleanCommitMs.get(file)
            : fsImpl.statSync(node_path_1.default.join(phaseDir, file)).mtimeMs;
        const verificationTimeMs = effectiveTimeMs(verificationFile);
        for (const summaryFile of summaryFiles) {
            // The caller only needs whether the phase is stale, not which summary —
            // the first stale summary (in sorted order) is enough. Short-circuit.
            if (effectiveTimeMs(summaryFile) > verificationTimeMs) {
                return { determined: true, stale: true, verificationFile, summaryFile };
            }
        }
        return { determined: true, stale: false };
    }
    catch {
        return { determined: false };
    }
}
/**
 * Read the verification status from the first `*-VERIFICATION.md` file in
 * phaseDir and return the routing result.
 *
 * Behavior:
 * 1. Find the phase's verification report via `resolveVerificationFile`
 *    (canonical `<phase-token>-VERIFICATION.md` preferred; falls back to the
 *    alphabetically-first `*-VERIFICATION.md` that belongs to THIS phase when
 *    none is canonical — #3357/#3511). If none → status 'missing'.
 * 2. Extract `status` from FRONTMATTER ONLY via the shared extractFrontmatter
 *    parser (DEFECT.FRONTMATTER-SCALAR-BROAD-GREP fix — parser anchors at byte 0).
 *    If no frontmatter block or no `status` key → status 'missing'.
 * 3. Map to routing table. Unknown non-empty value → status 'unknown'.
 *
 * The internal staleness check can itself fail (fs / scanPhasePlans / clock
 * error); when it does, `status` is routed as if nothing were stale (the
 * pre-existing no-throw fail-open contract — unchanged), but the returned
 * result carries `staleCheckIndeterminate: true` so a caller can distinguish
 * "checked; nothing is stale" from "could not check" (#3057 B3).
 *
 * @param phaseDir - Absolute path to the phase directory.
 * @param opts     - Options. `opts.fs` allows test injection (defaults to node:fs).
 *                   `opts.runtime` selects the command surface `next_command` is
 *                   projected into (#2617).
 */
function readVerificationStatus(phaseDir, opts = {}) {
    const fsImpl = opts.fs ?? defaultFsImpl;
    const phaseCleanCommitTimesMs = opts.phaseCleanCommitTimesMs ?? defaultPhaseCleanCommitTimesMs;
    const runtime = opts.runtime ?? 'claude';
    // Phase token for the gaps_found command
    const baseName = node_path_1.default.basename(phaseDir);
    const phaseToken = extractPhaseToken(baseName);
    const derivedPhaseNumber = phaseToken.length > 0 ? phaseToken : baseName;
    // #2617: the phase number becomes a COMMAND ARGUMENT, so it is appended only
    // when it is unambiguously one. extractPhaseToken also returns project-code
    // forms (`PROJ-07`), which are indistinguishable by shape from an ordinary
    // directory name — `gsd-651-parent` yields `gsd-651` — and emitting
    // `execute-phase gsd-651` is worse than emitting no argument at all. Callers
    // that already know the number (init) pass it explicitly and always get it.
    const phaseArgSource = opts.phaseNumber ?? (/^\d+(\.\d+)*$/.test(derivedPhaseNumber) ? derivedPhaseNumber : '');
    const phaseArg = phaseArgSource ? ` ${phaseArgSource}` : '';
    // 1. Find *-VERIFICATION.md
    let verificationFile = null;
    try {
        const entries = fsImpl.readdirSync(phaseDir);
        // #3492: pin selection to THIS phase's own token (already derived above
        // for the routed command argument) so a stray cross-phase or
        // sentinel-numbered canonically-shaped file cannot outrank this phase's
        // own (possibly non-canonical) report. #3511: baseName also scopes the
        // fallback path to this same phase (see resolveVerificationFile docs).
        verificationFile = resolveVerificationFile(entries, { phaseToken, phaseDirName: baseName });
    }
    catch {
        // Directory unreadable → treat as missing
        verificationFile = null;
    }
    if (!verificationFile) {
        return missingResult(runtime, phaseArg);
    }
    // 2. Read and parse frontmatter using the shared parser.
    // extractFrontmatter anchors at byte 0, so body `status:` lines are ignored.
    const filePath = node_path_1.default.join(phaseDir, verificationFile);
    let rawStatus = null;
    let fm = {};
    try {
        // #3707-CR follow-up MINOR 1: normalize line endings at this read
        // boundary — this function's own `readFileSync` is the equivalent seam
        // `planning.inspect`'s `buildUatRows`/`readDocument` route through for
        // UAT/REQUIREMENTS documents, but `readVerificationStatus` had no such
        // normalization of its own. A lone-CR VERIFICATION.md's `---\r...\r---`
        // frontmatter fence never matched `extractFrontmatter`'s byte-0
        // `---\n`/`---\r\n` check, so `status: passed` was read as absent and
        // this function reported 'missing' — under-reporting a completed
        // verification as if the step never ran, the fail-safe direction but the
        // same root cause as the false-clean class fixed elsewhere in #3707-CR.
        const content = normalizeLineEndings(fsImpl.readFileSync(filePath, 'utf-8'));
        fm = extractFrontmatter(content, filePath);
        const statusVal = fm['status'];
        // status is always a scalar string in a well-formed VERIFICATION.md frontmatter;
        // only accept string values — arrays and objects are not valid status values.
        if (typeof statusVal === 'string') {
            const trimmed = statusVal.trim();
            rawStatus = trimmed.length > 0 ? trimmed : null;
        }
    }
    catch {
        rawStatus = null;
    }
    if (!rawStatus) {
        return missingResult(runtime, phaseArg);
    }
    // gaps_found takes priority over stale — gap closure is the correct next
    // step regardless of whether summaries are newer than the verification file.
    if (rawStatus === 'gaps_found') {
        const entry = VERIFICATION_ROUTING_TABLE['gaps_found'];
        return {
            status: entry.status,
            next_action: entry.next_action,
            next_command: projectNextCommand('plan-phase', runtime, `${phaseArg} --gaps`),
        };
    }
    // #4155: a report that declares a covered-input fingerprint is checked by
    // RECOMPUTING that fingerprint over current file content — strictly
    // content-grounded, and it REPLACES (not supplements) the legacy
    // SUMMARY-mtime check below for that report. A report with no fingerprint
    // metadata (every report written before #4155) keeps the exact legacy
    // mtime-based behavior, unchanged.
    const coveredFilesVal = fm['covered_files'];
    const coveredDigestVal = fm['covered_digest'];
    // A report OPTS IN to the fingerprint check by declaring EITHER field —
    // once opted in, an incomplete or malformed pair (one field present but
    // not the other, an empty array, a non-array, a blank digest) fails closed
    // to `stale` rather than silently downgrading to the weaker legacy
    // mtime-only check, which would only ever notice a newer SUMMARY.
    const declaresFingerprint = coveredFilesVal !== undefined || coveredDigestVal !== undefined;
    const hasWellFormedFingerprint = Array.isArray(coveredFilesVal) &&
        coveredFilesVal.length > 0 &&
        coveredFilesVal.every((f) => typeof f === 'string') &&
        typeof coveredDigestVal === 'string' &&
        coveredDigestVal.trim().length > 0;
    let staleCheckIndeterminate = false;
    let isStale;
    if (declaresFingerprint) {
        // Stated directly rather than relying on `null !== coveredDigestVal`
        // being true whenever the pair is malformed: `!hasWellFormedFingerprint`
        // fails closed explicitly, and its `||` short-circuit means
        // computeCoveredDigest/allCurrentArtifactsCovered never run on a
        // malformed (wrong-shaped) `coveredFilesVal`. The two `||`s after it
        // short-circuit in turn: the live-directory re-scan (for a plan/summary
        // added AFTER verification and never declared in covered_files) only
        // runs once the digest itself has already matched.
        isStale =
            !hasWellFormedFingerprint ||
                computeCoveredDigest((0, project_root_cjs_1.findProjectRoot)(phaseDir), coveredFilesVal) !== coveredDigestVal ||
                !allCurrentArtifactsCovered(phaseDir, coveredFilesVal);
    }
    else {
        const staleCheck = findStaleVerificationSummary(phaseDir, fsImpl, phaseCleanCommitTimesMs);
        isStale = staleCheck.determined && staleCheck.stale;
        // staleCheck is either {determined:true, stale:false} (checked; nothing
        // stale) or {determined:false} (could not check — fs/scan/clock failure).
        // Both fall through to normal routing below (the pre-existing no-throw
        // fail-open contract is unchanged), but the indeterminate case is flagged
        // on the returned result so a caller can tell the two apart (#3057 B3).
        staleCheckIndeterminate = !staleCheck.determined;
    }
    if (isStale) {
        const entry = VERIFICATION_ROUTING_TABLE['stale'];
        return {
            status: entry.status,
            next_action: entry.next_action,
            next_command: projectNextCommand('verify-work', runtime, phaseArg),
        };
    }
    // 3. Route — exclude internal sentinels from raw-file lookup (they are
    // constructed internally above, never written by the verifier).
    if (rawStatus in VERIFICATION_ROUTING_TABLE &&
        rawStatus !== 'missing' &&
        rawStatus !== 'unknown' &&
        rawStatus !== 'stale' &&
        rawStatus !== 'gaps_found') {
        const entry = VERIFICATION_ROUTING_TABLE[rawStatus];
        return {
            status: entry.status,
            next_action: entry.next_action,
            next_command: projectNextCommand(entry.next_command, runtime, phaseArg),
            ...(staleCheckIndeterminate ? { staleCheckIndeterminate: true } : {}),
        };
    }
    // Unknown value
    const unknownRoute = VERIFICATION_ROUTING_TABLE['unknown'];
    return {
        status: unknownRoute.status,
        next_action: `Unexpected verification status '${rawStatus}'. If this is an intentional non-standard marker (e.g. a hand-set failed/superseded state), no action is needed. Otherwise, run execute-phase to regenerate verification — it will not re-run plans that already have a SUMMARY.md.`,
        next_command: projectNextCommand(unknownRoute.next_command, runtime, phaseArg),
        ...(staleCheckIndeterminate ? { staleCheckIndeterminate: true } : {}),
    };
}
/**
 * isPhaseComplete — the single canonical owner of "is phase P complete?"
 * (ADR-3180 §7.4, Decision 1). Sited beside readVerificationStatus, which it
 * wraps.
 *
 * DISK-STRICT (#2957, maintainer decision 2026-08-08; ADR-3180 §7.4 amended
 * af92fd4c9): readVerificationStatus is called UNCONDITIONALLY here — plan
 * count is NOT a precondition. A phase with zero plans and a passing
 * `*-VERIFICATION.md` is complete (#3168). A ROADMAP checkbox has no machine
 * authority and is never consulted — this function never reads ROADMAP.md.
 *
 * `complete` is exactly `verification.status === 'passed'`. `verification`
 * carries the FULL routing result (status/next_action/next_command), so a
 * caller can distinguish a failing verdict (`gaps_found`/`human_needed`/
 * `stale`/`unknown`) from an absent one (`missing`) — both are "not
 * complete", but they are not the same non-answer.
 *
 * `scope` is UNREADABLE when `phaseDir` itself could not be listed — this is
 * INDEPENDENT of readVerificationStatus's own no-throw fail-open contract for
 * a missing `*-VERIFICATION.md` file (a well-formed answer,
 * `verification.status === 'missing'`, scope COMPLETE): a caller must not
 * read `value.complete: false` here as a confident "not complete" the way it
 * can for a genuinely-checked missing file.
 *
 * Does NOT import scanPhasePlans / plan-scan.cjs — the owner consumes plan
 * counts from its caller when a caller needs them for a different question
 * (e.g. buildPhaseCompletionProjection's own `implementation_complete`); it
 * never re-derives or requires them itself.
 */
function isPhaseComplete(phaseDir, deps = {}) {
    const fsImpl = deps.fs ?? defaultFsImpl;
    let readable = true;
    try {
        fsImpl.readdirSync(phaseDir);
    }
    catch {
        readable = false;
    }
    const verification = readVerificationStatus(phaseDir, {
        fs: deps.fs,
        phaseCleanCommitTimesMs: deps.phaseCleanCommitTimesMs,
        runtime: deps.runtime,
        phaseNumber: deps.phaseNumber,
    });
    return {
        value: {
            complete: verification.status === 'passed',
            verification,
        },
        scope: readable ? SCOPE.COMPLETE : SCOPE.UNREADABLE,
    };
}
/**
 * CLI command handler: resolve phaseDir against cwd, call readVerificationStatus,
 * emit via io.output().
 *
 * @param cwd         - Current working directory (used to resolve phaseDirArg).
 * @param phaseDirArg - Phase directory path (absolute or relative to cwd).
 * @param raw         - Whether to emit raw (non-JSON) output.
 */
function cmdVerificationStatus(cwd, phaseDirArg, raw) {
    if (!phaseDirArg) {
        error('phase directory required for verification.status');
        return;
    }
    const phaseDir = node_path_1.default.resolve(cwd, phaseDirArg);
    const result = readVerificationStatus(phaseDir, { runtime: (0, runtime_slash_cjs_1.resolveRuntime)(cwd) });
    output(result, raw);
}
/**
 * CLI command handler: resolve which `*-VERIFICATION.md` in `phaseDirArg` is
 * the phase's own report, via the shared `resolveVerificationFile` seam, and
 * emit its absolute path.
 *
 * #3492 F3: the ONE seam shell callers (verify-work.md's writer, transition.md's
 * awk reader) route through instead of hand-rolling `ls *-VERIFICATION.md |
 * head -1` / an awk glob scan — both of which pick alphabetically-first and so
 * diverge from every JS reader now pinned to the phase's own token.
 *
 * Emits `{ verification_file: "<absolute path>" | "" }` (empty when no
 * candidate resolves, including an unreadable directory). `raw` emits the
 * bare path string (possibly empty) so `VAR=$(gsd_run query
 * verification.resolve-file "$PHASE_DIR" --raw)` is directly assignable.
 *
 * @param cwd         - Current working directory (used to resolve phaseDirArg).
 * @param phaseDirArg - Phase directory path (absolute or relative to cwd).
 * @param raw         - Whether to emit raw (non-JSON) output.
 */
function cmdVerificationResolveFile(cwd, phaseDirArg, raw) {
    if (!phaseDirArg) {
        error('phase directory required for verification.resolve-file');
        return;
    }
    const phaseDir = node_path_1.default.resolve(cwd, phaseDirArg);
    let verificationPath = '';
    try {
        const entries = node_fs_1.default.readdirSync(phaseDir);
        const phaseDirName = node_path_1.default.basename(phaseDir);
        const phaseToken = extractPhaseToken(phaseDirName);
        const verificationFile = resolveVerificationFile(entries, { allowBare: true, phaseToken, phaseDirName });
        if (verificationFile) {
            verificationPath = node_path_1.default.join(phaseDir, verificationFile);
        }
    }
    catch {
        verificationPath = '';
    }
    output({ verification_file: verificationPath }, raw, verificationPath);
}
/**
 * CLI command handler (#4155): compute the covered-input fingerprint the
 * verifier embeds in VERIFICATION.md frontmatter (`covered_files`,
 * `covered_digest`). The verifier is an LLM agent, not a hashing engine —
 * this command does the deterministic math so the agent only has to name
 * the covered paths and copy the result into frontmatter.
 *
 * Emits `{ covered_files: <sorted deduped paths>, covered_digest: <digest> }`
 * on success. A covered path that is missing, unreadable, or escapes the
 * project root fails the WHOLE command (fail closed — a partial fingerprint
 * would be worse than none): `error()` is called and nothing is emitted.
 *
 * @param cwd         - Current working directory.
 * @param phaseDirArg - Phase directory path (absolute or relative to cwd);
 *                       its project root is the base covered paths resolve against.
 * @param files       - Covered-input paths, relative to the project root.
 * @param raw         - Whether to emit raw (non-JSON) output: just the
 *                       `covered_digest` string, so `VAR=$(gsd_run query
 *                       verification.fingerprint "$PHASE_DIR" ... --raw)` is
 *                       directly assignable. `covered_files` is unambiguous
 *                       from the caller's own input list in that mode, so
 *                       only the computed digest needs a raw form.
 */
function cmdVerificationFingerprint(cwd, phaseDirArg, files, raw) {
    if (!phaseDirArg) {
        error('phase directory required for verification.fingerprint');
        return;
    }
    if (files.length === 0) {
        error('at least one covered file required for verification.fingerprint');
        return;
    }
    const phaseDir = node_path_1.default.resolve(cwd, phaseDirArg);
    const projectRoot = (0, project_root_cjs_1.findProjectRoot)(phaseDir);
    // canonicalizeCoveredFiles here is for the emitted `covered_files` field —
    // computeCoveredDigest canonicalizes its own `coveredFiles` argument
    // internally too (it must, for callers like readVerificationStatus that
    // pass raw, un-canonicalized frontmatter values), so passing an
    // already-canonical list keeps that internal pass a cheap no-op rather
    // than a second meaningfully different canonicalization.
    const uniqueSorted = canonicalizeCoveredFiles(files);
    const digest = computeCoveredDigest(projectRoot, uniqueSorted);
    if (digest === null) {
        error('could not compute fingerprint — a covered file is missing, unreadable, or escapes the project root');
        return;
    }
    output({ covered_files: uniqueSorted, covered_digest: digest }, raw, digest);
}
module.exports = {
    VERIFIER_STATUSES,
    VERIFICATION_ROUTING_TABLE,
    defaultPhaseCleanCommitTimesMs,
    resolveVerificationFile,
    resolveUatFile,
    findStaleVerificationSummary,
    readVerificationStatus,
    isPhaseComplete,
    cmdVerificationStatus,
    cmdVerificationResolveFile,
    computeCoveredDigest,
    cmdVerificationFingerprint,
};
