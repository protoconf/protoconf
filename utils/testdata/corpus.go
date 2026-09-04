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

// Shape constants approximating a real provider schema. Terraform's AWS protos
// run 80-613 lines with ~19 top-level messages per file, ~25 fields each, and a
// [json_name = "..."] option on nearly every field.
//
// Earlier cuts of this generator emitted one 4-field message per file and were
// 47x cheaper per file than the real corpus in BASELINE.md; raising the field
// count alone left it 21x cheaper. The message count and the field options are
// what close the rest: options must be interpreted during linking, which is the
// cost this whole milestone is about. See
// .planning/research/compiler-performance/BASELINE.md.
const (
	messagesPerFile        = 12
	scalarFieldsPerMessage = 20
)

// protoFile renders proto i's source: proto3, package corpus.pkgN, several
// top-level messages each carrying a spread of scalar, repeated, map and enum
// fields with json_name options, a nested message, and — on the first message
// only — one message-typed field per entry in deps so the imports resolve.
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

	fmt.Fprintf(&b, "enum Kind%d {\n", i)
	fmt.Fprintf(&b, "  KIND%d_UNSPECIFIED = 0;\n", i)
	fmt.Fprintf(&b, "  KIND%d_PRIMARY = 1;\n", i)
	fmt.Fprintf(&b, "  KIND%d_SECONDARY = 2;\n", i)
	b.WriteString("}\n\n")

	scalars := []string{"string", "int64", "bool", "double", "int32", "bytes", "uint64", "float"}

	for m := 0; m < messagesPerFile; m++ {
		// Msg{i} is the entry point the .mpconf loads; the rest are bulk, the
		// way a provider file carries one resource plus its config types.
		name := fmt.Sprintf("Msg%d", i)
		if m > 0 {
			name = fmt.Sprintf("Msg%dPart%d", i, m)
		}
		fmt.Fprintf(&b, "// %s version is 0\n", name)
		fmt.Fprintf(&b, "message %s {\n", name)
		b.WriteString("  string name = 1;\n")

		tag := 2
		for f := 0; f < scalarFieldsPerMessage; f++ {
			fmt.Fprintf(&b, "  %s field_%d = %d [json_name = \"field_%d\"];\n",
				scalars[f%len(scalars)], f, tag, f)
			tag++
		}
		fmt.Fprintf(&b, "  repeated string tags = %d [json_name = \"tags\"];\n", tag)
		tag++
		fmt.Fprintf(&b, "  map<string, string> labels = %d [json_name = \"labels\"];\n", tag)
		tag++
		fmt.Fprintf(&b, "  Kind%d kind = %d;\n", i, tag)
		tag++
		fmt.Fprintf(&b, "  Nested%d nested = %d;\n", i, tag)
		tag++

		// Only the entry-point message carries the cross-file references; that
		// is enough for the imports to be used, and mirrors a provider file
		// where the resource type is the one referencing shared meta types.
		if m == 0 {
			for _, j := range deps {
				fmt.Fprintf(&b, "  corpus.pkg%d.Msg%d dep_%d = %d;\n", j, j, j, tag)
				tag++
			}
		}
		b.WriteString("}\n\n")
	}

	fmt.Fprintf(&b, "message Nested%d {\n", i)
	b.WriteString("  string inner_name = 1;\n")
	b.WriteString("  int64 inner_count = 2;\n")
	b.WriteString("  repeated string inner_tags = 3;\n")
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
