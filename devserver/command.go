package devserver

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/mitchellh/cli"
	"github.com/protoconf/protoconf/agent"
	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
	"github.com/protoconf/protoconf/agent/filekv"
	"github.com/protoconf/protoconf/compiler"
	"github.com/protoconf/protoconf/consts"
	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	"github.com/protoconf/protoconf/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

type DevServerCommand struct{}

var _ cli.Command = &DevServerCommand{}

func (d *DevServerCommand) Help() string {
	return ""
}

func (d *DevServerCommand) Run(args []string) int {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	protoconfRoot := "."
	if len(args) > 0 {
		protoconfRoot = args[0]
	}

	rpcServer := grpc.NewServer()

	slog.Default().Info("starting agent...")
	config := &protoconf_agent_config.AgentConfig{
		DevRoot: protoconfRoot,
		Prefix:  consts.CompiledConfigPath,
		Store:   protoconf_agent_config.AgentConfig_file,
	}
	store, err := filekv.New(ctx, []string{}, &filekv.Config{ProtoconfRoot: config.DevRoot})
	if err != nil {
		slog.Error("Error creating filekv", "error", err)

	}
	agentSvc, err := agent.NewProtoconfKVAgent(store, config)
	if err != nil {
		slog.Error("error creating agent", "error", err)
	}
	protoconf_pb.RegisterProtoconfServiceServer(rpcServer, agentSvc)

	slog.Default().Info("starting compiler service...")
	compilerSvc := compiler.NewCompilerService(protoconfRoot, false)
	protoconf_pb.RegisterProtoconfCompileServer(rpcServer, compilerSvc)

	healthcheck := health.NewServer()
	healthgrpc.RegisterHealthServer(rpcServer, healthcheck)

	healthcheck.SetServingStatus("agent", healthgrpc.HealthCheckResponse_SERVING)
	healthcheck.SetServingStatus("mutation", healthgrpc.HealthCheckResponse_SERVING)
	healthcheck.SetServingStatus("compiler", healthgrpc.HealthCheckResponse_SERVING)
	healthcheck.SetServingStatus("reporting", healthgrpc.HealthCheckResponse_SERVING)

	slog.Default().Info("starting mutation server...")
	mutationServer := server.NewProtoconfMutationServer(protoconfRoot, server.WithCompiler(compilerSvc.Compiler))
	// mutationServer.PostMutationScript = ``
	mutationServer.Init(rpcServer)

	httpSrv := &http.Server{
		Addr: ":4300",
	}
	mutationServer.GenReflectionUI(ctx, rpcServer, httpSrv)

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				mutationServer.GenReflectionUI(ctx, rpcServer, httpSrv)
			}
		}
	}()

	context.AfterFunc(ctx, func() {
		slog.Info("stopping http server")
		httpSrv.Shutdown(ctx)
	})
	slog.Info("start listening", "address", httpSrv.Addr)
	httpSrv.ListenAndServe()

	return 0
}

func (d *DevServerCommand) Synopsis() string {
	return "runs a dev server"
}

func Command() (cli.Command, error) {
	return &DevServerCommand{}, nil
}
