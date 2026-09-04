package testdata

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GenerateCorpus writes a deterministic synthetic proto corpus of n files
// under dir, plus a main.mpconf that loads exactly 5 of them (pkg0..pkg4).
//
// dir is caller-owned (pass t.TempDir()/b.TempDir()); GenerateCorpus does not
// create or clean up dir itself, and it does not create materialized_config/,
// protoconf.lock, CONFIGSPACE, or a git repo — NewCompiler needs none of
// those to compile main.mpconf.
//
// Each generated proto imports up to 3 earlier protos (by lower index) and
// declares a field of each imported message type, so linking has real
// cross-file symbol work to do (see BASELINE.md — linking is ~26% of the
// eager-registry profile). Two calls with the same n produce byte-identical
// output: the only randomness is a fixed-seed math/rand.Rand, consumed in a
// deterministic order.
func GenerateCorpus(dir string, n int) error {
	if n < 5 {
		return fmt.Errorf("GenerateCorpus: n must be >= 5 (main.mpconf loads pkg0..pkg4), got %d", n)
	}

	srcDir := filepath.Join(dir, "src")
	rng := rand.New(rand.NewSource(1))

	for i := 0; i < n; i++ {
		deps := pickDeps(rng, i)

		pkgDir := filepath.Join(srcDir, fmt.Sprintf("pkg%d", i))
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			return fmt.Errorf("GenerateCorpus: failed to create %s: %w", pkgDir, err)
		}
		filename := filepath.Join(pkgDir, fmt.Sprintf("msg%d.proto", i))
		if err := os.WriteFile(filename, []byte(protoFile(i, deps)), 0644); err != nil {
			return fmt.Errorf("GenerateCorpus: failed to write %s: %w", filename, err)
		}
	}

	mainPath := filepath.Join(srcDir, "main.mpconf")
	if err := os.WriteFile(mainPath, []byte(mainConfig()), 0644); err != nil {
		return fmt.Errorf("GenerateCorpus: failed to write %s: %w", mainPath, err)
	}

	return nil
}

// pickDeps chooses up to 3 distinct earlier indices (< i) that proto i will
// import. rng.Perm(i) already guarantees distinctness and excludes i itself,
// so no separate dedupe/self-import check is needed.
func pickDeps(rng *rand.Rand, i int) []int {
	k := 3
	if i < k {
		k = i
	}
	deps := rng.Perm(i)[:k]
	sort.Ints(deps)
	return deps
}

// protoFile renders proto i's source: proto3, package corpus.pkgN, a single
// message MsgN with a string field plus one field per entry in deps typed as
// the imported message.
func protoFile(i int, deps []int) string {
	var b strings.Builder
	b.WriteString("syntax = \"proto3\";\n\n")
	fmt.Fprintf(&b, "package corpus.pkg%d;\n\n", i)
	for _, j := range deps {
		fmt.Fprintf(&b, "import \"pkg%d/msg%d.proto\";\n", j, j)
	}
	if len(deps) > 0 {
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "message Msg%d {\n", i)
	b.WriteString("  string name = 1;\n")
	for idx, j := range deps {
		fmt.Fprintf(&b, "  corpus.pkg%d.Msg%d dep_%d = %d;\n", j, j, j, idx+2)
	}
	b.WriteString("}\n")
	return b.String()
}

// mainConfig renders main.mpconf: exactly 5 load() statements (pkg0..pkg4),
// regardless of n, so N=50 and N=5000 differ only in how many protos are
// present, never in how many are demanded.
func mainConfig() string {
	var b strings.Builder
	for i := 0; i < 5; i++ {
		fmt.Fprintf(&b, "load(\"//pkg%d/msg%d.proto\", \"Msg%d\")\n", i, i, i)
	}
	b.WriteString("\ndef main():\n")
	b.WriteString("    return {\n")
	for i := 0; i < 5; i++ {
		fmt.Fprintf(&b, "        \"out%d\": Msg%d(name=\"msg%d\"),\n", i, i, i)
	}
	b.WriteString("    }\n")
	return b.String()
}
