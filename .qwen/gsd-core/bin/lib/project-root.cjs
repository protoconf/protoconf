"use strict";
/**
 * Project-Root Resolution Module — resolves a project root from a starting
 * directory by walking the ancestor chain and applying five heuristics:
 *   (0) own .planning/ guard (#1362)
 *   (1) parent .planning/config.json sub_repos
 *   (2) legacy multiRepo: true + ancestor .git
 *   (3) .git heuristic with parent .planning/
 *   (4) nearest ancestor .planning/ (#1414, Resolution Provenance P1)
 * Bounded by FIND_PROJECT_ROOT_MAX_DEPTH ancestors. Sync I/O.
 *
 * ADR-457 build-at-publish: the hand-written bin/lib/project-root.cjs
 * collapsed to a TypeScript source of truth. Behaviour is preserved
 * byte-for-behaviour from the prior hand-written .cjs; only types are added.
 */
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.findProjectRoot = findProjectRoot;
exports.consentProjectRoot = consentProjectRoot;
const node_fs_1 = __importDefault(require("node:fs"));
const node_path_1 = __importDefault(require("node:path"));
const node_os_1 = __importDefault(require("node:os"));
const FIND_PROJECT_ROOT_MAX_DEPTH = 10;
function findProjectRoot(startDir) {
    let resolvedStart;
    try {
        resolvedStart = node_path_1.default.resolve(startDir);
    }
    catch {
        return startDir;
    }
    const fsRoot = node_path_1.default.parse(resolvedStart).root;
    const home = node_os_1.default.homedir();
    // If startDir already contains .planning/, it IS the project root.
    try {
        const ownPlanningDir = resolvedStart + node_path_1.default.sep + '.planning';
        if (node_fs_1.default.existsSync(ownPlanningDir) && node_fs_1.default.statSync(ownPlanningDir).isDirectory()) {
            return startDir;
        }
    }
    catch {
        // fall through
    }
    // Walk upward, mirroring isInsideGitRepo from the CJS reference.
    function isInsideGitRepo(candidateParent) {
        let d = resolvedStart;
        while (d !== fsRoot) {
            try {
                if (node_fs_1.default.existsSync(d + node_path_1.default.sep + '.git'))
                    return true;
            }
            catch {
                // ignore
            }
            if (d === candidateParent)
                break;
            const next = node_path_1.default.dirname(d);
            if (next === d)
                break;
            d = next;
        }
        return false;
    }
    // #2843: nearest ancestor (including `from` itself) that contains a `.git`,
    // bounded by `upTo` (exclusive). Returns the git-repo root, or null if none
    // exists before `upTo` / the filesystem root. Used to detect a NESTED child
    // repo whose root is strictly below a candidate ancestor `.planning/` — in
    // that case the caller's repo boundary sits between start and the ancestor,
    // so trusting the ancestor's `.planning/` would silently cross into a
    // different project. (No `git` subprocess — fs walk only, matching
    // isInsideGitRepo's deliberate no-spawn contract.)
    function nearestGitRoot(from, upTo) {
        let d = from;
        while (d !== fsRoot) {
            if (d === upTo)
                break;
            try {
                if (node_fs_1.default.existsSync(d + node_path_1.default.sep + '.git'))
                    return d;
            }
            catch {
                // ignore
            }
            const next = node_path_1.default.dirname(d);
            if (next === d)
                break;
            d = next;
        }
        return null;
    }
    let dir = resolvedStart;
    let depth = 0;
    while (dir !== fsRoot && depth < FIND_PROJECT_ROOT_MAX_DEPTH) {
        const parent = node_path_1.default.dirname(dir);
        if (parent === dir)
            break;
        if (parent === home)
            break;
        const parentPlanning = parent + node_path_1.default.sep + '.planning';
        let parentPlanningIsDir = false;
        try {
            parentPlanningIsDir = node_fs_1.default.existsSync(parentPlanning) && node_fs_1.default.statSync(parentPlanning).isDirectory();
        }
        catch {
            parentPlanningIsDir = false;
        }
        if (parentPlanningIsDir) {
            const configPath = parentPlanning + node_path_1.default.sep + 'config.json';
            let matched = false;
            try {
                const raw = node_fs_1.default.readFileSync(configPath, 'utf-8');
                const config = JSON.parse(raw);
                if (config && typeof config === 'object') {
                    const cfg = config;
                    const subReposValue = cfg['sub_repos'] ??
                        (cfg['planning'] && typeof cfg['planning'] === 'object'
                            ? cfg['planning']['sub_repos']
                            : undefined);
                    const subRepos = Array.isArray(subReposValue) ? subReposValue : [];
                    if (subRepos.length > 0) {
                        const relPath = node_path_1.default.relative(parent, resolvedStart);
                        const topSegment = relPath.split(node_path_1.default.sep)[0];
                        if (subRepos.includes(topSegment)) {
                            return parent;
                        }
                    }
                    if (cfg['multiRepo'] === true && isInsideGitRepo(parent)) {
                        matched = true;
                    }
                }
            }
            catch {
                // config.json missing or unparseable — fall through to .git heuristic.
            }
            if (matched)
                return parent;
            // Heuristic (3): parent has .planning/ and we're inside a git repo.
            // Before returning, check if any further ancestor has sub_repos that explicitly
            // claims our startDir — explicit sub_repos config takes precedence over the
            // implicit .git signal. (#1422)
            if (isInsideGitRepo(parent)) {
                // #2843: do NOT cross a git-repo boundary. If the caller is inside its
                // OWN nested repo whose root is strictly below `parent`, trusting
                // `parent`'s .planning/ would silently resolve to a DIFFERENT project.
                // isInsideGitRepo only proved SOME .git exists between start and parent;
                // verify that .git is parent's own (or absent between), not a nested
                // child repo. If nearestGitRoot finds a .git strictly below parent,
                // the boundary is crossed — fall through (do not return parent).
                if (nearestGitRoot(resolvedStart, parent) !== null) {
                    dir = parent;
                    depth += 1;
                    continue;
                }
                // Lookahead: walk ancestors above `parent` to find a sub_repos claim.
                let ancestor = node_path_1.default.dirname(parent);
                let ancestorDepth = 0;
                while (ancestor !== fsRoot && ancestor !== home && ancestorDepth < FIND_PROJECT_ROOT_MAX_DEPTH) {
                    const ancestorPlanning = ancestor + node_path_1.default.sep + '.planning';
                    try {
                        if (node_fs_1.default.existsSync(ancestorPlanning) && node_fs_1.default.statSync(ancestorPlanning).isDirectory()) {
                            const ancestorConfig = ancestor + node_path_1.default.sep + '.planning' + node_path_1.default.sep + 'config.json';
                            const rawA = node_fs_1.default.readFileSync(ancestorConfig, 'utf-8');
                            const cfgA = JSON.parse(rawA);
                            const subReposValueA = cfgA['sub_repos'] ??
                                (cfgA['planning'] && typeof cfgA['planning'] === 'object'
                                    ? cfgA['planning']['sub_repos']
                                    : undefined);
                            const subReposA = Array.isArray(subReposValueA) ? subReposValueA : [];
                            if (subReposA.length > 0) {
                                const relPathA = node_path_1.default.relative(ancestor, resolvedStart);
                                const topSegmentA = relPathA.split(node_path_1.default.sep)[0];
                                if (subReposA.includes(topSegmentA)) {
                                    return ancestor;
                                }
                            }
                        }
                    }
                    catch {
                        // ignore — config missing or unparseable, keep walking
                    }
                    const nextAncestor = node_path_1.default.dirname(ancestor);
                    if (nextAncestor === ancestor)
                        break;
                    ancestor = nextAncestor;
                    ancestorDepth += 1;
                }
                return parent;
            }
        }
        dir = parent;
        depth += 1;
    }
    // Heuristic (4): nearest ancestor .planning/ — last resort before fallback.
    // Runs only after heuristics (1)–(3) have been exhausted without a match,
    // ensuring sub_repos / multiRepo / .git-based resolution always wins when
    // applicable. Walks upward again within the same FIND_PROJECT_ROOT_MAX_DEPTH
    // bound; returns the nearest ancestor directory that contains a .planning/
    // subdirectory so config resolves correctly when invoked from a plain
    // descendant of a single-repo project. (#1414)
    let dir2 = resolvedStart;
    let depth2 = 0;
    while (dir2 !== fsRoot && depth2 < FIND_PROJECT_ROOT_MAX_DEPTH) {
        const parent2 = node_path_1.default.dirname(dir2);
        if (parent2 === dir2)
            break;
        try {
            const candidatePlanning = parent2 + node_path_1.default.sep + '.planning';
            if (node_fs_1.default.existsSync(candidatePlanning) && node_fs_1.default.statSync(candidatePlanning).isDirectory()) {
                // #2843: do not cross a git-repo boundary. If the caller is inside its
                // own nested repo (a .git strictly below parent2), parent2's .planning/
                // belongs to a DIFFERENT project — keep walking is wrong; stop and fall
                // through to the startDir fallback instead of silently resolving to the
                // ancestor project. (Reached only when no .git exists anywhere in the
                // chain per the triage, but guard defensively.)
                if (nearestGitRoot(resolvedStart, parent2) !== null) {
                    break;
                }
                return parent2;
            }
        }
        catch {
            // ignore fs errors and continue walking
        }
        if (parent2 === home)
            break;
        dir2 = parent2;
        depth2 += 1;
    }
    return startDir;
}
/**
 * #1459 (IC-01 / CB-4): THE single canonical derivation of the PROJECT ROOT used to bind/lookup a
 * project-scope consent record. Install (the CLI/lifecycle RECORD site), the loader (the LOOKUP
 * site), and `trust revoke` (CB-4) MUST all derive the consent root through this one helper so the
 * recorded key always matches the looked-up key — otherwise installing from a SUBDIR records consent
 * at `realpath(subdir)` while the loader looks it up at `realpath(findProjectRoot)` and the freshly
 * installed cap is immediately INACTIVE (install-then-inactive).
 *
 * The rule: `realpath(findProjectRoot(cwd))` (findProjectRoot is total — it returns `cwd` itself when
 * no project root is found, so there is no null branch), falling back to `path.resolve(cwd)` when the
 * resolved root cannot be realpath'd (e.g. it does not exist yet). The consent store realpaths
 * whatever it is given, so passing the SAME logical root from every site is what guarantees the match.
 */
function consentProjectRoot(cwd) {
    const root = findProjectRoot(cwd);
    try {
        return node_fs_1.default.realpathSync(root);
    }
    catch {
        return node_path_1.default.resolve(root);
    }
}
