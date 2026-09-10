"use strict";
/**
 * Command Argument Projection Module (ADR-457 build-at-publish: the
 * hand-written bin/lib/command-arg-projection.cjs collapsed to a TypeScript
 * source of truth).
 *
 * ADR-3473 §8.4 ("failure is a value"): `parseNamedArgs` is now strict and
 * returns a `Result` instead of silently accepting unrecognized or stray
 * positional tokens. See .gsd/phase/feat-3884-failure-is-a-value/40-design.md
 * for the full behavior table (A1-A18) and negative-space section (N1-N8).
 *
 * Shared helpers for command-family adapters to project argv tokens into
 * typed named values and multi-word segments.
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.isFlagToken = isFlagToken;
exports.parseNamedArgs = parseNamedArgs;
exports.parseNamedArgsOrExit = parseNamedArgsOrExit;
exports.parseMultiwordArg = parseMultiwordArg;
// eslint-disable-next-line @typescript-eslint/no-require-imports
const io = require("./io.cjs");
const { ERROR_REASON, formatDiagnosticToken } = io;
// ─── Internal helpers ─────────────────────────────────────────────────────────
function isPlainObject(v) {
    return typeof v === 'object' && v !== null && !Array.isArray(v);
}
/**
 * ADR-3473 Decision 2: both ends of this seam are gsd-core's own source, so a
 * malformed spec (a stale call site still using the legacy
 * `parseNamedArgs(args, valueFlags, booleanFlags)` shape, or a missing spec
 * entirely) is an internal invariant violation, not user input — it throws
 * loudly instead of destructuring `undefined` off a Result.
 */
function assertValidSpec(spec) {
    if (!isPlainObject(spec)) {
        throw new TypeError('parseNamedArgs: spec must be an object of shape ' +
            '{ valueFlags?: string[], booleanFlags?: string[], positionals: number | "rest" }. ' +
            'Received a missing, array, or non-object value — this is the legacy ' +
            'parseNamedArgs(args, valueFlags, booleanFlags) call shape, retired by ADR-3473 §8.4.');
    }
    const positionals = spec.positionals;
    const positionalsValid = positionals === 'rest' ||
        (typeof positionals === 'number' && Number.isInteger(positionals) && positionals >= 0);
    if (!positionalsValid) {
        throw new TypeError('parseNamedArgs: spec.positionals must be a non-negative integer or the literal "rest" ' +
            `— received ${JSON.stringify(positionals)}.`);
    }
}
// Single predicate reused for both extraction (a value beginning with a
// single `-` is a value, not a flag) and validation (negative space N5).
// Exported for init-command-router's #3865 --phase alias normalization,
// which needs the same flag-shape test before deciding whether args[2] is a
// phase positional or a flag token.
function isFlagToken(tok) {
    return tok.startsWith('--');
}
// ─── parseNamedArgs ───────────────────────────────────────────────────────────
/**
 * Project argv tokens into typed named values, then strictly validate every
 * token past the caller's declared positional boundary.
 *
 * Extraction (unchanged semantics, #312): first occurrence wins; a value
 * flag whose next token is absent or starts with `--` yields `null` (this is
 * NOT a validation error — see the value-flag branch below); boolean flags
 * and optional-value flags (`optionalValueFlags`, #2932's `--wave` shape) are
 * presence tests. Kept as a single first-index Map so the flag loops don't
 * each re-scan argv — O(argv + flags) instead of O(flags * argv).
 *
 * Validation (skipped entirely when `positionals === 'rest'`): a single
 * left-to-right cursor walk from `spec.positionals`, per the design doc's
 * Kernighan's Law note — debuggable over clever, never a set-difference.
 */
function parseNamedArgs(args, spec) {
    assertValidSpec(spec);
    const valueFlags = spec.valueFlags ?? [];
    const booleanFlags = spec.booleanFlags ?? [];
    const optionalValueFlags = spec.optionalValueFlags ?? [];
    const firstIndex = new Map();
    for (let i = 0; i < args.length; i++) {
        if (!firstIndex.has(args[i]))
            firstIndex.set(args[i], i);
    }
    const data = {};
    for (const flag of valueFlags) {
        const idx = firstIndex.has(`--${flag}`) ? firstIndex.get(`--${flag}`) : -1;
        data[flag] =
            idx !== -1 && args[idx + 1] !== undefined && !isFlagToken(args[idx + 1])
                ? args[idx + 1]
                : null;
    }
    for (const flag of booleanFlags) {
        data[flag] = firstIndex.has(`--${flag}`);
    }
    // Optional-value flags (#2932: `--wave`-shaped) — presence-only, exactly
    // like booleanFlags. The value (if any) is deliberately not surfaced here;
    // see the NamedArgSpec.optionalValueFlags JSDoc.
    for (const flag of optionalValueFlags) {
        data[flag] = firstIndex.has(`--${flag}`);
    }
    if (spec.positionals === 'rest') {
        return { ok: true, data };
    }
    const valueFlagSet = new Set(valueFlags);
    const booleanFlagSet = new Set(booleanFlags);
    const optionalValueFlagSet = new Set(optionalValueFlags);
    const flagList = [
        ...valueFlags.map((f) => `--${f} <value>`),
        ...booleanFlags.map((f) => `--${f}`),
        ...optionalValueFlags.map((f) => `--${f} [value]`),
    ];
    let i = spec.positionals;
    while (i < args.length) {
        const tok = args[i];
        if (isFlagToken(tok)) {
            const name = tok.slice(2);
            if (valueFlagSet.has(name)) {
                // A value flag whose next token is absent or flag-shaped resolves to
                // `null` in `data` (see the extraction loop above) — this is NOT an
                // error (#3180/behavior-lock: emptyPrdValueIsFalsyAndTreatedAsAbsent,
                // section-manifest-init-facts.test.cjs "flag-shaped value"). Advance
                // by 1 so the following flag token is validated on its own merits on
                // the next iteration.
                const next = args[i + 1];
                i += next !== undefined && !isFlagToken(next) ? 2 : 1;
                continue;
            }
            if (booleanFlagSet.has(name)) {
                i += 1;
                continue;
            }
            if (optionalValueFlagSet.has(name)) {
                const next = args[i + 1];
                i += next !== undefined && !isFlagToken(next) ? 2 : 1;
                continue;
            }
            const reason = flagList.length > 0
                ? `unknown flag ${formatDiagnosticToken(tok)}; accepted: ${flagList.join(', ')}`
                : `unknown flag ${formatDiagnosticToken(tok)}; this command accepts no flags`;
            return { ok: false, kind: 'InvalidArgs', arg: tok, reason, exitReason: ERROR_REASON.USAGE };
        }
        return {
            ok: false,
            kind: 'InvalidArgs',
            arg: tok,
            reason: `unexpected positional argument ${formatDiagnosticToken(tok)}`,
            exitReason: ERROR_REASON.USAGE,
        };
    }
    return { ok: true, data };
}
/**
 * Thin projection over `parseNamedArgs`: on `ok:false` it calls
 * `fail(result.reason, result.exitReason)` and then throws.
 *
 * The trailing throw exists for two reasons: (1) TypeScript's control-flow
 * analysis needs a `never`-returning path so callers can destructure the
 * return value without a null check; (2) it is a fail-closed backstop — the
 * `fail` callbacks in this repo are `never`-returning at runtime (`io.error`
 * calls `process.exit(1)`) but are typed `void`, so if a caller ever passes a
 * `fail` that actually returns, this still halts instead of falling through
 * with a half-built `ParsedNamedArgs`.
 */
function parseNamedArgsOrExit(args, spec, fail) {
    const result = parseNamedArgs(args, spec);
    if (!result.ok) {
        fail(result.reason, result.exitReason);
        throw new Error(`parseNamedArgsOrExit: fail() returned instead of exiting (arg: ${result.arg})`);
    }
    return result.data;
}
/**
 * Collect all tokens after --flag until the next --flag or end of args.
 */
function parseMultiwordArg(args, flag) {
    const idx = args.indexOf(`--${flag}`);
    if (idx === -1)
        return null;
    const tokens = [];
    for (let i = idx + 1; i < args.length; i++) {
        if (args[i].startsWith('--'))
            break;
        tokens.push(args[i]);
    }
    return tokens.length > 0 ? tokens.join(' ') : null;
}
