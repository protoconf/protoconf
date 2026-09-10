"use strict";
/**
 * Update-context resolver (issue #498, candidate 3).
 *
 * ADR-457 build-at-publish: the hand-written bin/lib/update-context.cjs
 * collapsed to a TypeScript source of truth. Behaviour is preserved
 * byte-for-behaviour from the prior hand-written .cjs; only types are added.
 */
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.CODEX_CONFIG_MARKER = exports.RUNTIME_DIRS = void 0;
exports.inferPreferredRuntime = inferPreferredRuntime;
exports.envRuntimeDirs = envRuntimeDirs;
exports.resolveUpdateContext = resolveUpdateContext;
exports.loadUpdateContext = loadUpdateContext;
const node_path_1 = __importDefault(require("node:path"));
const node_fs_1 = __importDefault(require("node:fs"));
const node_os_1 = __importDefault(require("node:os"));
// Runtime -> candidate relative dir. Order matters: it is the probe order, and
// mirrors the RUNTIME_DIRS array the bash used (a runtime may have several
// candidate dirs). Kept here, not derived from the installer's getDirName,
// because update detection probes ALL historical dirs per runtime.
exports.RUNTIME_DIRS = [
    ['claude', '.claude'],
    ['opencode', '.config/opencode'],
    ['opencode', '.opencode'],
    ['antigravity', '.gemini/antigravity-ide'],
    ['antigravity', '.gemini/antigravity-cli'],
    ['antigravity', '.gemini/antigravity'],
    ['antigravity', '.agents'], // local Antigravity install dir canonical (#791; bin/install.js getDirName('antigravity'))
    ['antigravity', '.agent'], // local Antigravity install dir legacy (#503; backward-compat with pre-#791 installs)
    ['windsurf', '.windsurf'], // local Windsurf workflow dir canonical (#1615; bin/install.js getDirName('windsurf'))
    ['windsurf', '.devin'], // local Devin Desktop install dir legacy (#1085; backward-compat)
    ['kilo', '.config/kilo'],
    ['kilo', '.kilo'],
    ['codex', '.codex'],
];
const SEMVER_PREFIX = /^\d+\.\d+\.\d+/;
// Shared Codex config-root marker filename. Single-sourced here (this module
// is the pre-existing, dependency-free owner) and re-exported by
// host-runtime-detection.cts, whose detection rung reuses this exact probe
// filename (though NOT the truthiness rule below — see that module's header
// comment for the deliberate divergence).
exports.CODEX_CONFIG_MARKER = 'config.toml';
function expandHome(p, home) {
    if (!p)
        return '';
    return p.startsWith('~/') ? node_path_1.default.join(home, p.slice(2)) : p;
}
function versionFile(dir) { return node_path_1.default.join(dir, 'gsd-core', 'VERSION'); }
function markerFile(dir) { return node_path_1.default.join(dir, 'gsd-core', 'workflows', 'update.md'); }
// Detection: a dir "has GSD" if it carries a VERSION file or the update.md
// workflow marker.
function hasInstall(fs, dir) {
    return fs.exists(versionFile(dir)) || fs.exists(markerFile(dir));
}
// Read VERSION at dir; return a trimmed semver string, or null if missing/invalid.
function validVersionAt(fs, dir) {
    const raw = fs.readFile(versionFile(dir));
    if (raw == null)
        return null;
    const trimmed = String(raw).trim();
    return SEMVER_PREFIX.test(trimmed) ? trimmed : null;
}
// A version is TRUSTED only when BOTH the VERSION file and the update.md marker
// exist (and VERSION is valid semver).
function trustedVersionAt(fs, dir) {
    return dir && fs.exists(markerFile(dir)) ? validVersionAt(fs, dir) : null;
}
// Infer the preferred runtime from preferredConfigDir config files, then env.
function inferPreferredRuntime({ fs, env, preferredConfigDir }) {
    if (preferredConfigDir) {
        if (fs.exists(node_path_1.default.join(preferredConfigDir, 'kilo.json')) ||
            fs.exists(node_path_1.default.join(preferredConfigDir, 'kilo.jsonc')))
            return 'kilo';
        if (fs.exists(node_path_1.default.join(preferredConfigDir, 'opencode.json')) ||
            fs.exists(node_path_1.default.join(preferredConfigDir, 'opencode.jsonc')))
            return 'opencode';
        if (fs.exists(node_path_1.default.join(preferredConfigDir, exports.CODEX_CONFIG_MARKER)))
            return 'codex';
        const resolved = node_path_1.default.resolve(preferredConfigDir);
        const known = exports.RUNTIME_DIRS.find(([, reldir]) => resolved.endsWith(node_path_1.default.sep + reldir.split('/').join(node_path_1.default.sep)));
        if (known)
            return known[0];
    }
    if (env['CODEX_HOME'])
        return 'codex';
    if (env['ANTIGRAVITY_CONFIG_DIR'])
        return 'antigravity';
    if (env['KILO_CONFIG_DIR'] || env['KILO_CONFIG'])
        return 'kilo';
    if (env['OPENCODE_CONFIG_DIR'] || env['OPENCODE_CONFIG'])
        return 'opencode';
    if (env['CLAUDE_CONFIG_DIR'])
        return 'claude';
    return '';
}
// Absolute env-override candidates, mirroring the bash ENV_RUNTIME_DIRS block.
function envRuntimeDirs({ env, home }) {
    const out = [];
    const ex = (v) => expandHome(v, home);
    if (env['CLAUDE_CONFIG_DIR'])
        out.push(['claude', ex(env['CLAUDE_CONFIG_DIR'])]);
    if (env['ANTIGRAVITY_CONFIG_DIR'])
        out.push(['antigravity', ex(env['ANTIGRAVITY_CONFIG_DIR'])]);
    if (env['KILO_CONFIG_DIR'])
        out.push(['kilo', ex(env['KILO_CONFIG_DIR'])]);
    else if (env['KILO_CONFIG'])
        out.push(['kilo', node_path_1.default.dirname(ex(env['KILO_CONFIG']))]);
    else if (env['XDG_CONFIG_HOME'])
        out.push(['kilo', node_path_1.default.join(ex(env['XDG_CONFIG_HOME']), 'kilo')]);
    if (env['OPENCODE_CONFIG_DIR'])
        out.push(['opencode', ex(env['OPENCODE_CONFIG_DIR'])]);
    else if (env['OPENCODE_CONFIG'])
        out.push(['opencode', node_path_1.default.dirname(ex(env['OPENCODE_CONFIG']))]);
    else if (env['XDG_CONFIG_HOME'])
        out.push(['opencode', node_path_1.default.join(ex(env['XDG_CONFIG_HOME']), 'opencode')]);
    if (env['CODEX_HOME'])
        out.push(['codex', ex(env['CODEX_HOME'])]);
    return out;
}
// Stable reorder: entries whose runtime === preferred first, original order kept.
function preferFirst(entries, preferred) {
    const pref = entries.filter(([rt]) => rt === preferred);
    const rest = entries.filter(([rt]) => rt !== preferred);
    return [...pref, ...rest];
}
/**
 * Pure resolver. Returns { installedVersion, scope, runtime, gsdDir }.
 */
function resolveUpdateContext({ home, cwd, env = {}, fs, preferredConfigDir = '', preferredRuntime = '', }) {
    // Expand a leading `~/` before any probe.
    preferredConfigDir = expandHome(preferredConfigDir, home);
    const preferred = preferredRuntime || inferPreferredRuntime({ fs, env, preferredConfigDir });
    // Fast path: a validated preferredConfigDir (custom --config-dir install).
    if (preferredConfigDir && hasInstall(fs, preferredConfigDir)) {
        const resolvedPref = node_path_1.default.resolve(preferredConfigDir);
        let scope = 'GLOBAL';
        for (const [, reldir] of exports.RUNTIME_DIRS) {
            if (node_path_1.default.resolve(cwd, reldir) === resolvedPref) {
                scope = 'LOCAL';
                break;
            }
        }
        return {
            installedVersion: trustedVersionAt(fs, preferredConfigDir) ?? '0.0.0',
            scope,
            runtime: preferred,
            gsdDir: preferredConfigDir,
        };
    }
    const orderedEnv = preferFirst(envRuntimeDirs({ env, home }), preferred);
    const orderedRuntime = preferFirst(exports.RUNTIME_DIRS, preferred);
    // LOCAL probe (relative to cwd).
    let localRuntime = '', localDir = '';
    for (const [rt, reldir] of orderedRuntime) {
        const cand = node_path_1.default.resolve(cwd, reldir);
        if (hasInstall(fs, cand)) {
            localRuntime = rt;
            localDir = cand;
            break;
        }
    }
    // GLOBAL probe: absolute env candidates first, then $HOME-relative.
    let globalRuntime = '', globalDir = '';
    for (const [rt, absdir] of orderedEnv) {
        if (hasInstall(fs, absdir)) {
            globalRuntime = rt;
            globalDir = node_path_1.default.resolve(absdir);
            break;
        }
    }
    if (!globalRuntime) {
        for (const [rt, reldir] of orderedRuntime) {
            const cand = node_path_1.default.resolve(home, reldir);
            if (hasInstall(fs, cand)) {
                globalRuntime = rt;
                globalDir = cand;
                break;
            }
        }
    }
    const localValid = trustedVersionAt(fs, localDir);
    const isLocal = !!localValid && (!globalDir || localDir !== globalDir);
    if (isLocal) {
        return { installedVersion: localValid, scope: 'LOCAL', runtime: localRuntime, gsdDir: localDir };
    }
    const globalValid = trustedVersionAt(fs, globalDir);
    if (globalValid) {
        return { installedVersion: globalValid, scope: 'GLOBAL', runtime: globalRuntime, gsdDir: globalDir };
    }
    // A runtime dir was detected (VERSION or marker present) but is not a
    // complete, valid install: keep scope/runtime/dir and report 0.0.0 so the
    // caller re-installs.
    if (localRuntime && (!globalDir || localDir !== globalDir)) {
        return { installedVersion: '0.0.0', scope: 'LOCAL', runtime: localRuntime, gsdDir: localDir };
    }
    if (globalRuntime) {
        return { installedVersion: '0.0.0', scope: 'GLOBAL', runtime: globalRuntime, gsdDir: globalDir };
    }
    return { installedVersion: '0.0.0', scope: 'UNKNOWN', runtime: '', gsdDir: '' };
}
/**
 * CLI wiring: resolve against the real filesystem.
 */
function loadUpdateContext(opts = {}) {
    const fs = {
        exists: (p) => node_fs_1.default.existsSync(p),
        readFile: (p) => { try {
            return node_fs_1.default.readFileSync(p, 'utf8');
        }
        catch {
            return null;
        } },
    };
    return resolveUpdateContext({
        home: opts.home ?? node_os_1.default.homedir(),
        cwd: opts.cwd ?? process.cwd(),
        env: opts.env ?? process.env,
        fs,
        preferredConfigDir: opts.preferredConfigDir ?? '',
        preferredRuntime: opts.preferredRuntime ?? '',
    });
}
