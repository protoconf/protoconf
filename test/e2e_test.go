package test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/protoconf/protoconf/agent"
	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
	"github.com/protoconf/protoconf/agent/dummykv"
	"github.com/protoconf/protoconf/agent/filekv"
	"github.com/protoconf/protoconf/compiler/lib"
	"github.com/protoconf/protoconf/consts"
	"github.com/protoconf/protoconf/inserter"
	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	"github.com/protoconf/protoconf/server"
	"github.com/protoconf/protoconf/utils"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

// makeTempScript creates a temp executable shell script with the given body (e.g. "exit 0" or "exit 1").
func makeTempScript(t *testing.T, body string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "test-script-*.sh")
	require.NoError(t, err)
	_, err = f.WriteString("#!/bin/sh\n" + body + "\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, os.Chmod(f.Name(), 0755))
	return f.Name()
}

// mustNewAny wraps a proto.Message in an anypb.Any, panicking on error.
func mustNewAny(msg proto.Message) *anypb.Any {
	a, _ := anypb.New(msg)
	return a
}

func TestMutationWithScripts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	protoconfRoot := testdata.SmallTestDir()
	preScript := makeTempScript(t, "exit 0")
	postScript := makeTempScript(t, "exit 0")

	srv, err := server.NewProtoconfMutationServer(protoconfRoot)
	require.NoError(t, err)
	srv.PreMutationScript = preScript
	srv.PostMutationScript = postScript

	var mutClient protoconf_pb.ProtoconfMutationServiceClient
	closer := TestServer(ctx, func(s *grpc.Server) {
		protoconf_pb.RegisterProtoconfMutationServiceServer(s, srv)
	}, func(conn *grpc.ClientConn) {
		mutClient = protoconf_pb.NewProtoconfMutationServiceClient(conn)
	})
	defer closer()

	resp, err := mutClient.MutateConfig(ctx, &protoconf_pb.ConfigMutationRequest{
		Path: "mutation_test",
		Value: &protoconf_pb.ProtoconfValue{
			ProtoFile: "google/protobuf/struct.proto",
			Value:     mustNewAny(structpb.NewStringValue("scripted mutation")),
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Uuid)
	assert.NotNil(t, resp.PreScriptDuration, "PreScriptDuration should be set when pre script runs")
	assert.NotNil(t, resp.PostScriptDuration, "PostScriptDuration should be set when post script runs")
}

// generateSelfSignedCert generates a self-signed ECDSA P-256 cert for 127.0.0.1 valid for 1 hour.
func generateSelfSignedCert(t *testing.T) (certPEM, keyPEM []byte) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"Test"}},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	require.NoError(t, err)
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyBytes, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
	return certPEM, keyPEM
}

func TestTLSMutation(t *testing.T) {
	protoconfRoot := testdata.SmallTestDir()
	certPEM, keyPEM := generateSelfSignedCert(t)

	// Server-side TLS
	serverTLS, err := utils.BuildTLSConfig(utils.TLSFiles{
		CertText: string(certPEM),
		KeyText:  string(keyPEM),
	})
	require.NoError(t, err)

	srv, err := server.NewProtoconfMutationServer(protoconfRoot)
	require.NoError(t, err)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	rpcServer := grpc.NewServer(grpc.Creds(credentials.NewTLS(serverTLS)))
	protoconf_pb.RegisterProtoconfMutationServiceServer(rpcServer, srv)
	go rpcServer.Serve(lis)
	defer rpcServer.Stop()

	// Client-side TLS
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(certPEM)
	clientTLS := &tls.Config{RootCAs: pool, ServerName: "127.0.0.1"}

	conn, err := grpc.NewClient(lis.Addr().String(),
		grpc.WithTransportCredentials(credentials.NewTLS(clientTLS)))
	require.NoError(t, err)
	defer conn.Close()

	client := protoconf_pb.NewProtoconfMutationServiceClient(conn)
	resp, err := client.MutateConfig(context.Background(), &protoconf_pb.ConfigMutationRequest{
		Path: "tls_test",
		Value: &protoconf_pb.ProtoconfValue{
			ProtoFile: "google/protobuf/struct.proto",
			Value:     mustNewAny(structpb.NewStringValue("tls value")),
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Uuid)
}

// makeTokenInterceptor returns a gRPC unary interceptor that validates Bearer tokens.
// Mirrors the bearerTokenInterceptor in server/server.go.
func makeTokenInterceptor(token string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if token == "" {
			return handler(ctx, req)
		}
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}
		values := md.Get("authorization")
		if len(values) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}
		tok := strings.TrimPrefix(values[0], "Bearer ")
		if subtle.ConstantTimeCompare([]byte(tok), []byte(token)) != 1 {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}
		return handler(ctx, req)
	}
}

func TestAuthFlow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	protoconfRoot := testdata.SmallTestDir()
	srv, err := server.NewProtoconfMutationServer(protoconfRoot)
	require.NoError(t, err)

	const secretToken = "test-secret-token-42"
	buffer := 1024 * 1024
	lis := bufconn.Listen(buffer)
	rpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(makeTokenInterceptor(secretToken)))
	protoconf_pb.RegisterProtoconfMutationServiceServer(rpcServer, srv)
	go func() {
		context.AfterFunc(ctx, func() { rpcServer.GracefulStop() })
		rpcServer.Serve(lis)
	}()
	defer func() { lis.Close(); rpcServer.Stop() }()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client := protoconf_pb.NewProtoconfMutationServiceClient(conn)
	mutReq := &protoconf_pb.ConfigMutationRequest{
		Path: "auth_test",
		Value: &protoconf_pb.ProtoconfValue{
			ProtoFile: "google/protobuf/struct.proto",
			Value:     mustNewAny(structpb.NewStringValue("auth value")),
		},
	}

	t.Run("valid_token_accepted", func(t *testing.T) {
		md := metadata.New(map[string]string{"authorization": "Bearer " + secretToken})
		authCtx := metadata.NewOutgoingContext(ctx, md)
		resp, err := client.MutateConfig(authCtx, mutReq)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotEmpty(t, resp.Uuid)
	})

	t.Run("invalid_token_rejected", func(t *testing.T) {
		md := metadata.New(map[string]string{"authorization": "Bearer wrong-token"})
		authCtx := metadata.NewOutgoingContext(ctx, md)
		_, err := client.MutateConfig(authCtx, mutReq)
		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("missing_token_rejected", func(t *testing.T) {
		_, err := client.MutateConfig(ctx, mutReq)
		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})
}

func Test(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Init
	protoconfRoot := testdata.SmallTestDir()
	t.Run("mod tidy", func(t *testing.T) {
		ms, err := lib.NewModuleService(protoconfRoot)
		require.NoError(t, err)
		require.NoError(t, ms.Init(ctx, "CONFIGSPACE"))
		require.NoError(t, ms.Sync(ctx))
	})

	c, err := lib.NewCompiler(protoconfRoot, false)
	require.NoError(t, err)
	// Create dev agent
	devConfig := &protoconf_agent_config.AgentConfig{
		Prefix: consts.CompiledConfigPath,
		Store:  protoconf_agent_config.AgentConfig_file,
	}
	devStore, err := filekv.New(ctx, []string{}, &filekv.Config{ProtoconfRoot: protoconfRoot})
	assert.NoError(t, err)

	devAgentServer, err := agent.NewProtoconfKVAgent(devStore, devConfig)
	var devAgentClient protoconf_pb.ProtoconfServiceClient
	assert.NoError(t, err)

	devMutationServer, err := server.NewProtoconfMutationServer(protoconfRoot)
	require.NoError(t, err)
	var devMutationClient protoconf_pb.ProtoconfMutationServiceClient
	devCloser := TestServer(ctx, func(s *grpc.Server) {
		protoconf_pb.RegisterProtoconfServiceServer(s, devAgentServer)
		protoconf_pb.RegisterProtoconfMutationServiceServer(s, devMutationServer)
	}, func(conn *grpc.ClientConn) {
		devAgentClient = protoconf_pb.NewProtoconfServiceClient(conn)
		devMutationClient = protoconf_pb.NewProtoconfMutationServiceClient(conn)
	})
	defer devCloser()
	assert.NoError(t, err)

	// Create production agent
	prodStore, err := dummykv.New(ctx, []string{}, &dummykv.Config{})
	assert.NoError(t, err)
	inserter := inserter.NewProtoconfInserter(protoconfRoot, prodStore)
	prodAgentServer, err := agent.NewProtoconfKVAgentRollout(prodStore, &protoconf_agent_config.AgentConfig{})
	assert.NoError(t, err)
	var prodAgentClient protoconf_pb.ProtoconfServiceClient
	prodCloser := TestServer(ctx, func(s *grpc.Server) {
		protoconf_pb.RegisterProtoconfServiceServer(s, prodAgentServer)
	}, func(conn *grpc.ClientConn) {
		prodAgentClient = protoconf_pb.NewProtoconfServiceClient(conn)
	})
	defer prodCloser()

	// Get first message from materialized_configs
	devWatcher, err := devAgentClient.SubscribeForConfig(ctx, &protoconf_pb.ConfigSubscriptionRequest{Path: "load_mutable_test"})
	expected, _ := anypb.New(structpb.NewStringValue("hello"))
	t.Run("get first message on devClient", func(t *testing.T) {
		assert.NoError(t, err)

		devConfigValue, err := devWatcher.Recv()
		assert.NoError(t, err)
		if !proto.Equal(devConfigValue.Value, expected) {
			t.Errorf("expected \n%s, got \n%s", expected, devConfigValue.Value)
		}
	})

	tCtx, _ := context.WithTimeout(ctx, 60*time.Second)
	prodWatcher, err := prodAgentClient.SubscribeForConfig(tCtx, &protoconf_pb.ConfigSubscriptionRequest{Path: "load_mutable_test"})
	assert.NoError(t, err)
	// Get first message from prodStore
	t.Run("get first message on prodClient", func(t *testing.T) {
		t.Run("insert to prodStore", func(t *testing.T) {
			err = inserter.InsertConfigFile("load_mutable_test" + consts.CompiledConfigExtension)
			assert.NoError(t, err)
		})
		prodConfigValue, err := prodWatcher.Recv()
		assert.NoError(t, err)
		if err != nil {
			return
		}

		if !proto.Equal(prodConfigValue.Value, expected) {
			t.Errorf("expected \n%s, got \n%s", expected, prodConfigValue.Value)
		}
	})
	// Change config via mutation rpc

	mutationValue, _ := anypb.New(structpb.NewStringValue("hello mutation"))
	t.Run("change config via mutation rpc", func(t *testing.T) {
		_, err = devMutationClient.MutateConfig(ctx, &protoconf_pb.ConfigMutationRequest{
			Path: "mutation_test", Value: &protoconf_pb.ProtoconfValue{
				ProtoFile: "google/protobuf/struct.proto",
				Value:     mutationValue,
			},
		})
		assert.NoError(t, err)
	})

	// Compile after mutation
	t.Run("compile after mutation", func(t *testing.T) {
		err = c.CompileFile("load_mutable_test.pconf")
		require.NoError(t, err)
	})

	// fetch update from watcher and validate the value after mutation
	t.Run("fetch update on devClient", func(t *testing.T) {
		devConfigValue, err := devWatcher.Recv()
		assert.NoError(t, err)
		if !proto.Equal(devConfigValue.Value, mutationValue) {
			t.Errorf("expected \n%s, got \n%s", mutationValue, devConfigValue.Value)
		}

	})

	devWatcher.CloseSend()

	t.Run("get update on prodClient", func(t *testing.T) {
		t.Run("insert to prodStore", func(t *testing.T) {
			err = inserter.InsertConfigFile("load_mutable_test" + consts.CompiledConfigExtension)
			assert.NoError(t, err)
		})
		prodConfigValue, err := prodWatcher.Recv()
		assert.NoError(t, err)
		if err != nil {
			return
		}

		require.Truef(t, proto.Equal(prodConfigValue.Value, mutationValue),
			"expected \n%s, got \n%s", mutationValue, prodConfigValue.Value)

	})
	prodWatcher.CloseSend()
	err = c.CompileFile("load_remote_with_load_local.pconf")
	assert.NoError(t, err)
	err = c.CompileFile("load_remote.pconf")
	assert.NoError(t, err)
	// devWatcher, err = devAgentClient.SubscribeForConfig(ctx, &protoconfservice.ConfigSubscriptionRequest{Path: "load_remote_with_load_local"})
	t.Run("load_remote on dev", func(t *testing.T) {
		devWatcher, err = devAgentClient.SubscribeForConfig(ctx, &protoconf_pb.ConfigSubscriptionRequest{Path: "load_remote"})
		assert.NoError(t, err)
		devConfigValue, err := devWatcher.Recv()
		assert.NoError(t, err)
		expected = &anypb.Any{TypeUrl: "type.googleapis.com/terraform.v1.Terraform"}
		if !proto.Equal(devConfigValue.Value, expected) {
			t.Errorf("expected \n%s, got \n%s", expected, devConfigValue.Value)
		}
		devWatcher.CloseSend()
	})
	t.Run("load_remote prod", func(t *testing.T) {
		newCtx, _ := context.WithTimeout(ctx, 10*time.Second)
		watcher, err := prodAgentClient.SubscribeForConfig(newCtx, &protoconf_pb.ConfigSubscriptionRequest{Path: "load_remote"})
		require.NoError(t, err)
		t.Run("insert load_remote to prod", func(t *testing.T) {
			require.NoError(t,
				inserter.InsertConfigFile("load_remote"+consts.CompiledConfigExtension),
			)
		})
		value, err := watcher.Recv()
		require.NoError(t, err)
		expected = &anypb.Any{TypeUrl: "type.googleapis.com/terraform.v1.Terraform"}
		if !proto.Equal(value.Value, expected) {
			t.Errorf("expected \n%s, got \n%s", expected, value.Value)
		}
		watcher.CloseSend()
	})
}
