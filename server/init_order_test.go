package server

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/protoconf/protoconf/consts"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// alphaServiceProto and betaServiceProto each declare one custom mutation
// service, in separate files and separate packages. Two files is the point:
// Init discovers services by ranging discoveryFiles (a map), and Go randomizes
// map iteration order per range loop, so repeated Init calls in one process
// naturally sample different orders. With only a single service — as
// utils/testdata/small/src/test.proto provides — the registered set has one
// element and is order-independent trivially, so a test built on it would pass
// regardless of what Init did (the same "green without exercising the claim"
// failure mode code review flagged as WR-05).
const alphaServiceProto = `syntax = "proto3";
package uat.alpha.v1;

import "protoconf/v1/protoconf.proto";

message AlphaMessage {
    string value = 1;
}

service AlphaService {
    rpc PutAlpha(AlphaMessage) returns (protoconf.v1.ConfigMutationResponse);
}
`

const betaServiceProto = `syntax = "proto3";
package uat.beta.v1;

import "protoconf/v1/protoconf.proto";

message BetaMessage {
    string value = 1;
}

service BetaService {
    rpc PutBeta(BetaMessage) returns (protoconf.v1.ConfigMutationResponse);
}
`

// twoServiceRoot writes a throwaway protoconf root holding exactly the two
// service protos above. It deliberately does not reuse utils/testdata/small —
// adding a second .proto there would change a fixture many other packages
// compile against.
func twoServiceRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	srcDir := filepath.Join(root, consts.SrcPath)
	require.NoError(t, os.MkdirAll(srcDir, 0o755))

	for name, body := range map[string]string{
		"alpha_service.proto": alphaServiceProto,
		"beta_service.proto":  betaServiceProto,
	} {
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, name), []byte(body), 0o644))
	}

	return root
}

// registeredServiceNames runs Init against a fresh grpc.Server and returns the
// registered service names, sorted so the comparison is over the SET rather
// than the order — varying order is expected and allowed; varying membership is
// the bug.
func registeredServiceNames(t *testing.T, protoconfRoot string) []string {
	t.Helper()

	s, err := NewProtoconfMutationServer(protoconfRoot)
	require.NoError(t, err)

	rpcServer := grpc.NewServer()
	require.NotPanics(t, func() { s.Init(rpcServer) })

	names := make([]string, 0, len(rpcServer.GetServiceInfo()))
	for name := range rpcServer.GetServiceInfo() {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}

// TestInitServiceSetIsOrderIndependent covers plan 11-02's backstop truth: the
// set of services Init registers must not depend on the map iteration order of
// discoveryFiles.RangeFiles. Only registration ORDER may vary between runs,
// never WHICH services end up registered.
//
// A genuinely order-dependent Init surfaces loudly here rather than subtly:
// grpc.Server.RegisterService panics on a duplicate service name, and a service
// dropped on some orderings shows up as a set-membership difference.
func TestInitServiceSetIsOrderIndependent(t *testing.T) {
	root := twoServiceRoot(t)

	first := registeredServiceNames(t, root)

	// Both custom services must actually be discovered — otherwise this test
	// would "pass" against an Init that found neither, proving nothing.
	require.Subset(t, first, []string{"uat.alpha.v1.AlphaService", "uat.beta.v1.BetaService"},
		"both fixture services must be discovered, or the order-independence assertion below is vacuous")

	// Go reshuffles map iteration per range loop, so repeated Init calls in
	// this one process sample different orders without any fault injection.
	const runs = 20
	for i := 1; i < runs; i++ {
		require.Equal(t, first, registeredServiceNames(t, root),
			"registered service set changed between Init runs (run %d of %d) — discovery is order-dependent", i+1, runs)
	}
}
