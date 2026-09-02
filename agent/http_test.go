package agent

import (
	"context"
	"encoding/base64"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kvtools/valkeyrie/store"
	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
	"github.com/protoconf/protoconf/agent/dummykv"
	protoconfvalue "github.com/protoconf/protoconf/datatypes/proto/v1"
	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// newHTTPTestServer builds the real agent mux (grpc server wrapped by the
// vanguard transcoder, alongside pprof/metrics) over a dummykv store and
// returns an httptest.Server plus the underlying store for seeding keys.
// Plain HTTP/1.1 is correct here -- no TLS and no EnableHTTP2 -- because
// vanguard rewrites the target request to HTTP/2 itself before handing it
// to the gRPC handler.
func newHTTPTestServer(t *testing.T) (*httptest.Server, store.Store) {
	t.Helper()
	ctx := context.Background()
	kvStore, err := dummykv.New(ctx, []string{}, &dummykv.Config{})
	require.NoError(t, err)

	agent, err := NewProtoconfKVAgent(kvStore, &protoconf_agent_config.AgentConfig{})
	require.NoError(t, err)

	rpcServer := grpc.NewServer()
	protoconf_pb.RegisterProtoconfServiceServer(rpcServer, agent)

	mux, err := newAgentMux(rpcServer)
	require.NoError(t, err)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, kvStore
}

func seedConfig(t *testing.T, kvStore store.Store, path string, value *protoconfvalue.ProtoconfValue, jsonBody []byte) {
	t.Helper()
	ctx := context.Background()
	b, err := proto.Marshal(value)
	require.NoError(t, err)
	require.NoError(t, kvStore.Put(ctx, path, []byte(base64.StdEncoding.EncodeToString(b)), &store.WriteOptions{}))
	if jsonBody != nil {
		require.NoError(t, kvStore.Put(ctx, path+"/config.json", jsonBody, &store.WriteOptions{}))
	}
}

func TestHTTP_GetConfig(t *testing.T) {
	srv, kvStore := newHTTPTestServer(t)

	wantJSON := []byte(`{"greeting":"hello http"}`)
	seedConfig(t, kvStore,
		"some/config",
		&protoconfvalue.ProtoconfValue{Value: newAny(structpb.NewStringValue("hello http"))},
		wantJSON,
	)

	t.Run("plain HTTP GET returns the seeded config.json verbatim", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/v1/config/some/config")
		require.NoError(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mediaType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
		require.NoError(t, err)
		assert.Equal(t, "application/json", mediaType)
		assert.Equal(t, wantJSON, body, "HTTP body must be the config.json bytes verbatim, not base64 or wrapped")
	})

	t.Run("gRPC GetConfig still returns value and now also raw", func(t *testing.T) {
		agent, err := NewProtoconfKVAgent(kvStore, &protoconf_agent_config.AgentConfig{})
		require.NoError(t, err)
		client, closer := testServer(context.Background(), agent)
		defer closer()

		got, err := client.GetConfig(context.Background(), &protoconf_pb.ConfigRequest{Path: "some/config"})
		require.NoError(t, err)
		assert.True(t, proto.Equal(got.Value, newAny(structpb.NewStringValue("hello http"))))
		require.NotNil(t, got.Raw)
		assert.Equal(t, "application/json", got.Raw.ContentType)
		assert.Equal(t, wantJSON, got.Raw.Data)
	})

	t.Run("GET for a missing config returns 404", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/v1/config/does/not/exist")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("/metrics still resolves to the Prometheus handler with the transcoder mounted at /", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/metrics")
		require.NoError(t, err)
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, string(body), "# HELP", "expected Prometheus exposition format from the metrics handler, not the transcoder")
	})

	t.Run("absent config.json sibling: gRPC succeeds with nil raw, HTTP GET is 200 with an empty body", func(t *testing.T) {
		// ponytail: this pins the accepted D-03 ceiling -- a missing config.json
		// sibling degrades to an empty HTTP body rather than a 404 or 500.
		seedConfig(t, kvStore,
			"no/json/sibling",
			&protoconfvalue.ProtoconfValue{Value: newAny(structpb.NewStringValue("no sibling"))},
			nil,
		)

		agent, err := NewProtoconfKVAgent(kvStore, &protoconf_agent_config.AgentConfig{})
		require.NoError(t, err)
		client, closer := testServer(context.Background(), agent)
		defer closer()

		got, err := client.GetConfig(context.Background(), &protoconf_pb.ConfigRequest{Path: "no/json/sibling"})
		require.NoError(t, err)
		assert.True(t, proto.Equal(got.Value, newAny(structpb.NewStringValue("no sibling"))))
		assert.Nil(t, got.Raw)

		resp, err := http.Get(srv.URL + "/v1/config/no/json/sibling")
		require.NoError(t, err)
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Empty(t, body)
	})
}
