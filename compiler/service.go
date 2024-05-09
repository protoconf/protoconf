package compiler

import (
	"context"
	"errors"
	"log/slog"

	"github.com/protoconf/protoconf/compiler/lib"
	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	"golang.org/x/sync/errgroup"
)

type CompilerService struct {
	protoconf_pb.UnimplementedProtoconfCompileServer
	Compiler *lib.Compiler
}

var _ protoconf_pb.ProtoconfCompileServer = &CompilerService{}

func NewCompilerService(dir string, verbose bool) *CompilerService {
	return &CompilerService{
		Compiler: lib.NewCompiler(dir, verbose),
	}
}

func (s *CompilerService) CompileFiles(in *protoconf_pb.CompileRequest, srv protoconf_pb.ProtoconfCompile_CompileFilesServer) error {
	ctx := srv.Context()
	grp, ctx := errgroup.WithContext(ctx)
	for _, file := range in.Files {
		file := file
		grp.Go(func() error {
			ctx, cancel := context.WithCancelCause(ctx)
			ch, errCh := s.Compiler.CompileFileAsync(ctx, cancel, file)
			var err error
			for {
				select {
				case resp := <-ch:
					slog.Default().Debug("Compiled", slog.String("file", resp.Path))
				case e := <-errCh:
					err = errors.Join(err, e)
				case <-ctx.Done():
					e := context.Cause(ctx)
					if errors.Is(e, context.Canceled) {
						e = nil
					}
					return errors.Join(err, e)
				}
			}
		})
	}
	return grp.Wait()
}
