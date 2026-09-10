"use strict";
/**
 * Reviewer Lane Invocation Module (ADR-2782 Phase 5b, #2799 — closes #2718).
 *
 * Turns a DECLARED lane (`review-lane-descriptor.cts`) plus resolved configuration into a concrete,
 * inspectable INVOCATION PLAN. Phase 1's module declares; this one resolves; `review-lane-runner`
 * executes. The split exists so the interesting half is pure: a plan is a value, so the twelve
 * shipped lanes can be asserted against a golden table without spawning anything.
 *
 * WHY A GOLDEN TABLE AND NOT A FRESH DESIGN (Gall's Law). The 640 lines of hand-authored bash this
 * replaces is the simple system that worked, and every leg encodes a hard-won fix — #2494 and #2605
 * (empty output), #1698 (Codex stdout teardown noise), #1936 (OpenCode zero-output turns), #2073
 * (Antigravity's three modes), #2176 (repo-root anchoring), #2589 (no jq on stock Windows), #2794
 * (Qwen's missing sidecar). A resolver designed from the descriptor TYPES would throw that away and
 * rebuild the bugs. So each lane's plan was derived from its leg, and `tests/review-lane-invocation`
 * asserts all twelve against a frozen table. Old and new cannot literally run in parallel, so that
 * table is the strangler-fig substitute — it is what makes this cutover safe rather than hopeful.
 *
 * PURE. No filesystem, no network, no subprocess, no clock. Configuration arrives through the
 * `configGet` seam so a test drives it with a plain object and production wires it to the real
 * resolved config. Every function here is total: a malformed lane yields an `unavailable` result,
 * never a throw — a resolver that throws on bad input cannot report on it, and this runs against
 * third-party overlay manifests (ADR-2782 D4).
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.LANE_UNAVAILABLE = void 0;
exports.configString = configString;
exports.normalizeHost = normalizeHost;
exports.resolveLaneEffort = resolveLaneEffort;
exports.resolveTimeoutMs = resolveTimeoutMs;
exports.nativeTimeoutToken = nativeTimeoutToken;
exports.isEmptyReview = isEmptyReview;
exports.fileRefPrompt = fileRefPrompt;
exports.resolveLanePlan = resolveLanePlan;
const review_lane_descriptor_cjs_1 = require("./review-lane-descriptor.cjs");
/* ------------------------------------------------------------------ *
 * Unavailability — a frozen enum, because the reason is the product
 * ------------------------------------------------------------------ */
/**
 * Why a lane will not run.
 *
 * Frozen and exhaustive because the bash it replaces had exactly one outcome for every failure —
 * an empty file — and that ambiguity IS the defect class this epic exists to close (#2494/#2605: a
 * failed lane was indistinguishable from a reviewer that ran cleanly with nothing to report). The
 * caller renders these; tests assert on them. Never assert on the rendered prose.
 *
 * Adding a member is three coordinated changes: this enum, the emitting site, and the test locking
 * `Object.keys(...).sort()`.
 */
exports.LANE_UNAVAILABLE = Object.freeze({
    MALFORMED_LANE: 'malformed_lane',
    UNKNOWN_HANDLER: 'unknown_handler',
    UNKNOWN_TRANSPORT: 'unknown_transport',
    MISSING_BINARY: 'missing_binary',
    MISSING_REQUIRED_BINARY: 'missing_required_binary',
    PROBE_FAILED: 'probe_failed',
    PROBE_TIMEOUT: 'probe_timeout',
    HOST_UNREACHABLE: 'host_unreachable',
    EGRESS_HOST_CHANGED: 'egress_host_changed',
    BUDGET_TOO_SMALL: 'budget_too_small',
    BUDGET_TOOL_FAILED: 'budget_tool_failed',
});
/** Every handler name the runner can dispatch. Mirrors `LaneHandler` (D6's closed enum). */
const KNOWN_HANDLERS = new Set(['antigravity', 'openai-compatible', 'opencode']);
/* ------------------------------------------------------------------ *
 * Value normalization — the stringly-typed edges the bash lived with
 * ------------------------------------------------------------------ */
/**
 * A configured string value, or `null` when effectively unset.
 *
 * Four shapes all mean "not configured", and the bash had to handle three of them by hand:
 *   - absent / `null` / `undefined`;
 *   - `""` — the declared default of every federated `review.models.*` key;
 *   - `"null"` — the LITERAL four characters `config-get --raw` prints for a missing key, which
 *     every leg tested for (`[ "$X" != "null" ]`). Reading config in-process removes the source of
 *     this, but a `.planning/config.json` written by an older workflow can still contain it;
 *   - whitespace-only — never a meaningful model name or host.
 *
 * A non-string (number, bool, object, array) is NOT coerced. `String(0)` would put `"0"` into argv
 * as a model name; a wrong model silently reviewed is worse than no model override.
 *
 * Exported and shared with the runner's model-recovery arms (#2295) — "what counts as unset" has
 * ONE source, so the plan resolver and the runner's recovered-model normalization cannot disagree.
 */
function configString(raw) {
    if (typeof raw !== 'string')
        return null;
    const trimmed = raw.trim();
    if (trimmed === '' || trimmed === 'null' || trimmed === 'undefined')
        return null;
    return trimmed;
}
/**
 * Normalize a base URL for storage and for the D5 consent comparison.
 *
 * Trailing slash, case in the scheme/host, and an explicit default port are all the same
 * destination. Without normalizing, a cosmetic `.planning/config.json` edit — adding a trailing
 * slash — would read as "the egress destination changed" and block the lane, training the user to
 * dismiss the one warning that matters.
 *
 * Returns the input trimmed when it is not parseable as a URL: an unparseable host is compared
 * verbatim rather than silently rewritten.
 */
function normalizeHost(raw) {
    const trimmed = String(raw ?? '').trim();
    if (!trimmed)
        return '';
    let u;
    try {
        u = new URL(trimmed);
    }
    catch {
        return trimmed.replace(/\/+$/, '');
    }
    // `new URL('localhost:11434')` PARSES — protocol `localhost:`, empty hostname — so a plausible
    // but scheme-less config value would otherwise be rewritten to `localhost://11434` and compared
    // (and requested) as though it were a real destination. No hostname means this is not a URL;
    // return it verbatim so it fails visibly rather than silently becoming something else.
    if (!u.hostname)
        return trimmed.replace(/\/+$/, '');
    const scheme = u.protocol.toLowerCase();
    const isDefaultPort = (scheme === 'http:' && u.port === '80') || (scheme === 'https:' && u.port === '443');
    const port = isDefaultPort ? '' : u.port;
    const host = u.hostname.toLowerCase();
    const pathPart = u.pathname.replace(/\/+$/, '');
    return `${scheme}//${host}${port ? `:${port}` : ''}${pathPart}`;
}
/** Levels GSD's effort axis accepts (#3533). `inherit` selects the no-argument path. */
const EFFORT_LEVELS = new Set([
    'minimal', 'low', 'medium', 'high', 'xhigh', 'max', 'inherit',
]);
/**
 * Resolve one lane's reasoning effort from REVIEW configuration (#4255).
 *
 * Resolution order, highest first:
 *   1. `lane.effortConfigKey` — the per-lane review effort the operator set
 *   2. `lane.defaultEffort` — the lane's declared review default (`high` for prompt-fed,
 *      source-grounded lanes)
 *   3. nothing — no effort argument is emitted and the reviewer CLI's own configuration decides
 *
 * A configured `'inherit'` selects (3) explicitly. An unrecognized level is REFUSED rather than
 * passed to the host: it falls back to the lane default, because forwarding a typo would render an
 * argument the CLI rejects and kill the lane outright.
 *
 * What this function deliberately does NOT do is consult any agent's execution settings. Before
 * #4255 the level came from `gsd-plan-checker`'s installed frontmatter through a hardcoded agent
 * id, so every lane ran at a fast structural verifier's `low` — and, because the rendered argument
 * is a CLI config override, it silently beat the effort the operator had configured for that CLI
 * itself. A value inherited from an unrelated agent is worse than no value at all, which is why
 * (3) emits nothing rather than falling back to some other agent's number.
 *
 * `renderArgv` is injected (the host table and the ADR-2481 surface negotiation live in
 * `model-catalog` / `commands`, above this module's layer) so this stays a pure function of its
 * inputs and the golden lane table can assert it without a spawn.
 */
function resolveLaneEffort(lane, configGet, renderArgv) {
    const none = { argv: [], value: null, source: 'none' };
    if (!lane || typeof lane !== 'object')
        return none;
    const configured = lane.effortConfigKey ? configString(configGet(lane.effortConfigKey)) : null;
    const valid = configured !== null && EFFORT_LEVELS.has(configured) ? configured : null;
    const level = valid ?? configString(lane.defaultEffort);
    if (level === null || level === 'inherit')
        return none;
    const rendered = renderArgv(lane.slug, level);
    const argv = (rendered.argv ?? []).filter((a) => typeof a === 'string' && a !== '');
    if (argv.length === 0)
        return none;
    return {
        argv,
        value: configString(rendered.value) ?? level,
        source: valid !== null ? 'config' : 'lane-default',
    };
}
function resolveTimeoutMs(timeoutConfigKey, floorMs, configGet) {
    const configuredSeconds = typeof timeoutConfigKey === 'string' ? configGet(timeoutConfigKey) : undefined;
    return typeof configuredSeconds === 'number' && Number.isFinite(configuredSeconds) && configuredSeconds > 0
        ? configuredSeconds * 1000
        : floorMs;
}
/** Buffer (seconds) a lane's native inner timeout sits under its resolved outer wall-clock cap
 * (#3274). Matches the shipped 600s outer / 540s native relationship exactly when unconfigured:
 * floor(600000/1000) - 60 = 540. */
const NATIVE_TIMEOUT_BUFFER_SECONDS = 60;
/**
 * Render the `{{nativeTimeout}}` argv placeholder from a lane's resolved outer timeout (#3274).
 *
 * Clamped to a 1-second floor so a very small configured (or, today, only-ever-default) outer
 * timeout never produces a zero or negative duration string a CLI would reject or misinterpret.
 */
function nativeTimeoutToken(timeoutMs) {
    const seconds = Math.max(1, Math.floor(timeoutMs / 1000) - NATIVE_TIMEOUT_BUFFER_SECONDS);
    return `${seconds}s`;
}
/**
 * Classify a lane's output as a review or as empty.
 *
 * WHITESPACE-ONLY COUNTS AS EMPTY, for every lane. The bash tested `[ ! -s file ]`, which counts
 * BYTES — so a reply of three spaces passed as a successful review. Two legs (LM Studio,
 * llama.cpp) closed this locally with a case-glob; gemini, claude, codex, qwen and cursor did not.
 * Making it uniform is a deliberate, disclosed behavior change (a bug fix that breaks a workaround)
 * and is why this phase ships a changeset note.
 *
 * The `-n` / `-e` / `-E` hazard the two printf-using legs guarded against cannot occur here at all:
 * nothing in this path goes through `echo`.
 */
function isEmptyReview(text) {
    if (typeof text !== 'string')
        return true;
    return text.trim().length === 0;
}
/**
 * The standard `argv-file-ref` prompt (#2176).
 *
 * Two lanes take the prompt as an ARGUMENT rather than on stdin, and a full plan set inline would
 * approach the 32,767-character Windows `execFileSync` ceiling — so the argument is a short
 * instruction naming the prompt file. It must also carry the ABSOLUTE repo root: an argv-fed CLI
 * does not reliably inherit the review's working directory, and without the anchor the reviewer
 * reviews the plan text in isolation, which is exactly what the Review Instructions forbid.
 *
 * `antigravity` deliberately does NOT use this text — its handler owns a variant that additionally
 * demands a `REVIEWED-WITHOUT-REPO-ACCESS` self-report.
 */
function fileRefPrompt(promptPath, repoRoot) {
    return (`Read the file at ${promptPath} in full and carry out the review request it contains. ` +
        `The repository under review is at ${repoRoot} — resolve every relative file path in the ` +
        `review request against that absolute root. Output only the resulting markdown review. ` +
        `Do not edit any files.`);
}
/** Run-dir artifact paths. POSIX-joined: these are workflow-visible strings, not OS paths. */
function artifactPaths(runDir, slug) {
    const base = String(runDir ?? '').replace(/\/+$/, '');
    return {
        promptPath: `${base}/gsd-review-prompt.md`,
        reviewPath: `${base}/gsd-review-${slug}.md`,
        errPath: `${base}/gsd-review-${slug}.err`,
    };
}
/* ------------------------------------------------------------------ *
 * Resolution
 * ------------------------------------------------------------------ */
/**
 * Resolve one declared lane into an executable plan.
 *
 * TOTAL: never throws. Every rejection is a typed `LaneUnavailableReason`, because the caller has to
 * tell three different unavailabilities apart — a lane that is absent, a lane whose probe failed,
 * and a lane blocked on a changed egress destination are not the same event to a user.
 *
 * NOT resolved here: probe execution, prompt-budget trimming, and the D5 egress-host comparison.
 * Those need I/O and live in the runner. This function decides SHAPE.
 */
function resolveLanePlan(input) {
    const warnings = [];
    const fail = (reason, detail) => ({
        ok: false,
        reason,
        detail,
        warnings,
    });
    // Trust boundary: the declared type says ReviewerLane, but overlay manifests arrive here.
    const raw = input?.lane;
    if (raw === null || typeof raw !== 'object' || Array.isArray(raw)) {
        return fail(exports.LANE_UNAVAILABLE.MALFORMED_LANE, `lane is not an object: ${String(raw)}`);
    }
    const lane = raw;
    const slug = typeof lane.slug === 'string' ? lane.slug.trim() : '';
    if (!slug) {
        return fail(exports.LANE_UNAVAILABLE.MALFORMED_LANE, 'lane declares no slug');
    }
    // The slug is CONCATENATED into artifact paths below, so the grammar is enforced here rather
    // than trusted from upstream. `checkReviewerLaneParity` and the capability validator both check
    // it too, but neither runs on this path — and this module's whole premise is that it is the
    // trust boundary for third-party overlay manifests. A slug of `../../../tmp/evil` would
    // otherwise produce a reviewPath outside the run dir that `writeReviewOrStub` happily writes to.
    if (!review_lane_descriptor_cjs_1.LANE_SLUG_RE.test(slug)) {
        return fail(exports.LANE_UNAVAILABLE.MALFORMED_LANE, `lane slug '${slug}' is outside the declared grammar ${String(review_lane_descriptor_cjs_1.LANE_SLUG_RE)}`);
    }
    // D4 rule 4: an unknown handler FAILS CLOSED. A lane naming imperative code this GSD version does
    // not have cannot be run "mostly" — the handler is precisely the part data could not express.
    const handler = lane.handler ?? null;
    if (handler !== null && !KNOWN_HANDLERS.has(handler)) {
        return fail(exports.LANE_UNAVAILABLE.UNKNOWN_HANDLER, `lane '${slug}' names handler '${String(handler)}', which this GSD version does not provide`);
    }
    const { promptPath, reviewPath, errPath } = artifactPaths(input.runDir, slug);
    const floorMs = typeof lane.timeoutFloorMs === 'number' && Number.isFinite(lane.timeoutFloorMs) && lane.timeoutFloorMs > 0
        ? lane.timeoutFloorMs
        : 900_000;
    const timeoutMs = resolveTimeoutMs(lane.timeoutConfigKey, floorMs, input.configGet);
    const emptyOutput = lane.emptyOutput === 'handler-owned' ? 'handler-owned' : 'stub-with-stderr';
    // #3194: only an EXACT 'diff-only' declaration exempts a lane from evidence verification.
    // Anything else — including a missing or garbage value on a third-party overlay body —
    // resolves as 'source-grounded', so the runner verifies rather than trusts it.
    const evidenceClass = lane.evidenceClass === 'diff-only' ? 'diff-only' : 'source-grounded';
    const requiresBinaries = Array.isArray(lane.requiresBinaries)
        ? lane.requiresBinaries.filter((b) => typeof b === 'string')
        : [];
    const model = configString(typeof lane.modelConfigKey === 'string' ? input.configGet(lane.modelConfigKey) : undefined);
    if (lane.transport === 'openai-http') {
        const rawInvoke = lane.invoke;
        // The spawn branch below guards with `inv?.binary`; this one must too. Without it a lane
        // declaring `transport: 'openai-http'` and no `invoke` THROWS, which breaks this module's
        // documented totality — and a throw here is worse than it looks: the CLI seam resolves every
        // selected lane in one `.map`, so one malformed overlay manifest would abort the whole review
        // rather than dropping its own lane.
        if (rawInvoke === null || typeof rawInvoke !== 'object' || Array.isArray(rawInvoke)) {
            return fail(exports.LANE_UNAVAILABLE.MALFORMED_LANE, `openai-http lane '${slug}' declares no invoke object`);
        }
        const inv = rawInvoke;
        const hostConfigKey = typeof inv.hostConfigKey === 'string' ? inv.hostConfigKey : '';
        const configured = hostConfigKey ? configString(input.configGet(hostConfigKey)) : null;
        // Only a STRING declares a host. Coercing an object would produce the literal
        // '[object Object]' and normalize THAT as the lane's egress destination.
        const declaredDefault = typeof inv.defaultHost === 'string' ? inv.defaultHost : '';
        const host = normalizeHost(configured ?? declaredDefault);
        if (!host) {
            return fail(exports.LANE_UNAVAILABLE.MALFORMED_LANE, `lane '${slug}' resolves no host: '${hostConfigKey}' is unset and it declares no defaultHost`);
        }
        const apiPath = typeof inv.path === 'string' && inv.path ? inv.path : '/v1/chat/completions';
        const discovers = inv.modelDiscovery === 'first-from-models-endpoint';
        return {
            ok: true,
            warnings,
            plan: {
                transport: 'openai-http',
                slug,
                host,
                hostConfigKey,
                url: `${host}${apiPath}`,
                modelsUrl: discovers ? `${host}/v1/models` : null,
                model,
                fallbackModel: typeof inv.fallbackModel === 'string' ? inv.fallbackModel : 'local-model',
                promptPath,
                reviewPath,
                errPath,
                timeoutMs,
                emptyOutput,
                evidenceClass,
                handler,
                requiresBinaries,
                probe: lane.probe,
            },
        };
    }
    if (lane.transport !== 'spawn') {
        return fail(exports.LANE_UNAVAILABLE.UNKNOWN_TRANSPORT, `lane '${slug}' declares transport '${String(lane.transport)}'`);
    }
    const inv = lane.invoke;
    const binary = typeof inv?.binary === 'string' ? inv.binary.trim() : '';
    if (!binary) {
        return fail(exports.LANE_UNAVAILABLE.MALFORMED_LANE, `spawn lane '${slug}' declares no binary`);
    }
    // Output target first: `{{output}}` needs to know the path, and the target is also what tells the
    // runner whether to capture stdout or read a file the tool wrote itself (#1698).
    let outputTarget = { kind: 'stdout' };
    let outputExpansion = [];
    if (inv.outputChannel === 'file-arg') {
        const outputArg = typeof inv.outputArg === 'string' ? inv.outputArg : '';
        if (!outputArg) {
            return fail(exports.LANE_UNAVAILABLE.MALFORMED_LANE, `lane '${slug}' declares outputChannel 'file-arg' with no outputArg naming the argument`);
        }
        outputExpansion = [outputArg, reviewPath];
        outputTarget = { kind: 'file', path: reviewPath };
    }
    let stdin = null;
    let promptExpansion = [];
    switch (inv.promptChannel) {
        case 'stdin':
            stdin = promptPath; // the runner streams this file; the plan names it.
            break;
        case 'argv-file-ref':
            promptExpansion = [fileRefPrompt(promptPath, input.repoRoot)];
            break;
        case 'argv':
            promptExpansion = [promptPath];
            break;
        case 'none':
            break; // CodeRabbit reviews the working tree and is fed nothing (review.md:367).
        default:
            return fail(exports.LANE_UNAVAILABLE.MALFORMED_LANE, `lane '${slug}' declares promptChannel '${String(inv.promptChannel)}'`);
    }
    const modelExpansion = model && typeof inv.modelArg === 'string' && inv.modelArg ? [inv.modelArg, model] : [];
    const effortExpansion = inv.effortChannel === 'argv'
        ? (input.effortArgs ?? []).filter((a) => typeof a === 'string' && a !== '')
        : [];
    // Expand the argv template in declared order. A placeholder with nothing to contribute expands to
    // ZERO elements and disappears — that is what lets one template serve both the configured and the
    // unconfigured case without a conditional in the data.
    const expansions = {
        '{{model}}': modelExpansion,
        '{{effort}}': effortExpansion,
        '{{output}}': outputExpansion,
        '{{prompt}}': promptExpansion,
        '{{nativeTimeout}}': [nativeTimeoutToken(timeoutMs)],
    };
    const template = Array.isArray(inv.args)
        ? inv.args.filter((a) => typeof a === 'string')
        : [];
    const argv = [];
    for (const token of template) {
        // Own-property lookup: a lane could declare a literal `constructor` argument, and prototype
        // members must never resolve as expansions.
        if (Object.prototype.hasOwnProperty.call(expansions, token)) {
            argv.push(...expansions[token]);
        }
        else {
            argv.push(token);
        }
    }
    // Per-invocation env pairs (#2483). Own string-valued entries only — a non-string is dropped,
    // never coerced, and prototype members never resolve (same lookup discipline as the argv
    // expansions above). An empty or absent declaration resolves to `null`, so the runner has one
    // shape to test.
    let env = null;
    const declaredEnv = inv.env;
    if (declaredEnv !== null && typeof declaredEnv === 'object' && !Array.isArray(declaredEnv)) {
        const source = declaredEnv;
        const pairs = {};
        for (const k of Object.keys(source)) {
            const v = source[k];
            if (typeof v === 'string')
                pairs[k] = v;
        }
        if (Object.keys(pairs).length > 0)
            env = pairs;
    }
    return {
        ok: true,
        warnings,
        plan: {
            transport: 'spawn',
            slug,
            binary,
            argv,
            model: modelExpansion.length > 0 ? model : null,
            effort: effortExpansion.length > 0 ? (configString(input.effortValue) ?? null) : null,
            stdin,
            promptPath,
            outputTarget,
            reviewPath,
            errPath,
            timeoutMs,
            emptyOutput,
            evidenceClass,
            handler,
            requiresBinaries,
            probe: lane.probe,
            env,
        },
    };
}
