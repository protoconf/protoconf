package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/protoconf/protoconf/consts"
	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
)

// mutateRaceClientCount is the number of concurrent MutateConfig clients
// driven against one shared mutation server in
// TestMutateConfigConcurrentClientsAreRaceFree.
const mutateRaceClientCount = 16

// TestMutateConfigConcurrentClientsAreRaceFree proves SAFE-02 for the
// mutation server (D-07 e2e half): mutateRaceClientCount concurrent gRPC
// clients call MutateConfig over one shared bufconn connection, through the
// real bearerTokenInterceptor and the real gRPC codec, against one
// long-lived server with no compiler attached (so MutateConfig's marshal
// resolves in.Value through s.parser.TypeResolver and writes the file --
// the lazy-registry path SAFE-02 is about). A background goroutine calls
// GenReflectionUI in a loop for the duration of the writer wave, so the
// periodic reflection walk overlaps in-flight mutations -- the realistic
// production concurrency shape (server.go's own 5-second ticker runs
// GenReflectionUI alongside live MutateConfig traffic).
func TestMutateConfigConcurrentClientsAreRaceFree(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	protoconfRoot := testdata.SmallTestDir()
	srv, err := NewProtoconfMutationServer(protoconfRoot)
	require.NoError(t, err)

	const secretToken = "mutate-race-secret-token"
	buffer := 1024 * 1024
	lis := bufconn.Listen(buffer)
	rpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(bearerTokenInterceptor(secretToken)))
	srv.Init(rpcServer)
	go func() {
		context.AfterFunc(ctx, func() { rpcServer.GracefulStop() })
		_ = rpcServer.Serve(lis)
	}()
	t.Cleanup(func() { lis.Close(); rpcServer.Stop() })

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	client := protoconf_pb.NewProtoconfMutationServiceClient(conn)

	// Resolve a custom type from the fixture's src/ through the server's own
	// tiered resolver, so the request's Any genuinely routes through the
	// on-demand path this plan is proving, rather than a well-known type
	// already seeded at construction.
	mt, err := srv.parser.TypeResolver.FindMessageByURL("type.googleapis.com/test.v1.TestMessage")
	require.NoError(t, err)
	stringValueField := mt.Descriptor().Fields().ByName("stringValue")
	require.NotNil(t, stringValueField)

	md := metadata.New(map[string]string{"authorization": "Bearer " + secretToken})
	authCtx := metadata.NewOutgoingContext(ctx, md)

	// Reflection-walk goroutine: overlaps the writer wave below, started
	// before errgroup.Wait() and stopped after it. GenReflectionUI's
	// returned error is expected to be non-nil (SmallTestDir's mutable_config/
	// carries intentionally-broken fixtures), so it is deliberately not
	// asserted -- only that the run completes and the race detector is quiet.
	reflectionStop := make(chan struct{})
	reflectionDone := make(chan struct{})
	go func() {
		defer close(reflectionDone)
		for {
			select {
			case <-reflectionStop:
				return
			default:
				_ = srv.GenReflectionUI(ctx, grpc.NewServer(), &http.Server{})
			}
		}
	}()

	g, gctx := errgroup.WithContext(authCtx)
	for i := 0; i < mutateRaceClientCount; i++ {
		i := i
		g.Go(func() error {
			msg := dynamicpb.NewMessage(mt.Descriptor())
			msg.Set(stringValueField, protoreflect.ValueOfString(fmt.Sprintf("value-%d", i)))
			anyVal, err := anypb.New(msg)
			if err != nil {
				return fmt.Errorf("client %d: anypb.New: %w", i, err)
			}
			path := fmt.Sprintf("mutate_race_%d", i)
			resp, err := client.MutateConfig(gctx, &protoconf_pb.ConfigMutationRequest{
				Path: path,
				Value: &protoconf_pb.ProtoconfValue{
					ProtoFile: "test.proto",
					Value:     anyVal,
				},
			})
			if err != nil {
				return fmt.Errorf("client %d: MutateConfig: %w", i, err)
			}
			if resp == nil {
				return fmt.Errorf("client %d: nil response", i)
			}
			return nil
		})
	}
	require.NoError(t, g.Wait())

	close(reflectionStop)
	<-reflectionDone

	// 16 distinct files must exist under mutable_config/ -- a silently
	// dropped mutation cannot pass as success.
	entries, err := os.ReadDir(filepath.Join(protoconfRoot, consts.MutableConfigPath))
	require.NoError(t, err)
	seen := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), "mutate_race_") {
			seen[e.Name()] = true
		}
	}
	require.Len(t, seen, mutateRaceClientCount, "expected %d distinct mutation files, got %v", mutateRaceClientCount, seen)
}
