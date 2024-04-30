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
	next := &protoconf_pb.ConfigMutationRequest{}
	proto.Merge(next, in)
	result, err := s.srv.MutateConfig(ctx, next)
	if err != nil {
		return nil, err
	}
	out := &protoconfmutation.ConfigMutationResponse{}
	proto.Merge(out, result)
	return out, nil
}
