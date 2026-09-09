package lib

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/require"
)

// raceEnabled is flipped to true by race_detector_test.go's //go:build race
// init() when the test binary is built with -race. Declared here so the
// non-race build gets false by omission — no other file needs to know.
var raceEnabled bool

// TestGeneratedCorpusCompiles is a fixture test: it proves the generator
// emits a corpus the real compiler accepts, end-to-end through NewCompiler +
// CompileFile, with no lock file, no git repo, and no CONFIGSPACE marker. It
// also asserts the lazy registry's core promise (LAZY-01): compiling
// main.mpconf loads only its own transitive dependency graph, not the whole
// generated corpus. It is NOT a performance regression guard — see
// TestCompilerStartupScaling for that.
func TestGeneratedCorpusCompiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, testdata.GenerateCorpus(dir, 50))

	var onDiskProtos int
	require.NoError(t, filepath.WalkDir(filepath.Join(dir, "src"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".proto" {
			onDiskProtos++
		}
		return nil
	}))
	require.GreaterOrEqual(t, onDiskProtos, 50, "generator sanity: corpus should contain at least 50 .proto files on disk")

	start := time.Now()
	c, err := NewCompiler(dir, false)
	elapsed := time.Since(start)
	require.NoError(t, err)

	require.NoError(t, c.CompileFile("main.mpconf"))

	entries, err := os.ReadDir(filepath.Join(dir, "materialized_config", "main"))
	require.NoError(t, err)
	require.NotEmpty(t, entries, "expected materialized output under materialized_config/main/")

	loadedCount := c.ModuleService.GetProtoRegistry().LoadedFileCount()
	t.Logf("NewCompiler took %s for corpus n=50 (%d on-disk protos), compile loaded %d proto files", elapsed, onDiskProtos, loadedCount)
	require.Less(t, loadedCount, 50, "compile should not have loaded the whole corpus")
}

// compileCorpus builds a Compiler over dir and compiles main.mpconf. It is
// the measured operation shared by BenchmarkCompilerStartup and
// TestCompilerStartupScaling. Corpus generation is the caller's job and must
// happen before this is invoked, so it never contaminates the timed region.
func compileCorpus(dir string) error {
	c, err := NewCompiler(dir, false)
	if err != nil {
		return fmt.Errorf("compileCorpus: failed to create compiler: %w", err)
	}
	if err := c.CompileFile("main.mpconf"); err != nil {
		return fmt.Errorf("compileCorpus: failed to compile main.mpconf: %w", err)
	}
	return nil
}

// BenchmarkCompilerStartup reports NewCompiler+CompileFile cost as a function
// of repository size at a fixed config demand (5 protos, see
// GenerateCorpus). Run with `-benchtime=1x -benchmem`: at today's eager
// registry cost even one iteration at protos=5000 takes minutes (BASELINE.md:
// ~4.6s for 799 protos), so -benchtime=1x is the only sane way to run this
// case until the lazy-loading milestone lands.
//
// allocs/op is not optional here: BASELINE.md attributes ~40% of profile
// time to GC, so allocations move before ns/op does and are far less noisy.
func BenchmarkCompilerStartup(b *testing.B) {
	for _, n := range []int{50, 500, 5000} {
		if n == 5000 && testing.Short() {
			continue
		}
		b.Run(fmt.Sprintf("protos=%d", n), func(b *testing.B) {
			dir := b.TempDir()
			if err := testdata.GenerateCorpus(dir, n); err != nil {
				b.Fatalf("failed to generate corpus: %v", err)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := compileCorpus(dir); err != nil {
					b.Fatalf("compileCorpus failed: %v", err)
				}
			}
		})
	}
}

// TestCompilerStartupScaling asserts the bug this milestone fixed stays
// fixed: "cost scales with repository size, not config size" — compiling a
// config that loads 5 protos should cost the same whether the repo contains
// 50 protos or 5,000. See BASELINE.md — parsed + linked 5 requested -> 7
// total files in 2.7ms against a 799-proto repo that eagerly cost 4,639ms, a
// ~1,700x gap.
//
// This measures the allocation and wall-clock ratio between n=50 and n=400
// (not 5000: at ~4.6s per construction for 799 protos, n=400 already makes
// the ratio unambiguous without making `go test ./...` intolerable), logs
// both unconditionally, and gates on the allocation ratio only — wall-clock
// on a shared CI runner is not deterministic enough to assert on here; see
// TestCompilerStartupBudget, which asserts wall-clock against a
// CI-calibrated threshold in a dedicated non-race step instead.
//
// Measure-then-log is deliberate: the body still executes on every run, so
// it cannot silently rot into non-compiling or wrong code, and it prints a
// live number today that doubles as the generator's sanity check.
//
// The milestone's definition of done — this ratio at or below 2.0x — was
// reached in Phase 15: the assertion below now holds (require.LessOrEqual)
// where this test previously only measured and skipped.
func TestCompilerStartupScaling(t *testing.T) {
	if testing.Short() {
		t.Skip("scaling measurement is slow; skipped under -short")
	}

	const maxRatio = 2.0

	measure := func(n int) (elapsed time.Duration, allocBytes uint64, err error) {
		dir := t.TempDir()
		if genErr := testdata.GenerateCorpus(dir, n); genErr != nil {
			return 0, 0, fmt.Errorf("failed to generate corpus n=%d: %w", n, genErr)
		}

		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)

		start := time.Now()
		if compErr := compileCorpus(dir); compErr != nil {
			return 0, 0, fmt.Errorf("compileCorpus failed n=%d: %w", n, compErr)
		}
		elapsed = time.Since(start)

		runtime.ReadMemStats(&after)
		return elapsed, after.TotalAlloc - before.TotalAlloc, nil
	}

	elapsed50, alloc50, err := measure(50)
	require.NoError(t, err)
	elapsed400, alloc400, err := measure(400)
	require.NoError(t, err)

	allocRatio := float64(alloc400) / float64(alloc50)
	wallRatio := float64(elapsed400) / float64(elapsed50)

	// Gate on the allocation ratio, report the wall-clock. Allocation tracks
	// file count almost exactly and is deterministic/machine-independent; a
	// wall-clock ratio on a shared CI runner is not — per TESTING.md, a
	// threshold that flakes gets bumped until it is worthless.
	t.Logf("scaling n=50->400: alloc ratio=%.2fx (%d -> %d bytes), wall-clock ratio=%.2fx (%s -> %s)",
		allocRatio, alloc50, alloc400, wallRatio, elapsed50, elapsed400)

	require.LessOrEqual(t, allocRatio, maxRatio)
}

// corpusProtos sizes the corpus TestCompilerStartupBudget compiles. Sized so
// it costs what the real 799-proto protoconf-terraform corpus costs, not
// merely so it has 800 files: using the ~3x per-file multiplier GATE-05
// states, 799*3 rounds to 2400 (D-07). The ~3x figure is itself unsourced
// and taken at face value as an accepted risk (D-08) — exposure is low
// because measured startup cost is flat against corpus size (39ms at n=50,
// 35ms at n=500), so a wrong multiplier moves the file count without
// materially moving the gate's outcome.
const corpusProtos = 2400

// TestCompilerStartupBudget is GATE-02: a wall-clock assertion that
// compiling a calibrated-size corpus completes within a CI-observed budget.
// Unlike TestCompilerStartupScaling's allocation gate, wall clock cannot be
// measured meaningfully under the race detector (raceEnabled skip below) or
// under -short.
//
// It measures the same compileCorpus operation TestCompilerStartupScaling
// and BenchmarkCompilerStartup already measure (D-02), so the figure stays
// directly comparable to BASELINE.md's 6.97s breakdown. It does not run in
// parallel with other tests in the binary (no t.Parallel()): a wall-clock
// measurement sharing a runner with other tests measures contention, not the
// compiler.
// budget is calibrated from a real GitHub Actions observation, not from any
// number in CONTEXT.md, RESEARCH.md or BASELINE.md (all measured on
// darwin/arm64; GitHub-hosted ubuntu-latest runners are 2-core shared VMs
// and measure differently) — per D-03, picking a threshold from a document
// number instead of a CI run is the error this calibration step exists to
// avoid.
//
// Observed: ubuntu-latest, GitHub Actions run 34311638861 (2026-09-09),
// `startup budget: n=2400 compiled in 77.581263ms`. budget is set to
// roughly 2x that figure, rounded to a round number of milliseconds
// (D-03's 2x rule).
//
// GATE-02's text names a 200ms budget over an 800-proto corpus. This
// threshold (160ms) is tighter than 200ms and this corpus (2400) is larger
// than 800, so GATE-02 is satisfied strictly — do not "fix" this constant
// back up to 200.
const budget = 160 * time.Millisecond

func TestCompilerStartupBudget(t *testing.T) {
	if raceEnabled {
		t.Skip("wall-clock budget is meaningless under the race detector (~8x slowdown observed); see race_detector_test.go")
	}
	if testing.Short() {
		t.Skip("budget measurement is slow; skipped under -short")
	}

	dir := t.TempDir()
	require.NoError(t, testdata.GenerateCorpus(dir, corpusProtos))

	start := time.Now()
	require.NoError(t, compileCorpus(dir))
	elapsed := time.Since(start)

	t.Logf("startup budget: n=%d compiled in %s (budget %s)", corpusProtos, elapsed, budget)

	require.LessOrEqual(t, elapsed, budget)
}
