//go:build race

package lib

// This file exists because the wall-clock budget asserted by
// TestCompilerStartupBudget (startup_bench_test.go) is meaningless under the
// race detector: a 400-proto corpus measured 34ms plain and 268ms under
// -race, an ~8x factor. Both the CI "Run coverage" step and the pinned local
// command `go test -race -timeout 300s -skip 'Test_cliCommand_Run' ./...`
// run this whole package under -race, so TestCompilerStartupBudget needs to
// know it is running under the race detector in order to self-skip.
//
// raceEnabled is declared (false) in startup_bench_test.go for the
// non-race build; this build-tag-constrained file flips it to true only
// when the binary is built with -race.
func init() {
	raceEnabled = true
}
