package agent

import (
	"context"

	"google.golang.org/grpc/stats"
)

type noopStatsHandler struct{}

func (h *noopStatsHandler) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context {
	return ctx
}

func (h *noopStatsHandler) HandleRPC(_ context.Context, _ stats.RPCStats) {}

func (h *noopStatsHandler) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}

func (h *noopStatsHandler) HandleConn(_ context.Context, _ stats.ConnStats) {}
