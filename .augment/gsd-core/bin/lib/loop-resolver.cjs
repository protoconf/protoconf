"use strict";
/**
 * Loop Resolver — ADR-857 phase 3c registry-consuming query
 *
 * Given a loop point (one of the 12 canonical points from loop-host-contract.cjs),
 * filters the materialized Capability Registry by config activation and returns
 * the active hooks as a JSON envelope with a rendered-markdown field.
 *
 * Consumed live by the landed phase-6 loop-hook cutovers: plan-phase.md / autonomous.md
 * at plan:pre (ui-phase) and autonomous.md at verify:post (ui-review). Further per-feature
 * cutovers are ongoing.
 *
 * Command surface: gsd-tools loop render-hooks <point>
 *
 * Exports (three things):
 *   resolveLoopHooks({ point, registry, config }) → { point, activeHooks }
 *   renderLoopHooks(resolved) → markdown string
 *   cmdLoopRenderHooks(cwd, point, raw, options) — I/O entry point
 *
 * Both pure functions (resolveLoopHooks, renderLoopHooks) take explicit
 * registry/config arguments so they are trivially testable without I/O.
 *
 * Dependencies (leaf modules only — no circular risk):
 *   - ./config-loader.cjs  (loadConfig)
 *   - ./io.cjs             (output, error)
 *   - ./capability-activation.cjs (resolveConfigKey, _resolveActivationValue, _getNestedConfigValue, _readRawConfigKey)
 *   - loop-host-contract.cjs (CANONICAL_POINTS via LOOP_HOST_CONTRACT)
 *   - capability-registry.cjs (byLoopPoint, consumed at call time)
 *   - capability-state.cjs (resolveCapabilityRuntimeState — for capabilities list)
 */
// eslint-disable-next-line @typescript-eslint/no-require-imports
const ioMod = require("./io.cjs");
const { output: coreOutput, error: coreError } = ioMod;
// eslint-disable-next-line @typescript-eslint/no-require-imports
const configLoaderModule = require("./config-loader.cjs");
const { loadConfig } = configLoaderModule;
// eslint-disable-next-line @typescript-eslint/no-require-imports
const capabilityStateModule = require("./capability-state.cjs");
const { resolveCapabilityRuntimeState } = capabilityStateModule;
// ─── Capability-activation engine (single owner for config-key precedence) ────
// eslint-disable-next-line @typescript-eslint/no-require-imports
const capabilityActivationModule = require("./capability-activation.cjs");
const { _getNestedConfigValue, _readRawConfigKey, _resolveActivationValue, _resolvePointGate, resolveConfigKey } = capabilityActivationModule;
// ─── Canonical points (derived from LOOP_HOST_CONTRACT — authoritative 12) ───
// FIX 2: Derive the authoritative canonical set from LOOP_HOST_CONTRACT so it
// cannot drift from the host contract. CANONICAL_POINTS_FALLBACK is kept as an
// alias for backward compatibility in tests and exports.
// eslint-disable-next-line @typescript-eslint/no-require-imports
const _loopHostContract = require('./loop-host-contract.cjs');
const CANONICAL_POINTS = (() => {
    try {
        const contract = _loopHostContract.LOOP_HOST_CONTRACT;
        if (Array.isArray(contract)) {
            const pts = [];
            for (const step of contract) {
                if (step && Array.isArray(step.points)) {
                    for (const p of step.points) {
                        if (typeof p === 'string')
                            pts.push(p);
                    }
                }
            }
            if (pts.length > 0)
                return pts;
        }
    }
    catch { /* fall through to hardcoded fallback */ }
    return [
        'discuss:pre',
        'discuss:post',
        'plan:pre',
        'plan:post',
        'execute:pre',
        'execute:wave:pre',
        'execute:wave:post',
        'execute:post',
        'verify:pre',
        'verify:post',
        'ship:pre',
        'ship:post',
    ];
})();
// Alias for backward compatibility (tests import this name)
const CANONICAL_POINTS_FALLBACK = CANONICAL_POINTS;
// FIX 2: _getCanonicalPoints now returns the authoritative CANONICAL_POINTS set
// derived from LOOP_HOST_CONTRACT — not the registry's byLoopPoint keys.
// The registry's byLoopPoint is only used to READ hooks, not to define valid points.
function _getCanonicalPoints(_registry) {
    return CANONICAL_POINTS;
}
// ─── Pure resolver ─────────────────────────────────────────────────────────────
/**
 * Pure resolver: given a point, registry, and config, returns the active hooks.
 *
 * Throws if `point` is not one of the 12 canonical points (caller converts to
 * io.error). Never throws for malformed registry/hook entries — skips and
 * continues.
 *
 * Ordering: steps first, then contributions, then gates. Within each array,
 * the materialized registry order is preserved.
 *
 * Activation: a hook with no `when` is always active. With `when` (dotted key),
 * resolved against `config`; active iff truthy. Inactive hooks are filtered out.
 */
function resolveLoopHooks(input) {
    const { point, registry, config, cwd, capabilityStatesById } = input;
    // Validate point
    const canonicalPoints = _getCanonicalPoints(registry);
    if (!canonicalPoints.includes(point)) {
        throw new Error(`Invalid loop point: "${point}". Valid points: ${canonicalPoints.join(', ')}`);
    }
    // Guard: registry missing byLoopPoint
    const byLoopPoint = registry['byLoopPoint'];
    if (!byLoopPoint || typeof byLoopPoint !== 'object' || Array.isArray(byLoopPoint)) {
        return { point, activeHooks: [] };
    }
    const byLoopPointMap = byLoopPoint;
    // Guard: point missing in registry
    const entry = byLoopPointMap[point];
    if (!entry || typeof entry !== 'object' || Array.isArray(entry)) {
        return { point, activeHooks: [] };
    }
    const entryMap = entry;
    const activeHooks = [];
    // Helper: check activation using single-key precedence resolver (FIX 1 + FIX 3)
    function isActive(hook) {
        const when = hook['when'];
        if (when !== undefined && when !== null) {
            // FIX 3: `when` present but not a non-empty string → malformed registry data → INACTIVE
            if (typeof when !== 'string' || when.length === 0)
                return false;
            if (!_resolveActivationValue(when, config, cwd, registry))
                return false;
        }
        // #3661: optional point-selection gate — see capability-activation.cts.
        return _resolvePointGate(hook['pointFrom'], point, config, cwd, registry);
    }
    function isCapabilityActive(capId) {
        if (!capabilityStatesById)
            return true;
        const state = capabilityStatesById instanceof Map
            ? capabilityStatesById.get(capId)
            : capabilityStatesById[capId];
        if (!state)
            return false;
        // Fail-closed gate: only render the hook when active is explicitly true.
        // A capability can be installed and surfaced (enabled=true) but config-disabled
        // (active=false); in that case the hook must not render.
        // Phase 4 tri-state alignment: `active` is now required (not optional), so
        // `=== true` is the correct fail-closed check (not `!== false`).
        return state.active === true;
    }
    // Helper: safe string array
    function toStringArray(v) {
        if (!Array.isArray(v))
            return [];
        return v.filter((x) => typeof x === 'string');
    }
    function toFragment(v) {
        if (!v || typeof v !== 'object' || Array.isArray(v))
            return undefined;
        const raw = v;
        const fragment = {};
        if (typeof raw.inline === 'string')
            fragment.inline = raw.inline;
        if (typeof raw.path === 'string')
            fragment.path = raw.path;
        return Object.keys(fragment).length > 0 ? fragment : undefined;
    }
    /**
     * Resolve declared configValues for a contribution hook.
     * The hook may carry `configValues: { alias: "dotted.key", ... }`.
     * Each key is resolved using the same four-level precedence as activation resolution,
     * but returning the raw value (not coerced to boolean) so numeric/string config values
     * are preserved (e.g. security_asvs_level: 2, security_block_on: "medium").
     */
    function resolveConfigValues(hook) {
        const raw = hook['configValues'];
        if (!raw || typeof raw !== 'object' || Array.isArray(raw))
            return undefined;
        const rawMap = raw;
        const resolved = {};
        for (const [alias, dotKey] of Object.entries(rawMap)) {
            // Prototype-pollution guard (inline literal, CodeQL barrier)
            if (alias === '__proto__' || alias === 'constructor' || alias === 'prototype')
                continue;
            if (typeof dotKey !== 'string')
                continue;
            const r = resolveConfigKey(dotKey, { config, cwd, registry });
            if (r.found)
                resolved[alias] = r.value;
        }
        return Object.keys(resolved).length > 0 ? resolved : undefined;
    }
    // Process steps
    const stepsRaw = entryMap['steps'];
    const steps = Array.isArray(stepsRaw) ? stepsRaw : [];
    for (const hook of steps) {
        if (!hook || typeof hook !== 'object')
            continue;
        const capId = typeof hook['capId'] === 'string' ? hook['capId'] : '';
        if (!isCapabilityActive(capId))
            continue;
        if (!isActive(hook))
            continue;
        const ref = (typeof hook['ref'] === 'object' && hook['ref'] !== null)
            ? hook['ref']
            : undefined;
        const when = typeof hook['when'] === 'string' ? hook['when'] : undefined;
        const fragment = toFragment(hook['fragment']);
        const produces = toStringArray(hook['produces']);
        const consumes = toStringArray(hook['consumes']);
        const onError = typeof hook['onError'] === 'string' ? hook['onError'] : undefined;
        const active = { capId, kind: 'step' };
        if (ref !== undefined)
            active.ref = ref;
        if (fragment !== undefined)
            active.fragment = fragment;
        if (when !== undefined)
            active.when = when;
        if (produces.length > 0)
            active.produces = produces;
        if (consumes.length > 0)
            active.consumes = consumes;
        if (onError !== undefined)
            active.onError = onError;
        activeHooks.push(active);
    }
    // Process contributions
    const contributionsRaw = entryMap['contributions'];
    const contributions = Array.isArray(contributionsRaw) ? contributionsRaw : [];
    for (const hook of contributions) {
        if (!hook || typeof hook !== 'object')
            continue;
        const capId = typeof hook['capId'] === 'string' ? hook['capId'] : '';
        if (!isCapabilityActive(capId))
            continue;
        if (!isActive(hook))
            continue;
        const into = typeof hook['into'] === 'string' ? hook['into'] : undefined;
        const fragment = toFragment(hook['fragment']);
        const when = typeof hook['when'] === 'string' ? hook['when'] : undefined;
        const produces = toStringArray(hook['produces']);
        const consumes = toStringArray(hook['consumes']);
        const onError = typeof hook['onError'] === 'string' ? hook['onError'] : undefined;
        const configValuesResolved = resolveConfigValues(hook);
        const active = { capId, kind: 'contribution' };
        if (into !== undefined)
            active.into = into;
        if (fragment !== undefined)
            active.fragment = fragment;
        if (when !== undefined)
            active.when = when;
        if (produces.length > 0)
            active.produces = produces;
        if (consumes.length > 0)
            active.consumes = consumes;
        if (onError !== undefined)
            active.onError = onError;
        if (configValuesResolved !== undefined)
            active.configValues = configValuesResolved;
        activeHooks.push(active);
    }
    // Process gates
    const gatesRaw = entryMap['gates'];
    const gates = Array.isArray(gatesRaw) ? gatesRaw : [];
    for (const hook of gates) {
        if (!hook || typeof hook !== 'object')
            continue;
        const capId = typeof hook['capId'] === 'string' ? hook['capId'] : '';
        if (!isCapabilityActive(capId))
            continue;
        if (!isActive(hook))
            continue;
        const when = typeof hook['when'] === 'string' ? hook['when'] : undefined;
        const check = hook['check'] !== undefined ? hook['check'] : undefined;
        const blocking = typeof hook['blocking'] === 'boolean' ? hook['blocking'] : undefined;
        const onError = typeof hook['onError'] === 'string' ? hook['onError'] : undefined;
        const active = { capId, kind: 'gate' };
        if (when !== undefined)
            active.when = when;
        if (check !== undefined)
            active.check = check;
        if (blocking !== undefined)
            active.blocking = blocking;
        if (onError !== undefined)
            active.onError = onError;
        activeHooks.push(active);
    }
    return { point, activeHooks };
}
// ─── Pure renderer ─────────────────────────────────────────────────────────────
/**
 * Pure renderer: given a resolved result, returns a deterministic markdown string.
 *
 * Empty active set → returns a "no active hooks" placeholder line.
 * Steps: heading with ordinal + skill ref + capId, produces/consumes lines.
 * Contributions: labeled block.
 * Gates: check name, blocking flag, onError.
 */
function renderLoopHooks(resolved) {
    const { point, activeHooks } = resolved;
    if (activeHooks.length === 0) {
        return `_No active hooks at ${point}._`;
    }
    const lines = [];
    let stepOrdinal = 0;
    for (const hook of activeHooks) {
        if (hook.kind === 'step') {
            stepOrdinal += 1;
            const refStr = hook.ref?.skill
                ? `skill:${hook.ref.skill}`
                : hook.ref?.agent
                    ? `agent:${hook.ref.agent}`
                    : JSON.stringify(hook.ref ?? {});
            lines.push(`### Step ${stepOrdinal}: ${refStr} (${hook.capId})`);
            if (hook.produces && hook.produces.length > 0) {
                lines.push(`- produces: ${hook.produces.join(', ')}`);
            }
            if (hook.consumes && hook.consumes.length > 0) {
                lines.push(`- consumes: ${hook.consumes.join(', ')}`);
            }
            if (hook.when) {
                lines.push(`- when: \`${hook.when}\``);
            }
            if (hook.onError) {
                lines.push(`- onError: ${hook.onError}`);
            }
            if (hook.fragment?.inline) {
                lines.push('');
                lines.push(hook.fragment.inline);
            }
            else if (hook.fragment?.path) {
                lines.push('');
                lines.push(`_Step fragment path is declared but not rendered by loop-resolver: ${hook.fragment.path}_`);
            }
            lines.push('');
        }
        else if (hook.kind === 'contribution') {
            lines.push(`<contribution from="${hook.capId}" into="${hook.into ?? '(unset)'}">`);
            if (hook.fragment?.inline) {
                lines.push(hook.fragment.inline);
            }
            else if (hook.fragment?.path) {
                lines.push(`_Contribution fragment path is declared but not rendered by loop-resolver: ${hook.fragment.path}_`);
            }
            if (hook.produces && hook.produces.length > 0) {
                lines.push(`- produces: ${hook.produces.join(', ')}`);
            }
            if (hook.consumes && hook.consumes.length > 0) {
                lines.push(`- consumes: ${hook.consumes.join(', ')}`);
            }
            if (hook.when) {
                lines.push(`- when: \`${hook.when}\``);
            }
            if (hook.onError) {
                lines.push(`- onError: ${hook.onError}`);
            }
            lines.push('</contribution>');
            lines.push('');
        }
        else if (hook.kind === 'gate') {
            let checkStr = '(none)';
            if (hook.check !== undefined && hook.check !== null) {
                checkStr = typeof hook.check === 'object'
                    ? JSON.stringify(hook.check)
                    : typeof hook.check === 'string' || typeof hook.check === 'number' || typeof hook.check === 'boolean'
                        ? String(hook.check)
                        : '(complex)';
            }
            lines.push(`**Gate** (${hook.capId}): check=${checkStr}, blocking=${String(hook.blocking ?? false)}, onError=${hook.onError ?? 'skip'}`);
            if (hook.when) {
                lines.push(`- when: \`${hook.when}\``);
            }
            lines.push('');
        }
    }
    // Trim trailing blank line
    while (lines.length > 0 && lines[lines.length - 1] === '') {
        lines.pop();
    }
    return lines.join('\n');
}
// ─── I/O command handler ───────────────────────────────────────────────────────
/**
 * Command entry point: load registry + config, resolve + render, emit envelope.
 *
 * Envelope: { point, activeHooks, rendered }
 * On invalid point, emits io.error instead of throwing.
 *
 * Config note: FIX 1 replaced _loadMergedConfig (whole-config deep-merge) with a
 * per-hook single-key activation resolver (_resolveActivationValue). The resolver
 * checks loadConfig result first, then raw config.json files directly (workstream
 * then root), then the registry's configSchema default. This eliminates the
 * merged-object-from-untrusted-keys security concern and correctly handles
 * pre-cutover keys like `workflow.ui_phase` that live in config.json but are not
 * yet exposed through loadConfig's whitelist.
 *
 * --active-cap <capId>: when present, resolves hooks for <point> exactly as the
 * normal path does, then prints exactly `true` (if any resolved activeHook has
 * capId === <capId>) or `false` followed by a single newline, and exits 0.
 * No JSON envelope is emitted — output is clean for shell $(…) capture.
 * Missing <capId> value → coreError + non-zero exit.
 * Unknown/inactive capId → `false` (not an error).
 */
// #2009: a capability id surfaced inside the runnable `gsd capability remove <id>`
// remediation must match the canonical kebab-case id shape (identical to
// capability-consent.cts / capability-ledger.cts) before it is embedded — a raw
// overlay directory name is attacker-controlled and can carry shell/markdown
// metacharacters (backticks, ';', '|', '$()'). An id that fails this check is
// withheld and no runnable command is rendered for it.
const LOAD_FAIL_CAP_ID_RE = /^[a-z][a-z0-9-]*$/;
// #2009: neutralize control chars, newlines, and backticks from a third-party
// load-failure reason so a malicious manifest cannot break out of the warning
// line or inject markdown / prompt content into the surfaced message.
function sanitizeLoadFailReason(reason) {
    const cleaned = String(reason)
        // Strip C0 control chars, DEL, and backticks; collapse remaining whitespace.
        .replace(/[\x00-\x1F\x7F`]/g, ' ')
        .replace(/\s+/g, " ")
        .trim()
        .slice(0, 300);
    return cleaned || '(no reason given)';
}
function cmdLoopRenderHooks(cwd, point, raw, options = {}) {
    if (!point) {
        coreError('loop render-hooks requires a <point> argument. Valid points: ' + CANONICAL_POINTS.join(', '));
        return;
    }
    // --active-cap <capId> mode: emit 'true' or 'false' only (scanner-safe, no JSON envelope)
    const activeCapId = typeof options['activeCap'] === 'string' ? options['activeCap'] : undefined;
    if (activeCapId !== undefined && activeCapId === '') {
        coreError('--active-cap requires a <capId> value (e.g. --active-cap tdd)');
        return;
    }
    const runtimeConfigDir = typeof options['configDir'] === 'string'
        ? options['configDir']
        : undefined;
    // #2003: thread an explicit --runtime override into the capability-state
    // resolver so the config-dir resolution bypasses the persisted-runtime
    // fallback (GSD_RUNTIME → config.runtime). Without this, a repo with persisted
    // runtime:"codex" resolves the config dir to ~/.codex and execute:post /
    // verify:post hooks silently no-op when the operator drives from Claude Code.
    const runtimeOverride = typeof options['runtime'] === 'string' ? options['runtime'] : undefined;
    // Load the config snapshot ONCE and share it with both the capability-state
    // resolver (via configOverride) and loop-hook resolution, so federated keys
    // present in loadConfig resolve identically for `active` and for hook when/
    // configValues — eliminating the previous double loadConfig() call. Note: keys
    // absent from loadConfig still fall through to raw .planning/config.json reads
    // (precedence levels 2-3) in each pass; that residual re-read window is
    // pre-existing (unchanged by this consolidation), not introduced here.
    let config;
    try {
        config = loadConfig(cwd);
    }
    catch {
        config = {};
    }
    const state = resolveCapabilityRuntimeState(cwd, runtimeConfigDir, config, runtimeOverride);
    // Load overlay-aware registry (ADR-1244 D2 wiring) so installed third-party
    // capabilities are visible to loop rendering exactly like first-party ones.
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const { loadRegistry } = require('./capability-loader.cjs');
    // #1459 IC-04: thread the consent home (process.env.GSD_HOME) EXPLICITLY so a consented project cap's
    // loop surfaces (steps/gates/contributions) render here at the SAME home that gated its activation.
    const registry = loadRegistry({ includeInstalled: true, cwd, gsdHome: process.env['GSD_HOME'] });
    const capabilityStatesById = new Map();
    for (const cap of state.capabilities || []) {
        capabilityStatesById.set(cap.id, cap);
    }
    let resolved;
    try {
        resolved = resolveLoopHooks({ point, registry, config, cwd, capabilityStatesById });
    }
    catch (err) {
        const msg = (err instanceof Error) ? err.message : String(err);
        coreError(msg);
        return;
    }
    // ── ADR-1244 D2: load-failed capability gates FAIL OPEN with a loud warning ────
    // Decision (#2009): a capability that failed to LOAD must not block the loop.
    // The prior behavior injected a BLOCKING synthetic gate (blocking:true,
    // onError:'halt') at every point where the skipped cap declared a gate, so a
    // single incompatible capability halted every ship:pre / verify:post
    // project-wide for a load error unrelated to what the gate checked — with no
    // remediation surfaced. We now fail OPEN: no gate is injected (the loop proceeds
    // and `--active-cap <failed-cap>` correctly reports it inactive), and a loud
    // warning is emitted instead — to STDERR (which the operator, or the agent
    // running the command, actually sees regardless of how the host workflow
    // consumes stdout) AND in the envelope's `warnings` channel for structured
    // consumers. The warning names the load reason and the exact
    // `gsd capability remove <id>` remediation so the operator can clear the broken
    // capability. blockedGates is still recorded by the loader; only the consequence
    // changes from block to warn. step/contribution overlays were already skip-open.
    //
    // The gate injection was dropped rather than made non-blocking because no host
    // workflow generically surfaces an arbitrary gate's message at ship:pre /
    // verify:post (consumers dispatch on specific capIds / ref.skills), and the
    // generic gate consumers expect an object-shaped `check`, not a prose string —
    // so an injected advisory gate would be silently dropped or mis-dispatched. A
    // stderr warning is the channel that is actually surfaced. (See #2009 review.)
    const overlayMeta = registry['_overlay'];
    const loadFailWarnings = [];
    if (overlayMeta && Array.isArray(overlayMeta.blockedGates)) {
        for (const blocked of overlayMeta.blockedGates) {
            if (blocked.point !== point)
                continue;
            // Security (#2009 review): capId/reason come from a third-party manifest or
            // directory name. Validate capId before embedding it in the runnable
            // remediation command; withhold it (no runnable command) if it is not a
            // canonical id. Strip control chars/backticks from reason.
            const idValid = LOAD_FAIL_CAP_ID_RE.test(String(blocked.capId));
            const capLabel = idValid
                ? `"${blocked.capId}"`
                : 'with an invalid id (withheld) under .gsd/capabilities/';
            const remediation = idValid
                ? `Run \`gsd capability remove ${blocked.capId}\` to remove it, or fix the load error.`
                : 'Remove the offending capability directory under .gsd/capabilities/, or fix the load error.';
            loadFailWarnings.push(`capability ${capLabel} failed to load (${sanitizeLoadFailReason(blocked.reason)}); ` +
                `its gate at ${point} is SKIPPED and NOT enforced (failing open). ${remediation}`);
        }
    }
    // Emit loudly to stderr in EVERY output mode (including --active-cap), so a
    // skipped gate is never silently invisible to the operator/agent.
    for (const w of loadFailWarnings) {
        process.stderr.write(`gsd: warning — ${w}\n`);
    }
    // --active-cap mode: print exactly 'true' or 'false' with no envelope
    if (activeCapId !== undefined) {
        const isActive = resolved.activeHooks.some((h) => h.capId === activeCapId);
        process.stdout.write(isActive ? 'true\n' : 'false\n');
        return;
    }
    const rendered = renderLoopHooks(resolved);
    const envelope = {
        point: resolved.point,
        activeHooks: resolved.activeHooks,
        rendered,
    };
    // Surface capability-state warnings and the #2009 load-failure fail-open
    // warnings together in the structured `warnings` channel (in addition to the
    // stderr emission above, which is the channel host workflows actually see).
    const combinedWarnings = [...(state.warnings || []), ...loadFailWarnings];
    if (combinedWarnings.length > 0) {
        envelope.warnings = combinedWarnings;
    }
    coreOutput(envelope, raw);
}
module.exports = {
    resolveLoopHooks,
    renderLoopHooks,
    cmdLoopRenderHooks,
    // Exported for tests
    _getNestedConfigValue,
    _resolveActivationValue,
    _readRawConfigKey,
    // Re-exported for identity parity guard (FIX 2: resolveConfigValues in this module
    // calls resolveConfigKey; exporting it here makes the single-owner contract testable).
    resolveConfigKey,
    // #3661: re-exported for the same identity parity guard — isActive calls
    // _resolvePointGate; exporting it here makes the single-owner contract testable
    // (see tests/capability-precedence-parity.test.cjs's identity guard describe block).
    _resolvePointGate,
    CANONICAL_POINTS_FALLBACK,
    CANONICAL_POINTS,
};
