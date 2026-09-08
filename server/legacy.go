package server

import (
	"context"

	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	protoconfmutation "github.com/protoconf/protoconf/server/api/proto/v1"
	"google.golang.org/protobuf/proto"
)

type legacyProtoconfMutationServer struct {
	protoconfmutation.UnimplementedProtoconfMutationServiceServer
	srv *ProtoconfMutationServer
}

func (s *legacyProtoconfMutationServer) MutateConfig(ctx context.Context, in *protoconfmutation.ConfigMutationRequest) (*protoconfmutation.ConfigMutationResponse, error) {
	// in and next are distinct generated types (v1.ConfigMutationRequest vs
	// protoconf.v1.ConfigMutationRequest) with wire-compatible field numbers
	// but different descriptors, so proto.Merge panics ("descriptor
	// mismatch") if used here — it requires src and dst to share a
	// descriptor. Marshal/Unmarshal instead, the standard idiom for copying
	// between two wire-compatible-but-distinct proto message types.
	inBytes, err := proto.Marshal(in)
	if err != nil {
		return nil, err
	}
	next := &protoconf_pb.ConfigMutationRequest{}
	if err := proto.Unmarshal(inBytes, next); err != nil {
		return nil, err
	}
	result, err := s.srv.MutateConfig(ctx, next)
	if err != nil {
		return nil, err
	}
	resultBytes, err := proto.Marshal(result)
	if err != nil {
		return nil, err
	}
	out := &protoconfmutation.ConfigMutationResponse{}
	if err := proto.Unmarshal(resultBytes, out); err != nil {
		return nil, err
	}
	return out, nil
}
