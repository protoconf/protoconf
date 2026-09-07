package utils

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// TestRegisterFileRacesRangeFiles forces the RegisterFile-against-enumeration
// interleaving that TestConcurrentCompile cannot reliably force on its own:
// TestConcurrentCompile's goroutines spend most of their time inside
// protoparse.ParseFiles with no lock held, so the window where one goroutine
// is inside filesResolver.RegisterFile while another is inside
// FindFileByPath/RangeFiles is a small fraction of the run. Here, two reader
// goroutines do nothing but call the locked accessors in a tight loop for the
// entire duration of 8 concurrent ParseOne calls, making that window nearly
// continuous — this is exactly 12-RESEARCH.md's "Pitfall 1: Trusting the doc
// comment instead of reading RegisterFile's body": a locally-constructed
// *protoregistry.Files gates its only internal lock on r == GlobalFiles, so
// it has zero synchronization of its own once it starts growing.
//
// Every read here goes through the DescriptorRegistry accessor methods
// (FindFileByPath, RangeFiles) rather than the unexported filesResolver
// field directly — reaching into the field would bypass d.mu and produce a
// race report about the test rather than about the code.
func TestRegisterFileRacesRangeFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, testdata.GenerateCorpus(dir, 20))
	src := filepath.Join(dir, "src")

	dr := NewDescriptorRegistry()
	dr.ImportPaths = []string{src}
	require.NotNil(t, dr.GetFilesResolver())

	done := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				dr.RangeFiles(func(protoreflect.FileDescriptor) bool { return true })
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				_, _ = dr.FindFileByPath("pkg0/msg0.proto")
			}
		}
	}()

	g := new(errgroup.Group)
	for i := 8; i < 16; i++ {
		i := i
		g.Go(func() error {
			_, err := dr.ParseOne(fmt.Sprintf("pkg%d/msg%d.proto", i, i))
			return err
		})
	}
	require.NoError(t, g.Wait())

	close(done)
	wg.Wait()

	for key := range dr.FileRegistry {
		_, err := dr.FindFileByPath(key)
		require.NoError(t, err, "FileRegistry key %s must resolve through the growable resolver after the race", key)
	}
	require.Equal(t, 0, dr.FilesResolverRegistrationErrorCount())
	require.Greater(t, dr.FilesResolverRegistrationCount(), 8)
}
