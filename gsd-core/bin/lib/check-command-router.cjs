"use strict";
/**
 * Check subcommand router — auto-mode, decision-coverage-plan, decision-coverage-verify.
 *
 * ADR-457 build-at-publish: the hand-written bin/lib/check-command-router.cjs collapsed
 * to a TypeScript source of truth. Behaviour is preserved byte-for-behaviour
 * from the prior hand-written .cjs; only strict types are added.
 */
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
const node_fs_1 = __importDefault(require("node:fs"));
const node_path_1 = __importDefault(require("node:path"));
const node_child_process_1 = require("node:child_process");
// eslint-disable-next-line @typescript-eslint/no-require-imports
const io = require("./io.cjs");
const { output, error, ERROR_REASON } = io;
// eslint-disable-next-line @typescript-eslint/no-require-imports
const planningWorkspaceMod = require("./planning-workspace.cjs");
const { planningDir } = planningWorkspaceMod;
// eslint-disable-next-line @typescript-eslint/no-require-imports
const phaseLocatorMod = require("./phase-locator.cjs");
const { findPhaseInternal } = phaseLocatorMod;
const decisions_cjs_1 = require("./decisions.cjs");
// eslint-disable-next-line @typescript-eslint/no-require-imports
const frontmatterMod = require("./frontmatter.cjs");
const { extractFrontmatter } = frontmatterMod;
const markdown_sectionizer_cjs_1 = require("./markdown-sectionizer.cjs");
const security_cjs_1 = require("./security.cjs");
const ui_safety_gate_cjs_1 = require("./ui-safety-gate.cjs");
const ui_frontend_evidence_cjs_1 = require("./ui-frontend-evidence.cjs");
// eslint-disable-next-line @typescript-eslint/no-require-imports
const verifyModule = require("./verify.cjs");
const { cmdVerifySchemaDrift, cmdVerifyCodebaseDrift, cmdVerifyContextDrift } = verifyModule;
// eslint-disable-next-line @typescript-eslint/no-require-imports
const roadmapModule = require("./roadmap.cjs");
const { getRoadmapPhaseWithFallback } = roadmapModule;
// eslint-disable-next-line @typescript-eslint/no-require-imports
const gapCheckerModule = require("./gap-checker.cjs");
const { runGapAnalysis } = gapCheckerModule;
const prohibition_enforcement_cjs_1 = require("./prohibition-enforcement.cjs");
const tdd_red_evidence_cjs_1 = require("./tdd-red-evidence.cjs");
// eslint-disable-next-line @typescript-eslint/no-require-imports
const gatePredicateEval = require("./gate-predicate-evaluator.cjs");
const { evaluatePredicate } = gatePredicateEval;
// eslint-disable-next-line @typescript-eslint/no-require-imports
const apiCoverageMod = require("./api-coverage.cjs");
const { detectApiIntegration, validateCoverageMatrix } = apiCoverageMod;
const shell_command_projection_cjs_1 = require("./shell-command-projection.cjs");
// eslint-disable-next-line @typescript-eslint/no-require-imports
const planScanMod = require("./plan-scan.cjs");
const { scanPhasePlans } = planScanMod;
// eslint-disable-next-line @typescript-eslint/no-require-imports
const planningScopeMod = require("./planning-scope.cjs");
const { SCOPE } = planningScopeMod;
// eslint-disable-next-line @typescript-eslint/no-require-imports
const verifyCommandGroundingMod = require("./verify-command-grounding.cjs");
const { probePhaseVerifyCommands, probePhaseFailingDirections } = verifyCommandGroundingMod;
// ─── Helpers ──────────────────────────────────────────────────────────────────
function normalizePhrase(text) {
    // eslint-disable-next-line @typescript-eslint/no-base-to-string
    return String(text || '')
        .toLowerCase()
        .replace(/[^a-z0-9\s]/g, ' ')
        .replace(/\s+/g, ' ')
        .trim();
}
const SOFT_PHRASE_MIN_WORDS = 6;
function softPhrase(text) {
    const words = normalizePhrase(text).split(' ').filter(Boolean);
    if (words.length < SOFT_PHRASE_MIN_WORDS)
        return '';
    return words.slice(0, SOFT_PHRASE_MIN_WORDS).join(' ');
}
function decisionMentioned(haystack, decision) {
    if (!haystack)
        return false;
    if (new RegExp(`\\b${decision.id}\\b`).test(haystack))
        return true;
    const phrase = softPhrase(decision.text);
    return phrase ? normalizePhrase(haystack).includes(phrase) : false;
}
function readIfExists(filePath) {
    try {
        return node_fs_1.default.readFileSync(filePath, 'utf-8');
    }
    catch {
        return '';
    }
}
function resolvePath(inputPath, projectDir) {
    return node_path_1.default.isAbsolute(inputPath) ? inputPath : node_path_1.default.join(projectDir, inputPath);
}
function readWorkflowConfig(projectDir) {
    const configPath = node_path_1.default.join(projectDir, '.planning', 'config.json');
    try {
        const parsed = JSON.parse(node_fs_1.default.readFileSync(configPath, 'utf-8'));
        const wf = parsed['workflow'] || {};
        return {
            ...wf,
            auto_advance: (wf['auto_advance'] ?? parsed['auto_advance']),
            _auto_chain_active: (wf['_auto_chain_active'] ?? parsed['_auto_chain_active']),
            context_coverage_gate: (wf['context_coverage_gate'] ?? parsed['context_coverage_gate']),
        };
    }
    catch {
        return {};
    }
}
function cmdAutoMode(projectDir, raw) {
    const workflow = readWorkflowConfig(projectDir);
    const autoAdvance = Boolean(workflow.auto_advance ?? false);
    const autoChainActive = Boolean(workflow._auto_chain_active ?? false);
    let source = 'none';
    if (autoChainActive && autoAdvance)
        source = 'both';
    else if (autoChainActive)
        source = 'auto_chain';
    else if (autoAdvance)
        source = 'auto_advance';
    output({
        active: autoChainActive || autoAdvance,
        source,
        auto_chain_active: autoChainActive,
        auto_advance: autoAdvance,
    }, raw, undefined);
}
function gateEnabled(projectDir) {
    const value = readWorkflowConfig(projectDir).context_coverage_gate;
    if (typeof value === 'boolean')
        return value;
    if (typeof value === 'string') {
        const lower = value.toLowerCase();
        if (lower === 'false' || lower === 'true')
            return lower !== 'false';
    }
    return true;
}
function loadPlanContents(phaseDir) {
    if (!node_fs_1.default.existsSync(phaseDir))
        return [];
    // #3183 (lint-plan-count-drift): source live plan files from the single
    // owner (scanPhasePlans) instead of a local `-PLAN.md` readdirSync filter
    // — picks up bare PLAN.md and nested plans/, and excludes plans marked
    // `status: superseded`, which the prior root-only exact-suffix filter did
    // neither for.
    return scanPhasePlans(phaseDir).planFiles
        .map((entry) => readIfExists(node_path_1.default.join(phaseDir, entry)));
}
const DESIGNATED_HEADINGS_RE = /^#{1,6}\s+(?:must[_ ]haves?|truths?|tasks?|objective)\b/i;
// #2372: scanned-tag set must match the planner-canonical surfaces where a D-NN citation
// is meaningful. `<objective>`/`<tasks>`/`<task>`/`<action>` are the historical core. The
// planner is also explicitly told (plan-phase.md) to cite decisions in `<read_first>`,
// `<behavior>`, `<verify>`, `<acceptance_criteria>`, and `<done>` — those are now scanned too,
// so the gate no longer reports a false coverage gap when a decision is cited in any of them.
//
// Implementation: per-tag matching, NOT a single wide alternation. A single alternation
// like `<(?:a|b|c)>...<\/(?:a|b|c)>` halts the outer tag's body capture at any inner tag
// in the set, dropping any citation in the outer tag's prefix prose — e.g.
// `<action>per D-05 <verify>npm test</verify></action>` would lose D-05 because `<verify>`
// halts the `<action>` body before the citation. Per-tag matching avoids this: each tag's
// body terminates only at its OWN closing tag, so `<verify>` inside `<action>` is absorbed
// into `<action>`'s body (D-05 caught) AND `<verify>` is matched separately on its own pass.
// Each per-tag regex keeps the ReDoS-safe negative-lookahead tempering (#2128).
const XML_DECISION_TAG_NAMES = ['objective', 'tasks', 'task', 'action', 'read_first', 'behavior', 'verify', 'acceptance_criteria', 'done'];
function buildXmlDecisionTagRegex(tagName) {
    // Per-tag: body tempering stops only at the SAME tag's reopening or closing — other
    // scanned tags pass through as text into this body. Non-greedy `*?` to first close.
    return new RegExp(`<${tagName}(?:\\s[^>]{0,1000})?>((?:(?!<${tagName}[\\s>])[\\s\\S])*?)<\\/${tagName}>`, 'gi');
}
function stripCommentsAndFences(text) {
    // HTML-comment stripping stays caller-side (the seam does not strip HTML comments).
    // Stop-at-next-open body (ReDoS-safe, #2128); an UNCLOSED `<!--` does not match,
    // so downstream tags are preserved (unlike a `(?:-->|$)` fallback, which would
    // wipe to EOF and fail-close the decision-coverage gate).
    const htmlStripped = text.replace(/<!--(?:(?!<!--)[\s\S])*?-->/g, ' ');
    // Fenced-code stripping: delegate to the canonical CommonMark-correct seam.
    // replaces the prior independent regex copy (```` ``` ``` ````  + `~~~ ~~~`).
    return (0, markdown_sectionizer_cjs_1.stripFencedCode)(htmlStripped).text;
}
function extractYamlBlock(frontmatter, key) {
    const match = frontmatter.match(new RegExp(`^${key}\\s*:(.*)$`, 'm'));
    if (!match)
        return '';
    const startIdx = (match.index || 0) + match[0].length;
    const rest = frontmatter.slice(startIdx + 1).split(/\r?\n/);
    const block = [match[1] || ''];
    for (const line of rest) {
        if (line === '' || /^\s/.test(line))
            block.push(line);
        else
            break;
    }
    return block.join('\n');
}
function extractXmlTagBodies(text) {
    const parts = [];
    for (const tagName of XML_DECISION_TAG_NAMES) {
        const re = buildXmlDecisionTagRegex(tagName);
        for (const match of text.matchAll(re)) {
            if (match[1])
                parts.push(match[1]);
        }
    }
    return parts.join('\n');
}
function extractPlanDesignatedSections(planContent) {
    if (!planContent)
        return '';
    const cleaned = stripCommentsAndFences(planContent);
    const fmMatch = cleaned.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n?([\s\S]*)$/);
    const frontmatter = fmMatch ? fmMatch[1] : '';
    const body = fmMatch ? fmMatch[2] : cleaned;
    const parts = [];
    for (const key of ['must_haves', 'truths', 'objective']) {
        const block = extractYamlBlock(frontmatter, key);
        if (block)
            parts.push(block);
    }
    // Replace hand-rolled split(/\r?\n/) + heading walk with the seam's collectSections.
    // stopPredicate fires on EVERY heading (collectSections needs to start a section at
    // each heading), then we filter to designated ones — same semantics as the prior
    // inDesignated flag: emit the heading line + body only when DESIGNATED_HEADINGS_RE matches.
    const sections = (0, markdown_sectionizer_cjs_1.collectSections)(body, () => true);
    const bodyParts = [];
    for (const section of sections) {
        const headingLine = '#'.repeat(section.heading.level) + ' ' + section.heading.text;
        if (DESIGNATED_HEADINGS_RE.test(headingLine)) {
            bodyParts.push(headingLine);
            if (section.body)
                bodyParts.push(section.body);
        }
    }
    parts.push(bodyParts.join('\n'));
    parts.push(extractXmlTagBodies(cleaned));
    return parts.join('\n\n');
}
function buildPlanMessage(uncovered) {
    if (uncovered.length === 0)
        return 'All trackable CONTEXT.md decisions are covered by plans.';
    return [
        '## Decision Coverage Gap',
        '',
        `${uncovered.length} CONTEXT.md decision(s) are not covered by any plan:`,
        '',
        ...uncovered.map((item) => `- **${item.id}** (${item.category || 'uncategorized'}): ${item.text}`),
        '',
        'Resolve by citing `D-NN:` in any of the scanned plan surfaces: front-matter',
        '`must_haves`/`truths`/`objective`, a `## must_haves`/`truths`/`tasks`/`objective`',
        'heading, or an `<objective>`/`<tasks>`/`<task>`/`<action>`/`<read_first>`/`<behavior>`/`<verify>`/`<acceptance_criteria>`/`<done>`',
        'tag body. Other locations (prose outside those headings, comments, other XML tags) are not scanned.',
        'OR move the decision to `### Claude\'s Discretion` / tag it `[informational]` if it should not be tracked.',
    ].join('\n');
}
function buildVerifyMessage(notHonored) {
    if (notHonored.length === 0)
        return 'All trackable CONTEXT.md decisions are honored by shipped artifacts.';
    return [
        '### Decision Coverage (warning)',
        '',
        `${notHonored.length} decision(s) not found in shipped artifacts:`,
        '',
        ...notHonored.map((item) => `- **${item.id}** (${item.category || 'uncategorized'}): ${item.text}`),
        '',
        'This is a soft warning - verification status is unchanged.',
    ].join('\n');
}
function loadDecisionExtraction(contextPath) {
    const extraction = (0, decisions_cjs_1.extractDecisions)(readIfExists(contextPath));
    return {
        trackable: extraction.decisions.filter((d) => d.trackable),
        outcome: extraction.outcome,
    };
}
function cmdDecisionCoveragePlan(projectDir, args, raw) {
    const phaseDir = args[2] ? resolvePath(args[2], projectDir) : '';
    const contextArg = args[3];
    const contextPath = contextArg ? resolvePath(contextArg, projectDir) : '';
    if (!gateEnabled(projectDir)) {
        output({ passed: true, skipped: true, reason: 'workflow.context_coverage_gate is false', total: 0, covered: 0, uncovered: [], message: 'Decision coverage gate disabled by config.' }, raw, undefined);
        return;
    }
    // #2770: an EMPTY/MISSING contextPath argument is a CALLER ERROR (the workflow
    // forgot to pass the path — e.g. a shell variable lost between Bash blocks), not
    // evidence the phase has no CONTEXT.md. Fail closed (mirrors #1365 fail-loud) so a
    // blocking gate cannot silently certify success on a caller mistake.
    if (!contextArg || contextArg === '') {
        output({ passed: false, skipped: false, reason: 'missing context path argument', total: 0, covered: 0, uncovered: [], message: 'Decision coverage gate called without a context path argument — the caller (e.g. the plan-phase workflow) must pass the CONTEXT.md path. An empty argument is a caller error, not evidence there is nothing to check (#2770).' }, raw, undefined);
        return;
    }
    // A REAL path whose file genuinely does not exist is the LEGITIMATE green skip.
    if (!node_fs_1.default.existsSync(contextPath)) {
        output({ passed: true, skipped: true, reason: 'CONTEXT.md missing', total: 0, covered: 0, uncovered: [], message: 'No CONTEXT.md - nothing to check.' }, raw, undefined);
        return;
    }
    const { trackable: decisions, outcome } = loadDecisionExtraction(contextPath);
    // #1365 fail-loud gate: any could-not-parse outcome must NOT silently pass —
    // even when some decisions were extracted (e.g. D-01 valid but D-02 malformed).
    // A parse-miss on ANY bullet means the gate cannot certify full coverage.
    // Fire independent of decisions.length so a partial-parse still blocks.
    if (outcome === 'could-not-parse') {
        const partialParse = decisions.length > 0;
        output({
            passed: false,
            skipped: false,
            reason: 'could-not-parse',
            total: decisions.length,
            covered: 0,
            uncovered: [],
            message: partialParse
                ? 'Decision coverage gate: decisions could not be fully parsed — one or more ' +
                    '`- **D-NN ...**` bullets appear malformed (missing `:` or ` — ` separator). ' +
                    'Fix the bullet format so all D-NN decisions can be read before re-running the gate.'
                : 'Decision coverage gate: could not parse decisions — possible format mismatch. ' +
                    'The CONTEXT.md appears to be decision-shaped (has a <decisions> block, a decisions heading, ' +
                    'or D- tokens) but no D-NN bullets could be extracted. Check the formatting of the decisions ' +
                    'block and ensure bullets follow the `- **D-NN:** text` or `- **D-NN — title** body` form.',
        }, raw, undefined);
        return;
    }
    if (decisions.length === 0) {
        output({ passed: true, skipped: true, reason: 'no trackable decisions', total: 0, covered: 0, uncovered: [], message: 'No trackable decisions in CONTEXT.md.' }, raw, undefined);
        return;
    }
    const sections = loadPlanContents(phaseDir).map(extractPlanDesignatedSections);
    const uncovered = [];
    let covered = 0;
    for (const decision of decisions) {
        if (sections.some((section) => decisionMentioned(section, decision)))
            covered++;
        else
            uncovered.push({ id: decision.id, text: decision.text, category: decision.category });
    }
    output({
        passed: uncovered.length === 0,
        skipped: false,
        total: decisions.length,
        covered,
        uncovered,
        message: buildPlanMessage(uncovered),
    }, raw, undefined);
}
function recentCommitMessages(projectDir) {
    try {
        return (0, node_child_process_1.execFileSync)('git', ['log', '-n', '200', '--pretty=%s%n%b'], {
            cwd: projectDir,
            encoding: 'utf-8',
            maxBuffer: 4 * 1024 * 1024,
            windowsHide: true,
            timeout: 15_000,
        });
    }
    catch {
        return '';
    }
}
function isInsideRoot(candidatePath, rootDir) {
    const root = node_path_1.default.resolve(rootDir);
    const target = node_path_1.default.resolve(root, candidatePath);
    return target === root || target.startsWith(`${root}${node_path_1.default.sep}`);
}
function readModifiedFilesContent(projectDir, summaries) {
    const out = [];
    let total = 0;
    for (const summary of summaries) {
        if (!summary)
            continue;
        for (const blockMatch of summary.matchAll(/files_modified:\s*\n((?:[ \t]*-\s+.+\n?)+)/g)) {
            const files = [...(blockMatch[1] || '').matchAll(/-\s+(.+)/g)]
                .map((match) => match[1].trim().replace(/^["']|["']$/g, ''));
            for (const file of files) {
                if (total >= 50)
                    break;
                if (!file || !isInsideRoot(file, projectDir))
                    continue;
                const raw = readIfExists(resolvePath(file, projectDir));
                out.push(raw.length > 256 * 1024 ? raw.slice(0, 256 * 1024) : raw);
                total++;
            }
            if (total >= 50)
                break;
        }
        if (total >= 50)
            break;
    }
    return out.join('\n\n');
}
function cmdDecisionCoverageVerify(projectDir, args, raw) {
    const phaseDir = args[2] ? resolvePath(args[2], projectDir) : '';
    const contextPath = args[3] ? resolvePath(args[3], projectDir) : '';
    if (!gateEnabled(projectDir)) {
        output({ skipped: true, blocking: false, reason: 'workflow.context_coverage_gate is false', total: 0, honored: 0, not_honored: [], message: 'Decision coverage gate disabled by config.' }, raw, undefined);
        return;
    }
    if (!contextPath || !node_fs_1.default.existsSync(contextPath)) {
        output({ skipped: true, blocking: false, reason: 'CONTEXT.md missing', total: 0, honored: 0, not_honored: [], message: 'No CONTEXT.md - nothing to check.' }, raw, undefined);
        return;
    }
    const { trackable: decisions, outcome: decisionOutcome } = loadDecisionExtraction(contextPath);
    // Mirror could-not-parse surface for verify (non-blocking advisory WARN).
    // Fire independent of decisions.length — a parse-miss on any bullet must surface,
    // even when some decisions were partially extracted (#1365 fix-parity with plan gate).
    if (decisionOutcome === 'could-not-parse') {
        const partialParse = decisions.length > 0;
        output({
            skipped: false,
            blocking: false,
            reason: 'could-not-parse',
            total: decisions.length,
            honored: 0,
            not_honored: [],
            message: partialParse
                ? 'Decision coverage verify (warning): decisions could not be fully parsed — one or more ' +
                    '`- **D-NN ...**` bullets appear malformed. Fix the bullet format in the CONTEXT.md decisions block.'
                : 'Decision coverage verify (warning): could not parse decisions — possible format mismatch. ' +
                    'Check the formatting of the CONTEXT.md decisions block.',
        }, raw, undefined);
        return;
    }
    if (decisions.length === 0) {
        output({ skipped: true, blocking: false, reason: 'no trackable decisions', total: 0, honored: 0, not_honored: [], message: 'No trackable decisions in CONTEXT.md.' }, raw, undefined);
        return;
    }
    const planContents = loadPlanContents(phaseDir);
    // #3183 (lint-plan-count-drift): same single-owner sourcing as
    // loadPlanContents above — scanPhasePlans's summaryFiles instead of a
    // local `-SUMMARY.md` readdirSync filter.
    const summaryParts = node_fs_1.default.existsSync(phaseDir)
        ? scanPhasePlans(phaseDir).summaryFiles.map((entry) => readIfExists(node_path_1.default.join(phaseDir, entry)))
        : [];
    const haystack = [
        planContents.join('\n\n'),
        summaryParts.join('\n\n'),
        readModifiedFilesContent(projectDir, summaryParts),
        recentCommitMessages(projectDir),
    ].join('\n\n');
    const notHonored = [];
    let honored = 0;
    for (const decision of decisions) {
        if (decisionMentioned(haystack, decision))
            honored++;
        else
            notHonored.push({ id: decision.id, text: decision.text, category: decision.category });
    }
    output({
        skipped: false,
        blocking: false,
        total: decisions.length,
        honored,
        not_honored: notHonored,
        message: buildVerifyMessage(notHonored),
    }, raw, undefined);
}
// ─── ui-plan-gate ─────────────────────────────────────────────────────────────
/**
 * ui-plan-gate: given a phase number, checks whether the phase has frontend
 * indicators and whether a *-UI-SPEC.md already exists in the phase directory.
 *
 * Returns JSON: { frontend, hasFrontendEvidence, hasUiSpec, block, uiSpecPath, matchedToken, matchedLine }
 *   block = frontend && hasFrontendEvidence && !hasUiSpec (#3312: gate fires when
 *   UI work is detected AND the repo has static frontend evidence but no spec exists)
 *
 * Invocable as: gsd_run check ui-plan-gate <phase>
 *
 * Uses checkUiPresence from ui-safety-gate.cjs — does NOT reimplement frontend detection.
 * Uses getRoadmapPhaseWithFallback + findPhaseInternal from leaf modules for phase data.
 */
function findUiSpecInDir(phaseDir) {
    if (!phaseDir || !node_fs_1.default.existsSync(phaseDir))
        return '';
    try {
        const files = node_fs_1.default.readdirSync(phaseDir);
        const found = files.find((f) => /-UI-SPEC\.md$/.test(f));
        return found ? node_path_1.default.join(phaseDir, found) : '';
    }
    catch {
        return '';
    }
}
/**
 * Pure logic for ui-plan-gate — exposed for direct behavioral testing.
 *
 * Given a projectDir and phase number:
 *   (a) Reads the phase section from ROADMAP.md via getRoadmapPhaseWithFallback —
 *       same two-pass lookup (current milestone → full roadmap) as `roadmap.get-phase`
 *       (cmdRoadmapGetPhase). Cross-milestone / older frontend phases resolve correctly.
 *       If ROADMAP.md is missing, phaseSection is '' (ROADMAP.md not present = project
 *       has no roadmap = cannot be frontend). If the phase truly can't be found after
 *       both passes, phaseSection is '' and phaseLookupFailed is set so callers can
 *       surface the miss — we do NOT silently degrade to frontend:false if the roadmap
 *       exists but the phase header is absent.
 *   (b) Runs checkUiPresence (frontend detection) — no reimplementation.
 *   (c) Resolves the phase directory via findPhaseInternal (phase-locator.cjs); checks for *-UI-SPEC.md.
 *
 * Returns: { frontend, hasFrontendEvidence, hasUiSpec, block, uiSpecPath, matchedToken, matchedLine, phaseLookupFailed }
 *   block = frontend && hasFrontendEvidence && !hasUiSpec   (#3312)
 *   phaseLookupFailed = ROADMAP.md present but phase header not found (surfaced for
 *                       onError:halt gates so a missing phase doesn't silently bypass)
 *
 * #3312 — structural corroboration: `frontend` is a vocabulary signal only. A
 * hyphen is a word boundary, so a phase naming the repo `dashboard-financeiro`
 * matches the token `dashboard` exactly like the real compound `micro-frontend`
 * (the boundary rule of #3718 is intentional and untouched). The gate therefore
 * blocks only when the token match is corroborated by static frontend evidence
 * in the repo tree (hasStaticFrontendEvidence: package.json UI-framework dep or
 * a component-framework file). This mirrors the sibling post-wave gate
 * computeUiSafetyGate, which requires `hasUiFiles` (git diff) before blocking.
 * matchedToken/matchedLine surface what tripped the sniffer so an operator can
 * judge the flag in one second instead of reaching for --skip-ui.
 */
function computeUiPlanGate(projectDir, phase) {
    // (a) Read the phase section text using the same two-pass lookup as roadmap.get-phase.
    // getRoadmapPhaseWithFallback: current-milestone first, then stripShippedMilestones
    // fallback — mirrors cmdRoadmapGetPhase exactly.
    let phaseSection = '';
    let phaseLookupFailed;
    try {
        const section = getRoadmapPhaseWithFallback(projectDir, phase);
        if (section === null) {
            // Distinguish: ROADMAP.md missing (no-roadmap project) vs phase not found in ROADMAP.
            // planningDir(cwd) resolves the .planning/ root for workstream-aware paths.
            const planDir = planningDir(projectDir);
            const roadmapPath = node_path_1.default.join(planDir, 'ROADMAP.md');
            if (node_fs_1.default.existsSync(roadmapPath)) {
                // ROADMAP.md exists but phase was not found → surface the miss
                phaseLookupFailed = true;
            }
            // phaseSection stays ''
        }
        else {
            phaseSection = section;
        }
    }
    catch { /* roadmap read failure → treat as empty (non-frontend) */ }
    // (b) Run checkUiPresence (frontend detection) — reuse existing helper; no reimplementation
    const presenceResult = (0, ui_safety_gate_cjs_1.checkUiPresence)(phaseSection);
    const frontend = presenceResult.hasUI;
    // (b') #3312 — static structural corroboration. Only probed when the sniffer
    // matched (evidence is irrelevant otherwise); failures degrade to false.
    const hasFrontendEvidence = frontend ? (0, ui_frontend_evidence_cjs_1.hasStaticFrontendEvidence)(projectDir) : false;
    // (c) Resolve phase directory via findPhaseInternal and check for *-UI-SPEC.md
    let phaseDir = '';
    try {
        const result = findPhaseInternal(projectDir, phase);
        if (result && typeof result === 'object') {
            // findPhaseInternal returns { directory: '<relative-posix-path>', ... }
            // directory is relative to cwd — resolve it to absolute.
            const relDir = typeof result['directory'] === 'string' ? result['directory'] : '';
            if (relDir) {
                phaseDir = node_path_1.default.resolve(projectDir, relDir);
            }
        }
        else if (typeof result === 'string') {
            phaseDir = result;
        }
    }
    catch { /* phase dir lookup failure → hasUiSpec=false */ }
    const uiSpecPath = findUiSpecInDir(phaseDir);
    const hasUiSpec = uiSpecPath !== '';
    // block = frontend phase with structural frontend evidence and no UI-SPEC (#3312)
    const block = frontend && hasFrontendEvidence && !hasUiSpec;
    const result = {
        frontend, hasFrontendEvidence, hasUiSpec, block,
        uiSpecPath: hasUiSpec ? uiSpecPath : null,
        matchedToken: presenceResult.matchedToken,
        matchedLine: presenceResult.matchedLine,
    };
    if (phaseLookupFailed)
        result.phaseLookupFailed = true;
    return result;
}
function cmdUiPlanGate(projectDir, args, raw) {
    // args[0] = 'check', args[1] = 'ui-plan-gate', args[2] = phase
    const phase = args[2] || '';
    if (!phase) {
        error('ui-plan-gate requires a phase argument: check ui-plan-gate <phase>', ERROR_REASON.SDK_MISSING_ARG);
        return;
    }
    output(computeUiPlanGate(projectDir, phase), raw, undefined);
}
// ─── ui-safety-gate ───────────────────────────────────────────────────────────
/**
 * ui-safety-gate: post-wave check that verifies UI-changed files conform to
 * the active UI-SPEC for the phase. Called after each wave by execute:wave:post.
 *
 * Returns JSON: { frontend: boolean, hasUiFiles: boolean, hasUiSpec: boolean, block: boolean, message?: string }
 *   block = frontend && hasUiFiles && !hasUiSpec
 *
 * Args: check ui-safety-gate <phase>
 * Invocable as: gsd_run check ui-safety-gate <phase>
 *             or gsd_run check ui.safety-gate <phase> (dots normalized to hyphens)
 *
 * Uses checkUiPresence from ui-safety-gate.cjs — does NOT reimplement frontend detection.
 * Checks whether any files changed in recent git history match frontend file patterns.
 * Also checks whether a *-UI-SPEC.md exists in the phase directory (same as ui-plan-gate).
 *
 * Limitation: uses git diff HEAD~1..HEAD which covers only the last commit; in a
 * multi-plan wave the wave-start commit would be more accurate but is not yet stored
 * in the wave manifest. This is tracked as a known limitation.
 */
const UI_FILE_EXTENSIONS_RE = /\.(tsx|jsx|css|scss|sass|less|vue|svelte|html)$/i;
const UI_PATH_PATTERNS_RE = /\/(components|pages|views|screens|layouts|ui|frontend)\//i;
/**
 * Pure logic for ui-safety-gate — exposed for direct behavioral testing.
 *
 * Given a projectDir and phase number:
 *   (a) Reads the phase section from ROADMAP.md via getRoadmapPhaseWithFallback —
 *       same lookup as computeUiPlanGate — to determine if this is a frontend phase.
 *   (b) Runs checkUiPresence (frontend detection) — no reimplementation.
 *   (c) Checks git diff HEAD~1..HEAD for UI file changes in the current worktree.
 *   (d) Resolves the phase directory via findPhaseInternal (phase-locator.cjs); checks for *-UI-SPEC.md.
 *
 * Returns: { frontend, hasUiFiles, hasUiSpec, block, message?, phaseLookupFailed? }
 *   block = frontend && hasUiFiles && !hasUiSpec
 *   phaseLookupFailed = ROADMAP.md present but phase header not found
 */
function computeUiSafetyGate(projectDir, phase) {
    // (a) Read the phase section text (same two-pass lookup as computeUiPlanGate)
    let phaseSection = '';
    let phaseLookupFailed;
    try {
        const section = getRoadmapPhaseWithFallback(projectDir, phase);
        if (section === null) {
            const planDir = planningDir(projectDir);
            const roadmapPath = node_path_1.default.join(planDir, 'ROADMAP.md');
            if (node_fs_1.default.existsSync(roadmapPath)) {
                phaseLookupFailed = true;
            }
        }
        else {
            phaseSection = section;
        }
    }
    catch { /* roadmap read failure → treat as empty (non-frontend) */ }
    // (b) Run checkUiPresence (frontend detection) — reuse existing helper; no reimplementation
    const presenceResult = (0, ui_safety_gate_cjs_1.checkUiPresence)(phaseSection);
    const frontend = presenceResult.hasUI;
    // (c) Check whether any UI files were changed in recent git commits
    // Uses git diff HEAD~1..HEAD to detect frontend file changes since last commit.
    // Known limitation: multi-plan waves may need the wave-start commit for full coverage.
    let hasUiFiles = false;
    try {
        const changed = (0, node_child_process_1.execFileSync)('git', ['diff', '--name-only', 'HEAD~1', 'HEAD'], {
            cwd: projectDir,
            encoding: 'utf-8',
            maxBuffer: 2 * 1024 * 1024,
            windowsHide: true,
            timeout: 10_000,
        });
        hasUiFiles = changed.split('\n').some((f) => f.trim() && (UI_FILE_EXTENSIONS_RE.test(f) || UI_PATH_PATTERNS_RE.test(f)));
    }
    catch { /* git unavailable or no prior commit — treat as no UI files changed */ }
    // (d) Resolve phase directory and check for *-UI-SPEC.md (same as computeUiPlanGate)
    let phaseDir = '';
    try {
        const result = findPhaseInternal(projectDir, phase);
        if (result && typeof result === 'object') {
            const relDir = typeof result['directory'] === 'string' ? result['directory'] : '';
            if (relDir) {
                phaseDir = node_path_1.default.resolve(projectDir, relDir);
            }
        }
        else if (typeof result === 'string') {
            phaseDir = result;
        }
    }
    catch { /* phase dir lookup failure → hasUiSpec=false */ }
    const uiSpecPath = findUiSpecInDir(phaseDir);
    const hasUiSpec = uiSpecPath !== '';
    // block only when: this is a frontend phase AND UI files were changed AND no UI-SPEC exists
    const block = frontend && hasUiFiles && !hasUiSpec;
    const result = { frontend, hasUiFiles, hasUiSpec, block };
    if (block) {
        result.message = `UI files changed in this wave but no UI-SPEC.md exists for Phase ${phase}. ` +
            `Run /gsd:ui-phase ${phase} to generate the design contract before continuing.`;
    }
    if (phaseLookupFailed)
        result.phaseLookupFailed = true;
    return result;
}
function cmdUiSafetyGate(projectDir, args, raw) {
    // args[0] = 'check', args[1] = 'ui-safety-gate', args[2] = phase
    const phase = args[2] || '';
    if (!phase) {
        error('ui-safety-gate requires a phase argument: check ui-safety-gate <phase>', ERROR_REASON.SDK_MISSING_ARG);
        return;
    }
    output(computeUiSafetyGate(projectDir, phase), raw, undefined);
}
function cmdTddReviewCheckpoint(projectDir, args, raw) {
    // args[0] = 'check', args[1] = 'tdd-review-checkpoint' (normalized), args[2] = phase
    const phase = args[2] || '';
    if (!phase) {
        error('tdd.review-checkpoint requires a phase argument: check tdd.review-checkpoint <phase>', ERROR_REASON.SDK_MISSING_ARG);
        return;
    }
    // Resolve phase directory
    let phaseDir = '';
    try {
        const result = findPhaseInternal(projectDir, phase);
        if (result && typeof result === 'object') {
            const relDir = typeof result['directory'] === 'string' ? result['directory'] : '';
            if (relDir)
                phaseDir = node_path_1.default.resolve(projectDir, relDir);
        }
        else if (typeof result === 'string') {
            phaseDir = result;
        }
    }
    catch { /* phase dir lookup failure */ }
    // Find all PLAN.md files with type: tdd in frontmatter
    const tddPlanFiles = [];
    if (phaseDir) {
        try {
            // #3183: canonical plan set (root+nested, superseded-excluded) from the
            // single owner, rather than a root-only hand-rolled readdirSync filter.
            const files = scanPhasePlans(phaseDir).planFiles;
            for (const file of files) {
                const planPath = node_path_1.default.join(phaseDir, file);
                const content = readIfExists(planPath);
                // Check frontmatter for type: tdd
                // CRLF-tolerant: a PLAN.md written with Windows line endings (---\r\n...---)
                // must still match. The same CRLF-tolerant form is already used at line 205
                // (extractPlanDesignatedSections); this is the same canonical pattern, applied
                // here for the tdd-classification path. Fixes #2449.
                const frontmatterMatch = content.match(/^---\r?\n([\s\S]*?)\r?\n---/);
                if (frontmatterMatch) {
                    const fm = frontmatterMatch[1];
                    if (/^type:\s*tdd\s*$/m.test(fm)) {
                        tddPlanFiles.push(planPath);
                    }
                }
            }
        }
        catch { /* directory read failure */ }
    }
    if (tddPlanFiles.length === 0) {
        const result = {
            // Uniform gate contract: block = violations > 0 (advisory; never truly blocks).
            block: false,
            passed: true,
            tddPlans: 0,
            violations: 0,
            table: '',
            rows: [],
            message: `No type:tdd plans found in phase ${phase}. TDD review skipped.`,
        };
        // Pass undefined as rawValue so --raw emits JSON (not plain text).
        // The human-readable report is carried in `result.message` for the
        // dispatch's advisory branch to surface.
        output(result, raw, undefined);
        return;
    }
    // For each TDD plan, extract the plan ID (padded plan number) and check git log
    const rows = [];
    for (const planPath of tddPlanFiles) {
        // Extract plan ID from filename (e.g. "01-02-PLAN.md" → "01-02", or "03-PLAN.md" → "03")
        const basename = node_path_1.default.basename(planPath, '-PLAN.md');
        // planId for commit grep: phase-plan format, e.g. "01-02"
        const planId = basename;
        // Check for RED gate commit: test({planId}):
        let red = false;
        let green = false;
        let refactor = false;
        try {
            const redCommit = (0, node_child_process_1.execFileSync)('git', ['log', '--oneline', `--grep=^test(${planId}):`, '--', '.'], { cwd: projectDir, encoding: 'utf-8', maxBuffer: 1024 * 1024, windowsHide: true, timeout: 10_000 });
            red = redCommit.trim().length > 0;
        }
        catch { /* git unavailable or no match */ }
        try {
            const greenCommit = (0, node_child_process_1.execFileSync)('git', ['log', '--oneline', `--grep=^feat(${planId}):`, '--', '.'], { cwd: projectDir, encoding: 'utf-8', maxBuffer: 1024 * 1024, windowsHide: true, timeout: 10_000 });
            green = greenCommit.trim().length > 0;
        }
        catch { /* git unavailable or no match */ }
        try {
            const refactorCommit = (0, node_child_process_1.execFileSync)('git', ['log', '--oneline', `--grep=^refactor(${planId}):`, '--', '.'], { cwd: projectDir, encoding: 'utf-8', maxBuffer: 1024 * 1024, windowsHide: true, timeout: 10_000 });
            refactor = refactorCommit.trim().length > 0;
        }
        catch { /* git unavailable or no match */ }
        const missing = [];
        if (!red)
            missing.push('RED');
        if (!green)
            missing.push('GREEN');
        const status = missing.length === 0 ? 'Pass' : 'FAIL';
        rows.push({ planId, red, green, refactor, status, missing });
    }
    const violations = rows.filter(r => r.status === 'FAIL').length;
    // Build review table
    const tableHeader = '| Plan | RED | GREEN | REFACTOR | Status |';
    const tableDivider = '|------|-----|-------|----------|--------|';
    const tableRows = rows.map(r => `| ${r.planId.padEnd(4)} | ${r.red ? ' ✓ ' : ' ✗ '} | ${r.green ? '  ✓  ' : '  ✗  '} | ${r.refactor ? '   ✓    ' : '   —    '} | ${r.status.padEnd(6)} |`);
    let table = [
        `### TDD REVIEW — Phase ${phase}`,
        '',
        `TDD Plans: ${tddPlanFiles.length} | Gate violations: ${violations}`,
        '',
        tableHeader,
        tableDivider,
        ...tableRows,
    ].join('\n');
    if (violations > 0) {
        table += '\n\n⚠ Gate violations are advisory — review before advancing.';
        for (const r of rows.filter(row => row.status === 'FAIL')) {
            table += `\n  Plan ${r.planId} missing: ${r.missing.join(', ')} gate commit(s).`;
            table += `\n  Expected commit pattern: test(${r.planId}): ... → feat(${r.planId}): ...`;
        }
    }
    const result = {
        // Uniform gate contract: block = violations > 0.
        // This gate is advisory (blocking: false in capability.json) so block:true
        // only surfaces as a warning, never halts. Kept here so the host-loop
        // dispatch can read a single consistent `block` field.
        block: violations > 0,
        passed: true,
        tddPlans: tddPlanFiles.length,
        violations,
        table,
        rows,
        // Human-readable report in `message` so the dispatch's advisory branch
        // can surface it. --raw emits JSON (rawValue=undefined), not plain text.
        message: table,
    };
    // Pass undefined as rawValue so --raw emits JSON (not the raw table text).
    // The review table is carried in `result.message` and `result.table` so
    // the host-loop dispatch's advisory branch can surface it.
    output(result, raw, undefined);
}
// ─── tdd-red-evidence (#3770) ──────────────────────────────────────────────────
/**
 * tdd-red-evidence: validates a persisted RED-phase test-run record for a
 * `type: tdd` plan (#3770). Only an INTENTIONAL failure of the target test
 * (verdict RED_EVIDENCE_OK) may authorize GREEN; zero-test discovery, fixture/
 * load crashes, nonzero exits without a failing test, unrelated failures, and
 * unexpected greens are INVALID_RED and block GREEN.
 *
 * The record is the JSON the executor persists after running the RED command:
 *   { command, exitCode, output, targetTest, targetFile?, expected?, actual? }
 * Fail-closed: a missing/unreadable/unparseable record is INVALID_RED
 * (reason unreadable_record), never a pass.
 *
 * Args: check tdd-red-evidence <record.json>
 */
function cmdTddRedEvidence(_projectDir, args, raw) {
    const recordPath = typeof args[2] === 'string' ? args[2] : '';
    if (!recordPath) {
        error('tdd-red-evidence requires a record path: check tdd-red-evidence <record.json>', ERROR_REASON.SDK_MISSING_ARG);
        return;
    }
    const resolved = node_path_1.default.resolve(recordPath);
    const text = readIfExists(resolved);
    const input = (() => {
        if (!text)
            return null;
        try {
            return (JSON.parse(text) ?? {});
        }
        catch {
            return null;
        }
    })();
    if (!input) {
        output({
            passed: false,
            block: true,
            verdict: 'INVALID_RED',
            reason: 'unreadable_record',
            record: resolved,
            readError: text ? `record is not valid JSON: ${resolved}` : `record not found or unreadable: ${resolved}`,
        }, raw, undefined);
        return;
    }
    const evidenceInput = {
        command: input['command'],
        exitCode: input['exitCode'],
        output: input['output'],
        targetTest: input['targetTest'],
        targetFile: input['targetFile'],
        expected: input['expected'],
        actual: input['actual'],
    };
    const result = (0, tdd_red_evidence_cjs_1.classifyRedEvidence)(evidenceInput);
    const record = (0, tdd_red_evidence_cjs_1.buildRedEvidenceRecord)(evidenceInput, result);
    output({
        // Uniform gate contract: block = !passed. INVALID_RED blocks GREEN.
        passed: result.verdict === 'RED_EVIDENCE_OK',
        block: result.verdict !== 'RED_EVIDENCE_OK',
        verdict: result.verdict,
        reason: result.reason,
        evidence: result.evidence,
        record,
        message: result.verdict === 'RED_EVIDENCE_OK'
            ? `RED evidence verified: target test "${result.evidence.target_test}" failed as expected (exit ${result.evidence.exit_code}). GREEN authorized.`
            : `INVALID_RED (${result.reason}): GREEN blocked. Fix the RED phase — only an intentional failure of target test "${result.evidence.target_test}" authorizes production edits.`,
    }, raw, undefined);
}
/**
 * Resolve a phase argument to an absolute phase directory, or '' when it
 * cannot be resolved. Shared by every `check` arm that probes a phase's
 * PLAN.md files, so the two never drift (DEFECT.GENERATIVE-FIX-DIVERGENCE).
 * Never throws — the callers emit a degraded JSON payload instead, because
 * a consumer must be able to tell "nothing to report" from "could not look".
 */
function resolvePhaseDirOrEmpty(projectDir, phase) {
    try {
        const result = findPhaseInternal(projectDir, phase);
        if (result && typeof result === 'object') {
            // findPhaseInternal returns { directory: '<relative-posix-path>', ... }
            // directory is relative to cwd — resolve it to absolute.
            const relDir = typeof result['directory'] === 'string' ? result['directory'] : '';
            if (relDir) {
                return node_path_1.default.resolve(projectDir, relDir);
            }
        }
        else if (typeof result === 'string') {
            return result;
        }
    }
    catch { /* phase dir lookup failure → caller emits degraded payload */ }
    return '';
}
// ─── verify-command-paths (#2401) ──────────────────────────────────────────────
/**
 * verify-command-paths: probes every `<automated>` verify command declared in a
 * phase's `-PLAN.md` files against the filesystem WITHOUT executing anything —
 * see verify-command-grounding.cjs for the recognizer contract.
 *
 * Args: check verify-command-paths <phase>
 * Invocable as: gsd_run check verify-command-paths <phase>
 *
 * When the phase cannot be resolved to a directory, this emits a non-throwing
 * degraded JSON payload (status/commands/counts all zeroed, `readError`
 * populated) rather than calling `error()` — the plan-checker parses this
 * result and must be able to distinguish "nothing to report" from "could not
 * look", which a non-zero exit / thrown error would collapse.
 */
function cmdVerifyCommandPaths(projectDir, args, raw) {
    // args[0] = 'check', args[1] = 'verify-command-paths', args[2] = phase
    const phase = args[2] || '';
    if (!phase) {
        output({
            status: 'unresolvable',
            commands: [],
            counts: { blocker: 0, warning: 0, total: 0 },
            readError: 'verify-command-paths requires a phase argument: check verify-command-paths <phase>',
        }, raw, undefined);
        return;
    }
    const phaseDir = resolvePhaseDirOrEmpty(projectDir, phase);
    if (!phaseDir) {
        output({
            status: 'unresolvable',
            commands: [],
            counts: { blocker: 0, warning: 0, total: 0 },
            readError: `could not resolve phase directory for phase ${phase}`,
        }, raw, undefined);
        return;
    }
    const probed = probePhaseVerifyCommands({ phaseDir, projectRoot: projectDir });
    output(probed, raw, undefined);
}
// ─── verify-failure-directions (#3172) ─────────────────────────────────────────
/**
 * verify-failure-directions: probes every `<automated>` verify command
 * declared in a phase's `-PLAN.md` files for a stated `<fails_when>` failing
 * direction — see verify-command-grounding.cjs for the recognizer contract.
 *
 * Args: check verify-failure-directions <phase>
 * Invocable as: gsd_run check verify-failure-directions <phase>
 *
 * When the phase cannot be resolved to a directory, this emits a non-throwing
 * degraded JSON payload (status/commands/counts all zeroed, `readError`
 * populated) rather than calling `error()` — the plan-checker parses this
 * result and must be able to distinguish "nothing to report" from "could not
 * look", which a non-zero exit / thrown error would collapse.
 */
function cmdVerifyFailureDirections(projectDir, args, raw) {
    // args[0] = 'check', args[1] = 'verify-failure-directions', args[2] = phase
    const phase = args[2] || '';
    if (!phase) {
        output({
            status: 'unresolvable',
            commands: [],
            counts: { blocker: 0, warning: 0, total: 0 },
            readError: 'verify-failure-directions requires a phase argument: check verify-failure-directions <phase>',
        }, raw, undefined);
        return;
    }
    const phaseDir = resolvePhaseDirOrEmpty(projectDir, phase);
    if (!phaseDir) {
        output({
            status: 'unresolvable',
            commands: [],
            counts: { blocker: 0, warning: 0, total: 0 },
            readError: `could not resolve phase directory for phase ${phase}`,
        }, raw, undefined);
        return;
    }
    const result = probePhaseFailingDirections({ phaseDir });
    output(result, raw, undefined);
}
// ─── gap-analysis-plan-post ───────────────────────────────────────────────────
/**
 * gap-analysis-plan-post: non-blocking advisory check that runs the post-planning
 * gap analysis after all PLAN.md files are generated for a phase.
 *
 * Cross-references every REQ-ID and D-ID from REQUIREMENTS.md and CONTEXT.md
 * against the concatenated text of all *-PLAN.md files, emitting a coverage table.
 *
 * This gate is always advisory (passed: true) — it never blocks phase advancement.
 *
 * Args: check gap-analysis.plan-post <phase-dir> [phase-req-ids]
 * Invocable as: gsd_run check gap-analysis.plan-post <phase-dir> [phase-req-ids]
 */
function cmdGapAnalysisPlanPost(projectDir, args, raw) {
    // args[0] = 'check', args[1] = 'gap-analysis-plan-post' (normalized), args[2] = phaseDir, args[3] = phaseReqIds
    const phaseDir = args[2] || '';
    if (!phaseDir) {
        error('gap-analysis.plan-post requires a phase-dir argument: check gap-analysis.plan-post <phase-dir> [phase-req-ids]', ERROR_REASON.SDK_MISSING_ARG);
        return;
    }
    const phaseReqIds = args[3] ?? undefined;
    const result = runGapAnalysis(projectDir, phaseDir, { phaseReqIds });
    // Uniform gate contract: block = false (gap-analysis is always advisory, never blocks).
    // `message` carries the human-readable gap analysis report so the dispatch's
    // advisory branch can surface it. --raw emits JSON (rawValue=undefined), not
    // plain markdown text.
    output({
        block: false,
        passed: true,
        enabled: result.enabled,
        table: result.table,
        summary: result.summary,
        counts: result.counts,
        // Human-readable report in `message` for the host-loop advisory branch.
        message: result.table || result.summary || '',
    }, raw, undefined);
}
// ─── predicate (generic gate-predicate evaluator, #2008) ──────────────────────
/**
 * Production subprocess binding for the gate-predicate evaluator. Wraps the
 * bounded `execTool` seam (shell-command-projection) as a `runBoundedShell`
 * the pure evaluator consumes. `sh -c` runs the interpolated command; the
 * subprocess inherits the process env and is killed (SIGTERM) on timeout.
 *
 * `timedOut` is derived from the kill signal: spawnSync sets `signal: 'SIGTERM'`
 * when the `timeout` fires, distinct from a normal non-zero exit code. A command
 * that self-terminates with SIGTERM is indistinguishable at this seam and is
 * reported as a timeout — either way the gate blocks (non-zero), so the outcome
 * is fail-closed and correct. See ADR-2008.
 */
function buildPredicateDeps() {
    return {
        runBoundedShell(opts) {
            const r = (0, shell_command_projection_cjs_1.execTool)('sh', ['-c', opts.command], { cwd: opts.cwd, timeout: opts.timeoutMs });
            return {
                exitCode: r.exitCode,
                stdout: r.stdout,
                stderr: r.stderr,
                signal: r.signal,
                timedOut: r.timedOut,
            };
        },
        findPhaseArtifact(phaseDir, artifactSuffix) {
            if (!node_fs_1.default.existsSync(phaseDir))
                return null;
            if (artifactSuffix === '.' ||
                artifactSuffix === '..' ||
                artifactSuffix.includes('\0') ||
                node_path_1.default.basename(artifactSuffix) !== artifactSuffix ||
                node_path_1.default.win32.basename(artifactSuffix) !== artifactSuffix) {
                return null;
            }
            const directPath = (0, security_cjs_1.validatePath)(artifactSuffix, phaseDir);
            if (directPath.safe && node_fs_1.default.existsSync(directPath.resolved) && node_fs_1.default.statSync(directPath.resolved).isFile()) {
                return directPath.resolved;
            }
            const planningPath = (0, security_cjs_1.validatePath)(node_path_1.default.join('.planning', artifactSuffix), phaseDir);
            if (planningPath.safe && node_fs_1.default.existsSync(planningPath.resolved) && node_fs_1.default.statSync(planningPath.resolved).isFile()) {
                return planningPath.resolved;
            }
            try {
                const files = node_fs_1.default.readdirSync(phaseDir);
                for (const f of files) {
                    if (f.endsWith('-' + artifactSuffix) || f === artifactSuffix) {
                        const candidate = (0, security_cjs_1.validatePath)(f, phaseDir);
                        if (candidate.safe && node_fs_1.default.statSync(candidate.resolved).isFile())
                            return candidate.resolved;
                    }
                }
            }
            catch { /* ignore */ }
            return null;
        },
        readFrontmatter(filePath) {
            const content = (0, shell_command_projection_cjs_1.platformReadSync)(filePath);
            if (content === null)
                throw new Error(`predicate artifact disappeared before it could be read: ${filePath}`);
            const parsed = extractFrontmatter(content, filePath);
            return parsed;
        }
    };
}
/** Parse `--flag value` pairs from an args array into a map (last write wins). */
function parsePredicateFlags(args) {
    const out = {};
    for (let i = 0; i < args.length; i++) {
        const a = args[i];
        if (typeof a !== 'string')
            continue;
        if (!a.startsWith('--'))
            continue;
        const key = a.slice(2);
        const next = args[i + 1];
        if (key.length > 0 && typeof next === 'string' && !next.startsWith('--')) {
            out[key] = next;
            i++;
        }
    }
    return out;
}
/**
 * `check predicate` — generic evaluator for capability gate `check.predicate`
 * blocks (#2008). The workflow gate-dispatch invokes this for any gate whose
 * `check` carries a `predicate` (instead of a `query`); the predicate object is
 * passed as `--predicate '<json>'`. Emits the standard `{ block, message,
 * details? }` gate contract on success. A malformed predicate / unknown kind
 * THROWS inside the evaluator and is mapped here to `error()` (non-zero exit),
 * which the workflow's two-step gate contract treats as a step-1 command failure
 * routed per the gate's `onError`.
 *
 * Invocation:
 *   gsd_run check predicate --predicate '<json>' \
 *     [--phase-dir <dir>] [--phase-number <n>] [--phase-req-ids <ids>] --raw
 *
 * The subprocess runs at the runtime project root (the `cwd` passed to this
 * router), inheriting the process env. Interpolation placeholders
 * ${PHASE_NUMBER}/${PHASE_DIR}/${PHASE_REQ_IDS} are substituted from the flags.
 */
function cmdCheckPredicate(projectDir, args, raw) {
    const flags = parsePredicateFlags(args);
    const predicateJson = flags['predicate'];
    if (!predicateJson) {
        error('predicate requires --predicate <json> (the gate hook check.predicate object)', ERROR_REASON.SDK_MISSING_ARG);
        return;
    }
    let predicate;
    try {
        predicate = JSON.parse(predicateJson);
    }
    catch {
        error('predicate --predicate value must be valid JSON', ERROR_REASON.USAGE);
        return;
    }
    const ctx = {
        cwd: projectDir,
        phaseNumber: flags['phase-number'],
        phaseDir: flags['phase-dir'],
        phaseReqIds: flags['phase-req-ids'],
    };
    let result;
    try {
        result = evaluatePredicate(predicate, ctx, buildPredicateDeps());
    }
    catch (e) {
        error(`gate predicate evaluation failed: ${e.message}`, ERROR_REASON.USAGE);
        return;
    }
    output(result, raw, undefined);
}
// ─── api-coverage-verify-pre ──────────────────────────────────────────────────
/**
 * api-coverage.verify-pre: BLOCKING seal-time gate for the ai-integration
 * capability (#1562). Enforces "Full API Coverage by Default — Opt Out, Never
 * Opt In." A phase that integrates an external API/SDK/service may not seal
 * until a COVERAGE.md matrix enumerates the surface and every non-integrated
 * capability is an explicit, reasoned opt-out.
 *
 * Contract (two touch points composed into one check):
 *   1. If COVERAGE.md exists in the phase dir → validate it (acceptance #2).
 *      Block on any validation error (empty matrix, OPT-OUT without reason,
 *      duplicate/empty capability).
 *   2. If COVERAGE.md is absent → run detectApiIntegration over the phase scope
 *      (PLAN.md body, then ROADMAP phase section as fallback). If a strong
 *      external-API-integration signal is detected → BLOCK ("integration
 *      detected without coverage matrix"). If no signal → PASS (treat as a
 *      non-API phase; acceptance #4 — low false positives).
 *
 * The detector is the FALLBACK for the "nobody decided / forgot the matrix"
 * case; the primary path is the plan:pre contribution prompting COVERAGE.md.
 *
 * Args: check api-coverage.verify-pre <phase-dir>
 * Emits the uniform gate contract: { block, passed, message, ...details }.
 */
function cmdApiCoverageVerifyPre(projectDir, args, raw) {
    const phaseArg = typeof args[2] === 'string' ? args[2] : '';
    if (!phaseArg) {
        error('api-coverage.verify-pre requires a phase argument: check api-coverage.verify-pre <phase-dir-or-token>', ERROR_REASON.SDK_MISSING_ARG);
        return;
    }
    const pDir = planningDir(projectDir);
    const phasesRoot = node_path_1.default.join(pDir, 'phases');
    // SECURITY (path traversal): the phase argument is taken ONLY as a phase
    // token — its basename — and resolved by findPhaseInternal strictly under
    // .planning/phases/ (or a milestone archive). The raw arg is never used as a
    // path, so `..`, absolute paths, and arbitrary directories cannot reach a
    // file read. Mirrors cmdVerifySchemaDrift's token-match approach.
    let token = (0, shell_command_projection_cjs_1.posixNormalize)(phaseArg).split('/').filter(Boolean).pop() || '';
    // A token like ".." or "." carries no phase identity → unresolvable.
    if (token === '.' || token === '..')
        token = '';
    // Not a GSD project (no phases tree at all) → fail-open: nothing to gate.
    if (!node_fs_1.default.existsSync(phasesRoot)) {
        output({
            block: false,
            passed: true,
            coverage_present: false,
            detected: false,
            message: 'api-coverage: no .planning/phases directory; gate skipped (not a GSD project layout)',
        }, raw, undefined);
        return;
    }
    // Resolve the phase dir under the contained phases root.
    let resolvedDir = null;
    let phaseNumber = '';
    if (token) {
        const found = findPhaseInternal(projectDir, token);
        if (found && found.directory) {
            resolvedDir = found.directory;
            phaseNumber = found.phase_number || '';
        }
    }
    if (!resolvedDir) {
        // The phases tree EXISTS but THIS phase could not be resolved. For a
        // BLOCKING gate, fail-closed: a missing phase dir must not silently bypass
        // the coverage requirement. (Distinguished from "no .planning at all"
        // above, which is a genuine non-GSD-project → pass.)
        output({
            block: true,
            passed: false,
            coverage_present: false,
            detected: false,
            phase_lookup_failed: true,
            message: `api-coverage: could not resolve phase "${phaseArg}" under .planning/phases/. ` +
                'Resolve the phase directory (or produce COVERAGE.md) before sealing.',
        }, raw, undefined);
        return;
    }
    // Defense-in-depth: the resolved dir must be inside the phases root (or a
    // milestone archive under .planning/milestones).
    const milestonesRoot = node_path_1.default.join(pDir, 'milestones');
    if (!isInsideRoot(resolvedDir, phasesRoot) && !isInsideRoot(resolvedDir, milestonesRoot)) {
        output({
            block: true,
            passed: false,
            coverage_present: false,
            detected: false,
            message: 'api-coverage: resolved phase dir escapes .planning/ — refusing to evaluate',
        }, raw, undefined);
        return;
    }
    // (1) locate COVERAGE.md — prefer the exact name, then a single *-COVERAGE.md.
    let coverageFile = '';
    let suffixed = [];
    try {
        const entries = node_fs_1.default.readdirSync(resolvedDir, { withFileTypes: true });
        const files = entries.filter((e) => e.isFile()).map((e) => e.name);
        const exact = files.find((f) => /^COVERAGE\.md$/i.test(f));
        if (exact) {
            coverageFile = exact;
        }
        else {
            suffixed = files.filter((f) => /-COVERAGE\.md$/i.test(f)).sort();
            if (suffixed.length === 1)
                coverageFile = suffixed[0];
        }
    }
    catch {
        // readdir failure → treat as no matrix readable; fall through to detection.
    }
    if (coverageFile) {
        let matrixText;
        try {
            matrixText = node_fs_1.default.readFileSync(node_path_1.default.join(resolvedDir, coverageFile), 'utf8');
        }
        catch {
            // COVERAGE.md exists but is unreadable (EACCES/EIO/encoding). Fail-closed
            // with a useful message rather than a raw throw.
            output({
                block: true,
                passed: false,
                coverage_present: true,
                message: `api-coverage: COVERAGE.md exists but is unreadable — fix file permissions/encoding before sealing`,
            }, raw, undefined);
            return;
        }
        const v = validateCoverageMatrix(matrixText);
        if (v.valid) {
            if (v.none_declared) {
                // The declaration is the human override for the detector — it PASSES
                // even when detection fires (that is acceptance #5's point: the
                // detector is fallible and the declaration is the reasoned overrule).
                // But a contradiction must be VISIBLE, not silent: re-run detection
                // over the phase scope and surface any signals it still finds
                // (#2365 review S-1).
                const declScope = readPhaseScope(projectDir, resolvedDir, phaseNumber);
                const declDetection = detectApiIntegration(declScope.text);
                const declSignals = declDetection.signals.map((s) => ({ verb: s.verb, noun: s.noun }));
                // The declaration legitimately wins even over a read error (it is the
                // human overrule), but if scope was incomplete we say so — the contract
                // is that contradictions stay visible, not silent (#2365 review).
                const baseMsg = declDetection.detected
                    ? `api-coverage: COVERAGE.md declares no external API integration, overriding ${declSignals.length} detected signal(s) — confirm the declaration is accurate`
                    : 'api-coverage: COVERAGE.md declares no external API integration — matrix not required';
                output({
                    block: false,
                    passed: true,
                    coverage_present: true,
                    matrix: coverageFile,
                    counts: v.counts,
                    none_declared: true,
                    detected: declDetection.detected,
                    ...(declDetection.detected ? { signals: declSignals } : {}),
                    ...(declScope.readError ? { scope_read_error: declScope.readError } : {}),
                    message: declScope.readError
                        ? `${baseMsg} (note: phase scope was incompletely read — ${declScope.readError})`
                        : baseMsg,
                }, raw, undefined);
                return;
            }
            output({
                block: false,
                passed: true,
                coverage_present: true,
                matrix: coverageFile,
                counts: v.counts,
                message: `api-coverage: matrix present (${v.counts.surface} capabilities, ${v.counts.optout} opt-out)`,
            }, raw, undefined);
            return;
        }
        // Fixed-template message (no raw cell content echoed into the LLM-facing
        // message). The structured `errors` array is safe (row-indexed, no cell
        // values) and travels as data for tooling that wants detail.
        output({
            block: true,
            passed: false,
            coverage_present: true,
            matrix: coverageFile,
            error_count: v.errors.length,
            errors: v.errors,
            message: `api-coverage: COVERAGE.md has ${v.errors.length} problem(s) — fix the matrix (every capability INTEGRATE or OPT-OUT with a reason) before sealing`,
        }, raw, undefined);
        return;
    }
    if (suffixed.length > 1) {
        output({
            block: true,
            passed: false,
            coverage_present: false,
            message: `api-coverage: multiple *-COVERAGE.md files found (${suffixed.length}) — consolidate into one COVERAGE.md before sealing`,
        }, raw, undefined);
        return;
    }
    // (2) no matrix — detect whether this phase integrates an external API.
    const scope = readPhaseScope(projectDir, resolvedDir, phaseNumber);
    if (scope.readError) {
        // Fail-closed: an unreadable plan could be the one describing the
        // integration, so we cannot certify "no integration" — block and surface it.
        output({
            block: true,
            passed: false,
            coverage_present: false,
            detected: false,
            message: `api-coverage: could not read the phase scope (${scope.readError}); ` +
                'refusing to certify no external-API integration from incomplete scope. ' +
                'Fix the unreadable plan file, or add a COVERAGE.md declaration.',
        }, raw, undefined);
        return;
    }
    // An EMPTY scope is not a negative verdict. This gate's neighbouring arms
    // already fail closed (unresolvable phase → block; unreadable plan → block),
    // but a phase with no plan body AND no roadmap section fell through to
    // detection over zero bytes and CERTIFIED "no external-API integration" —
    // clearing a blocking seal gate on a probe that examined nothing
    // (ADR-3889 failure class (c), #3909). The discriminator is BYTES EXAMINED,
    // never SIGNALS FOUND: a phase with real plans and no API vocabulary still
    // reaches the pass below unchanged.
    if (scope.text.trim() === '') {
        output({
            block: true,
            passed: false,
            coverage_present: false,
            detected: false,
            scope_unavailable: true,
            message: 'api-coverage: the phase scope is empty — no plan body and no roadmap section were ' +
                'found, so nothing was examined. Refusing to certify no external-API integration ' +
                'from an unestablished scope. Add the phase plan, or add a COVERAGE.md declaration.',
        }, raw, undefined);
        return;
    }
    const detection = detectApiIntegration(scope.text);
    if (detection.detected) {
        // Surface only verb/noun (typed, bounded) — NOT raw prose snippets — so the
        // gate output cannot relay injected PLAN.md instructions to the orchestrator.
        const signals = detection.signals.map((s) => ({ verb: s.verb, noun: s.noun }));
        output({
            block: true,
            passed: false,
            coverage_present: false,
            detected: true,
            signals,
            message: 'api-coverage: external-API integration detected without a coverage matrix. ' +
                'Produce COVERAGE.md enumerating the API surface (every capability INTEGRATE or ' +
                'OPT-OUT with a reason) before sealing. Full coverage is the default.',
        }, raw, undefined);
        return;
    }
    output({
        block: false,
        passed: true,
        coverage_present: false,
        detected: false,
        message: 'api-coverage: no external-API integration detected; coverage matrix not required',
    }, raw, undefined);
}
/** A filesystem error that is NOT "does not exist" — i.e. a real read failure
 *  (EACCES/EIO/…) the gate must not swallow. `ENOENT` is a legitimate "not
 *  there yet" and is treated as absence, not error. */
function isRealReadFailure(err) {
    const code = err?.code;
    return err != null && code !== 'ENOENT';
}
function readPhaseScope(projectDir, phaseDir, phaseNumber) {
    const chunks = [];
    let readError = null;
    // A MISSING phase directory is fine (no plans yet → fall through to the
    // roadmap). Checked up front (rather than via a readdirSync catch) because
    // #3183 (lint-plan-count-drift) now sources the plan-file list from the
    // single owner (scanPhasePlans) instead of a local `-PLAN\.md$` readdirSync
    // filter — picks up bare PLAN.md and nested plans/, and excludes
    // superseded plans, none of which the prior root-only exact-suffix filter
    // did.
    if (node_fs_1.default.existsSync(phaseDir)) {
        const scan = scanPhasePlans(phaseDir);
        if (scan.scope === SCOPE.UNREADABLE) {
            // Directory exists but scanPhasePlans's own readdirSync(phaseDir) call
            // failed (EACCES/EIO race) — a real read failure the gate must not
            // silently pass (#2365 review), mirroring the prior isRealReadFailure
            // branch below for the readdirSync-throws case.
            return {
                text: '',
                readError: 'could not read the phase directory: scanPhasePlans reported scope UNREADABLE',
            };
        }
        const plans = [...scan.planFiles].sort();
        for (const p of plans) {
            try {
                chunks.push(node_fs_1.default.readFileSync(node_path_1.default.join(phaseDir, p), 'utf8'));
            }
            catch (err) {
                // A plan file that exists but cannot be read — record it and keep
                // reading the rest so the message names the first failure.
                if (!readError) {
                    readError = `could not read ${p}: ${err instanceof Error ? err.message : String(err)}`;
                }
            }
        }
    }
    if (readError)
        return { text: chunks.join('\n\n'), readError };
    if (chunks.join('').trim().length > 0)
        return { text: chunks.join('\n\n'), readError: null };
    // Fallback: ONLY this phase's ROADMAP section (not the whole file, which
    // would pollute detection with sibling-phase prose). A MISSING roadmap/section
    // is non-fatal; a roadmap that exists but cannot be read is a real failure.
    if (phaseNumber) {
        try {
            const section = getRoadmapPhaseWithFallback(projectDir, phaseNumber);
            if (section)
                return { text: section, readError: null };
        }
        catch (err) {
            if (isRealReadFailure(err)) {
                return {
                    text: '',
                    readError: `could not read the roadmap fallback: ${err instanceof Error ? err.message : String(err)}`,
                };
            }
        }
    }
    return { text: '', readError: null };
}
function routeCheckCommand({ args, cwd, raw }) {
    // Normalize dots to hyphens in the subcommand so both forms are accepted.
    // This makes `check.query = "ui.plan-gate"` (dotted form in capability.json gates)
    // directly runnable as `gsd_run check ui.plan-gate` — the dot is normalized to
    // `ui-plan-gate` before routing. The generic gate-dispatch in §5.6 reads
    // `check.query` from the active gate hook and runs `gsd_run check ${hook.check.query}`,
    // so the declared query must be dispatchable exactly as declared.
    const rawSubcommand = args[1];
    const subcommand = typeof rawSubcommand === 'string' ? rawSubcommand.replace(/\./g, '-') : rawSubcommand;
    if (subcommand === 'auto-mode') {
        cmdAutoMode(cwd, raw);
        return;
    }
    if (subcommand === 'decision-coverage-plan') {
        cmdDecisionCoveragePlan(cwd, args, raw);
        return;
    }
    if (subcommand === 'decision-coverage-verify') {
        cmdDecisionCoverageVerify(cwd, args, raw);
        return;
    }
    if (subcommand === 'ui-plan-gate') {
        cmdUiPlanGate(cwd, args, raw);
        return;
    }
    if (subcommand === 'gap-analysis-plan-post') {
        cmdGapAnalysisPlanPost(cwd, args, raw);
        return;
    }
    if (subcommand === 'verify-command-paths') {
        // Deterministic filesystem probe for <automated> verify commands (#2401) —
        // never executes anything; see verify-command-grounding.cjs.
        cmdVerifyCommandPaths(cwd, args, raw);
        return;
    }
    if (subcommand === 'verify-failure-directions') {
        // Presence probe for a stated <fails_when> per <automated> command
        // (#3172) — never executes anything; see verify-command-grounding.cjs.
        cmdVerifyFailureDirections(cwd, args, raw);
        return;
    }
    if (subcommand === 'api-coverage-verify-pre') {
        // ai-integration capability blocking gate at verify:pre (#1562). Dot-to-
        // hyphen normalization means query "api-coverage.verify-pre" routes here.
        cmdApiCoverageVerifyPre(cwd, args, raw);
        return;
    }
    if (subcommand === 'tdd-review-checkpoint') {
        cmdTddReviewCheckpoint(cwd, args, raw);
        return;
    }
    if (subcommand === 'tdd-red-evidence') {
        // #3770: intentional-RED evidence gate — only a target-test failure may
        // authorize GREEN. Validates the persisted record; never executes anything.
        cmdTddRedEvidence(cwd, args, raw);
        return;
    }
    if (subcommand === 'ui-safety-gate') {
        cmdUiSafetyGate(cwd, args, raw);
        return;
    }
    if (subcommand === 'verify-schema-drift') {
        // Delegates to verify.schema-drift — drift capability gate at execute:wave:post (blocking).
        // Dot-to-hyphen normalization means query "verify.schema-drift" routes here.
        // Honor GSD_SKIP_SCHEMA_CHECK=true to bypass the gate (preserves the original inline gate behavior).
        const phaseArg = typeof args[2] === 'string' ? args[2] : '';
        const skipSchemaCheck = process.env['GSD_SKIP_SCHEMA_CHECK'] === 'true';
        cmdVerifySchemaDrift(cwd, phaseArg, skipSchemaCheck, raw);
        return;
    }
    if (subcommand === 'verify-codebase-drift') {
        // Delegates to verify.codebase-drift — drift capability gate at execute:wave:post (non-blocking).
        // Dot-to-hyphen normalization means query "verify.codebase-drift" routes here.
        cmdVerifyCodebaseDrift(cwd, raw);
        return;
    }
    if (subcommand === 'verify-context-drift') {
        // Delegates to verify.context-drift — drift capability gate at plan:pre (non-blocking).
        // Dot-to-hyphen normalization means query "verify.context-drift" routes here.
        const phaseArg = typeof args[2] === 'string' ? args[2] : '';
        cmdVerifyContextDrift(cwd, phaseArg, raw);
        return;
    }
    if (subcommand === 'predicate') {
        // Generic gate-predicate evaluator (#2008). The workflow gate-dispatch calls
        // this for any gate whose `check` carries a `predicate` (instead of a `query`),
        // passing the predicate object as --predicate '<json>'. NOTE: unlike the
        // `check.query` subcommands above (which take positional phase args), this
        // subcommand parses --flag value pairs.
        cmdCheckPredicate(cwd, args, raw);
        return;
    }
    if (subcommand === 'prohibition-enforcement') {
        // The deterministic test-tier prohibition PRODUCER/gate (#1259, ADR-550 D5d). Locates the
        // wired mechanical check (node-test or lint-rule), confirms fail-first, runs it, builds
        // enforcementEvidence, and emits the dispositionForProhibition verdict. Invocable as
        // `gsd_run check prohibition-enforcement <request.json>`.
        (0, prohibition_enforcement_cjs_1.routeProhibitionEnforcement)(args, raw);
        return;
    }
    error('Unknown check subcommand. Available: api-coverage-verify-pre, auto-mode, decision-coverage-plan, decision-coverage-verify, gap-analysis-plan-post, predicate, prohibition-enforcement, tdd-red-evidence, tdd-review-checkpoint, ui-plan-gate, ui-safety-gate, verify-command-paths, verify-failure-directions, verify-schema-drift, verify-codebase-drift, verify-context-drift', ERROR_REASON.SDK_UNKNOWN_COMMAND);
}
module.exports = {
    routeCheckCommand,
    decisionMentioned,
    extractPlanDesignatedSections,
    computeUiPlanGate,
    computeUiSafetyGate,
    cmdGapAnalysisPlanPost,
    cmdVerifyCommandPaths,
    cmdVerifyFailureDirections,
    cmdTddReviewCheckpoint,
    cmdTddRedEvidence,
    cmdCheckPredicate,
    buildPredicateDeps,
    parsePredicateFlags,
    // Fail-closed phase-scope reader for the api-coverage gate — exported for
    // in-process failure-injection tests (#2365 review).
    readPhaseScope,
};
