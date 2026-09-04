package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/require"
)

// TestGeneratedCorpusCompiles is a fixture test: it proves the generator
// emits a corpus the real compiler accepts, end-to-end through NewCompiler +
// CompileFile, with no lock file, no git repo, and no CONFIGSPACE marker. It
// passes before and after the lazy-loading milestone and is NOT a
// performance regression guard — see TestCompilerStartupScaling for that.
func TestGeneratedCorpusCompiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, testdata.GenerateCorpus(dir, 50))

	start := time.Now()
	c, err := NewCompiler(dir, false)
	elapsed := time.Since(start)
	require.NoError(t, err)

	require.NoError(t, c.CompileFile("main.mpconf"))

	entries, err := os.ReadDir(filepath.Join(dir, "materialized_config", "main"))
	require.NoError(t, err)
	require.NotEmpty(t, entries, "expected materialized output under materialized_config/main/")

	fileCount := len(c.ModuleService.GetProtoRegistry().FileRegistry)
	// Sanity-check the generator against BASELINE.md: a wildly
	// unrepresentative corpus (e.g. NewCompiler finishing in microseconds
	// with a handful of files) should be visible here immediately, rather
	// than surfacing later as a milestone planned against a fake number.
	t.Logf("NewCompiler took %s for corpus n=50, FileRegistry has %d files", elapsed, fileCount)
	require.GreaterOrEqual(t, fileCount, 50, "generated corpus should be reflected in the proto registry")
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

// TestCompilerStartupScaling is the only check in this file with teeth, and
// it has none *yet*. The bug is "cost scales with repository size, not
// config size": compiling a config that loads 5 protos should cost the same
// whether the repo contains 50 protos or 5,000. See BASELINE.md — parsed +
// linked 5 requested -> 7 total files in 2.7ms against a 799-proto repo that
// eagerly cost 4,639ms, a ~1,700x gap.
//
// This measures the allocation and wall-clock ratio between n=50 and n=400
// (not 5000: at ~4.6s per construction for 799 protos, n=400 already makes
// the ratio unambiguous without making `go test ./...` intolerable), logs
// both unconditionally, and gates on the allocation ratio only.
//
// Measure-then-skip is deliberate: the body still executes on every run, so
// it cannot silently rot into non-compiling or wrong code; it prints a live
// number today that doubles as the generator's sanity check; and it
// auto-greens the instant the fix lands. Deleting the t.Skipf below (turning
// this into require.LessOrEqual(t, allocRatio, maxRatio)) is the
// lazy-loading milestone's definition of done.
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

	if allocRatio > maxRatio {
		t.Skipf("compiler startup does not yet scale with config size, not repo size: "+
			"alloc ratio=%.2fx exceeds target %.1fx (see .planning/research/compiler-performance/BASELINE.md); "+
			"this Skip is the lazy-loading milestone's definition of done — delete it, turning this into "+
			"require.LessOrEqual(t, allocRatio, maxRatio), once the ratio is in bounds", allocRatio, maxRatio)
	}
}
