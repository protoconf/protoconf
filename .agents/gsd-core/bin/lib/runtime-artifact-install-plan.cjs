'use strict';
/**
 * Runtime Artifact Install Plan Module.
 *
 * Turns a pre-resolved runtime artifact layout into staged copy inputs. The
 * installer adapter still owns pruning, copying, migrations, output, and final
 * cleanup execution.
 */
// In .cts (CommonJS output) files, `require` is available as a global.
const _require = require;
const path = _require('node:path');
// #2870: InstallScope is owned by install-scope.cts, not re-declared here.
// `isGlobalScope` centralizes the `scope === 'global'` boolean projection
// this module needs at `_computePathPrefix`'s `isGlobal: boolean` boundary
// (see the module-level doc comment on `isGlobalScope` for why the
// projection is centralized rather than eliminated).
const install_scope_cjs_1 = require("./install-scope.cjs");
/**
 * Asserts that `destSubpath` resolves to a path inside `configDir`.
 *
 * Rejects any path that escapes the configDir root (e.g. "../../etc") and any
 * path containing a NUL byte. This is a security gate for Phase B of
 * ADR-1239: third-party descriptors must never be able to write outside the
 * designated config home directory.
 *
 * @param configDir - The root config directory (e.g. .agents).
 * @param destSubpath - The relative path declared by the runtime descriptor.
 * @returns The resolved absolute path under configDir.
 * @throws {Error} if destSubpath escapes configDir or contains a NUL byte.
 */
function assertDestWithinConfigHome(configDir, destSubpath) {
    if (destSubpath.includes('\0')) {
        throw new Error(`destSubpath "${destSubpath}" contains a NUL byte and is not valid`);
    }
    const root = path.resolve(configDir);
    const resolved = path.resolve(configDir, destSubpath);
    if (resolved === root || !resolved.startsWith(root + path.sep)) {
        throw new Error(`destSubpath "${destSubpath}" must be a strict subpath of configHome "${configDir}" — not configHome itself or outside it (escapes configHome)`);
    }
    return resolved;
}
function errorMessage(err) {
    if (err instanceof Error)
        return err.message;
    return String(err);
}
function addCleanupDir(cleanupDirs, stagedDir, rewrittenDir) {
    const sourceDir = rewrittenDir ?? stagedDir;
    if (sourceDir !== stagedDir)
        cleanupDirs.push(sourceDir);
    return sourceDir;
}
function createRuntimeArtifactInstallPlan(args) {
    const { layout, resolvedProfile, homedir, platform, resolveAttribution, projectDir, deps = {}, } = args;
    const conversionExports = _require('./runtime-artifact-conversion.cjs');
    const rewriteStagedSkillBodies = deps.rewriteStagedSkillBodies ?? conversionExports.rewriteStagedSkillBodies;
    const rewriteStagedCommandBodies = deps.rewriteStagedCommandBodies ?? conversionExports.rewriteStagedCommandBodies;
    const cleanupDirs = [];
    const items = [];
    const scope = layout.scope ?? 'global';
    const rewriteOpts = {
        runtime: layout.runtime,
        configDir: layout.configDir,
        scope,
        homedir,
        platform,
        resolveAttribution,
    };
    // ADR-1235 §1: build the staging context once per plan. Agent kinds apply
    // the CORRECT pre-converter cross-cutting (path rewrites → attribution →
    // converter → normalize). This
    // mirrors the exact per-file order in the former inline agent loop.
    // NO _stampNonClaudeRuntimeDefaults — agents are NOT stamped in the inline loop.
    const os = _require('node:os');
    const { posixNormalize } = _require('./shell-command-projection.cjs');
    const homedirFn = homedir ?? (() => os.homedir());
    const resolvedTarget = posixNormalize(path.resolve(layout.configDir));
    const homeDir = posixNormalize(homedirFn());
    // #2870: `scope` above is already the module-owned `InstallScope` value
    // (`layout.scope ?? 'global'`, defaulted before this point, so it is never
    // `undefined` here) — `isGlobalScope` projects it to the boolean
    // `_computePathPrefix`'s existing `isGlobal: boolean` API requires.
    const isGlobal = (0, install_scope_cjs_1.isGlobalScope)(scope);
    const isOpencode = layout.runtime === 'opencode';
    const isWindowsHost = (platform ?? process.platform) === 'win32';
    const pathPrefix = conversionExports._computePathPrefix({ isGlobal, isOpencode, isWindowsHost, resolvedTarget, homeDir });
    const attribution = resolveAttribution ? resolveAttribution(layout.runtime) : undefined;
    // #2875 Part 2 (row I1): layout.configDir IS the install root the inline
    // agent loop called `targetDir` — same value, same resolution.
    const agentCtx = {
        runtime: layout.runtime,
        pathPrefix,
        attribution,
        targetDir: layout.configDir,
        projectDir: projectDir ?? layout.configDir,
    };
    for (const kind of layout.kinds) {
        let stagedDir;
        try {
            // Agent kinds use the context for their pre-converter cross-cutting
            // sequence; other kinds ignore it.
            stagedDir = kind.stage(resolvedProfile, agentCtx);
        }
        catch (err) {
            return { ok: false, kind: 'stage_failed', message: errorMessage(err), cleanupDirs, failedKind: kind.kind };
        }
        let sourceDir = stagedDir;
        try {
            if (kind.kind === 'commands') {
                const rewrittenDir = rewriteStagedCommandBodies(stagedDir, rewriteOpts);
                sourceDir = addCleanupDir(cleanupDirs, stagedDir, rewrittenDir);
            }
            else if (kind.kind === 'skills' || kind.kind === 'kimi-agents') {
                const rewrittenDir = rewriteStagedSkillBodies(stagedDir, rewriteOpts);
                sourceDir = addCleanupDir(cleanupDirs, stagedDir, rewrittenDir);
            }
            // Agent kinds: cross-cutting already applied INSIDE kind.stage() via agentCtx.
            // No POST-step needed. sourceDir stays as stagedDir.
        }
        catch (err) {
            return { ok: false, kind: 'rewrite_failed', message: errorMessage(err), cleanupDirs, failedKind: kind.kind };
        }
        items.push({
            kind: kind.kind,
            sourceDir,
            destDir: assertDestWithinConfigHome(kind.home ?? layout.configDir, kind.destSubpath),
        });
    }
    return { ok: true, plan: { items, cleanupDirs } };
}
function createRuntimeArtifactUninstallPlan(layout) {
    return {
        items: layout.kinds.map((kind) => ({
            kind: kind.kind,
            destDir: assertDestWithinConfigHome(kind.home ?? layout.configDir, kind.destSubpath),
        })),
    };
}
module.exports = { assertDestWithinConfigHome, createRuntimeArtifactInstallPlan, createRuntimeArtifactUninstallPlan };
