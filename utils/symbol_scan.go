package utils

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// scanCandidateLimit bounds the per-lookup parse count the scan tier (Tier
// 2, D-01/D-05) will pay before giving up and letting the caller escalate to
// the symbol index (13-02).
//
// Cost of one candidate: one ParseOne on a corpus-scale file, ~5ms
// (measured, D-01).
//
// Cold break-even -- the number of candidates at which paying to parse them
// one at a time costs as much as building the index outright: cold index
// cost is ~30ms dirhash.HashDir (measured, plan <verified_facts> fact 5)
// plus ~1,286ms ParseFilesButDoNotLink over the 799-proto corpus
// (OPTIONS.md) = ~1,316ms. 1316ms / 5ms = ~263 candidates.
//
// Warm break-even -- the same ~30ms hash plus a read of the persisted index
// file. That read is NOT measured: 13-02 is the plan that creates the
// on-disk artifact (a ~138K-entry line-oriented file), so it did not exist
// when this constant was derived. Counting the read as zero gives a warm
// break-even FLOOR of 30ms / 5ms = ~6 candidates; the true figure is 6 plus
// one fifth of the read's cost in milliseconds. This comment records the
// floor, not a single warm break-even number, until 13-02 measures the read.
//
// 32 is therefore NOT a break-even point -- it sits between the two figures
// above by construction. It is a worst-case bound on a single lookup's parse
// burst: 32 x ~5ms = ~160ms, roughly an eighth of a cold index build, far
// below the cold break-even (~263) so a repo with no index never parses more
// than it would have spent building one, and deliberately above the warm
// floor (>=6) so the limit fires only in the pathological many-declarations
// case (the measured 351-file "Timeouts" shape, D-05) rather than on a
// handful of near misses.
const scanCandidateLimit = 32

// declPattern returns the nesting-tolerant declaration pattern for sym: any
// amount of leading horizontal whitespace, then "message" or "enum", then
// sym, then "{". Anchoring on "^message" alone (no leading-whitespace
// allowance) finds only 3.1% of symbols on the D-05 benchmark corpus,
// because 96.9% of message declarations are indented nested types.
func declPattern(sym string) *regexp.Regexp {
	return regexp.MustCompile(`(?m)^[ \t]*(message|enum)[ \t]+` + regexp.QuoteMeta(sym) + `[ \t]*\{`)
}

// splitSymbolPackage splits a fully-qualified proto symbol name into its
// package prefix and the remaining name segments. Proto package names are
// conventionally all-lowercase and message/enum names are PascalCase, so the
// package prefix is the longest leading run of dot-separated segments whose
// first rune is not an uppercase letter -- the first PascalCase segment
// marks the start of the (possibly nested) type name. For
// "nested.v1.Outer.Middle.Inner" that is package "nested.v1", rest
// ["Outer", "Middle", "Inner"].
func splitSymbolPackage(fullName string) (pkg string, rest []string) {
	if fullName == "" {
		return "", nil
	}
	segments := strings.Split(fullName, ".")
	i := 0
	for ; i < len(segments); i++ {
		if segments[i] == "" {
			continue
		}
		if unicode.IsUpper([]rune(segments[i])[0]) {
			break
		}
	}
	return strings.Join(segments[:i], "."), segments[i:]
}

// symbolScanCandidates returns the root-relative paths (matching what find
// and ParseOne expect) of every .proto file under roots whose bytes match
// fullName's top-level (first non-package) segment's nesting-tolerant
// declaration pattern. A lexical hit here is a CANDIDATE, never an answer
// (D-05, T-13-02): the caller must still ParseOne it and re-check the
// registry before trusting it.
//
// For each root, when fullName's package prefix maps to an existing
// directory under that root, only that subtree is scanned; otherwise the
// whole root is scanned. filepath.WalkDir does not follow symlinks (T-13-03).
// Results are de-duplicated, sorted for a stable order across calls (so two
// scans of an unchanged tree return an identical slice), and capped at
// scanCandidateLimit: exceeding it returns nil so the caller escalates
// rather than paying for a pathological number of parses (T-13-04).
func symbolScanCandidates(roots []string, fullName string) []string {
	pkg, rest := splitSymbolPackage(fullName)
	if len(rest) == 0 || rest[0] == "" {
		return nil
	}
	pattern := declPattern(rest[0])

	seen := map[string]struct{}{}
	var candidates []string
	for _, root := range roots {
		scanRoot := root
		if pkg != "" {
			pkgDir := filepath.Join(root, filepath.FromSlash(strings.ReplaceAll(pkg, ".", "/")))
			if info, err := os.Stat(pkgDir); err == nil && info.IsDir() {
				scanRoot = pkgDir
			}
		}
		_ = filepath.WalkDir(scanRoot, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d == nil || d.IsDir() {
				return nil
			}
			if filepath.Ext(d.Name()) != ".proto" {
				return nil
			}
			b, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			if !pattern.Match(b) {
				return nil
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)
			if _, ok := seen[rel]; !ok {
				seen[rel] = struct{}{}
				candidates = append(candidates, rel)
			}
			return nil
		})
	}
	sort.Strings(candidates)
	if len(candidates) > scanCandidateLimit {
		return nil
	}
	return candidates
}

// LoadSymbolByScan is the scan tier (Tier 2, D-01): a scoped lexical
// candidate search over d.ImportPaths, each candidate confirmed by a real
// ParseOne plus a MessageRegistry re-check before it is trusted (D-05,
// T-13-02) -- a lexical match alone is never an answer. Returns false
// immediately when the registry has no ImportPaths (every eager registry,
// D-03).
//
// Lock discipline (Phase 11): d.mu is never held across symbolScanCandidates
// or ParseOne -- ParseOne's own LookupImport closure takes RLock from inside
// protoparse, and utils/parse_all_deadlock_test.go exists because of that
// inversion.
func (d *DescriptorRegistry) LoadSymbolByScan(fullName string) bool {
	d.mu.RLock()
	importPaths := d.ImportPaths
	d.mu.RUnlock()
	if len(importPaths) == 0 {
		return false
	}

	url := "type.googleapis.com/" + fullName
	for _, candidate := range symbolScanCandidates(importPaths, fullName) {
		if _, err := d.ParseOne(candidate); err != nil {
			continue
		}
		if md, mErr := d.MessageRegistry.FindMessageTypeByUrl(url); mErr == nil && md != nil {
			d.mu.Lock()
			d.scanResolutions++
			d.mu.Unlock()
			return true
		}
	}
	return false
}

// ScanResolutionCount reports how many times the scan tier (LoadSymbolByScan)
// has answered a lookup by confirming a candidate. Test-only observable, no
// log line, no CLI surface. Safe to call concurrently with ParseOne.
func (d *DescriptorRegistry) ScanResolutionCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.scanResolutions
}
