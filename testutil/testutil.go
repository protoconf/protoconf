// Package testutil provides shared test helpers for use across all protoconf packages.
// It must not import any protoconf service-specific proto packages to avoid circular dependencies.
package testutil

import (
	"context"
	"log"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/protoconf/protoconf/utils/testdata"
)

// NewBufconnServer starts a gRPC server backed by an in-process bufconn listener.
// The register function is called with the grpc.Server so any service can be registered.
// Returns a *grpc.ClientConn connected to the server and a cleanup function that
// closes the listener and stops the server.
func NewBufconnServer(ctx context.Context, register func(*grpc.Server)) (*grpc.ClientConn, func()) {
	const buffer = 101024 * 1024
	lis := bufconn.Listen(buffer)
	srv := grpc.NewServer()
	register(srv)

	go func() {
		if err := srv.Serve(lis); err != nil {
			log.Printf("testutil: bufconn server error: %v", err)
		}
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Printf("testutil: error connecting to bufconn server: %v", err)
	}

	cleanup := func() {
		if err := lis.Close(); err != nil {
			log.Printf("testutil: error closing bufconn listener: %v", err)
		}
		srv.Stop()
	}

	return conn, cleanup
}

// NewAny wraps anypb.New and ignores the error. Test proto messages are always
// valid, so ignoring the error keeps test helper call-sites concise.
func NewAny(msg proto.Message) *anypb.Any {
	a, _ := anypb.New(msg)
	return a
}

// NewTestProtoconfRoot creates an isolated temporary protoconf root directory
// by copying the embedded SmallTestDir contents into a t.TempDir(). Each call
// returns a unique path so concurrent tests do not interfere.
func NewTestProtoconfRoot(t *testing.T) string {
	t.Helper()
	// SmallTestDir already creates a temp dir, initialises a git repo, and
	// returns the directory path; we can use it directly since it is already
	// isolated per invocation.
	return testdata.SmallTestDir()
}
