package server

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fullstorydev/grpcui/standalone"
	"github.com/google/uuid"
	"github.com/mitchellh/cli"
	"github.com/protoconf/protoconf/compiler/lib"
	"github.com/protoconf/protoconf/compiler/lib/parser"
	"github.com/protoconf/protoconf/consts"
	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	protoconfmutation "github.com/protoconf/protoconf/server/api/proto/v1"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
)

var logger = slog.Default()

type cliCommand struct{}

type cliConfig struct {
	grpcAddress        string
	preMutationScript  string
	postMutationScript string
}

func newFlagSet() (*flag.FlagSet, *cliConfig) {
	flags := flag.NewFlagSet("", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: [OPTION]... protoconfRoot")
		flags.PrintDefaults()
	}

	config := &cliConfig{}
	flags.StringVar(&config.grpcAddress, "grpc-address", consts.ServerDefaultAddress, "Server gRPC address")
	flags.StringVar(&config.preMutationScript, "pre", "", "Pre mutation script")
	flags.StringVar(&config.postMutationScript, "post", "", "Post mutation script")

	return flags, config
}

type exampleFunc func(path string, msg proto.Message) standalone.Example

func (c *cliCommand) Run(args []string) int {
	flags, config := newFlagSet()
	flags.Parse(args)

	if flags.NArg() < 1 {
		flags.Usage()
		return 1
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var err error

	expTracer, err := otlptracegrpc.New(ctx)
	if err != nil {
		panic(err)
	}

	resources, _ := resource.New(ctx,
		resource.WithFromEnv(),   // pull attributes from OTEL_RESOURCE_ATTRIBUTES and OTEL_SERVICE_NAME environment variables
		resource.WithProcess(),   // This option configures a set of Detectors that discover process information
		resource.WithOS(),        // This option configures a set of Detectors that discover OS information
		resource.WithContainer(), // This option configures a set of Detectors that discover container information
		resource.WithHost(),      // This option configures a set of Detectors that discover host information
		resource.WithAttributes(
			semconv.ServiceNameKey.String("protoconf"),
			semconv.ServiceVersionKey.String(consts.Version),
		),
	)
	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(expTracer),
		trace.WithResource(resources),
	)
	context.AfterFunc(ctx, func() {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			slog.Default().Error("error shutting down tracer provider", slog.String("error", err.Error()))
		}
	})
	otel.SetTracerProvider(tracerProvider)

	// From here, the tracerProvider can be used by instrumentation to collect
	// telemetry.
	expMeter, err := otlpmetricgrpc.New(ctx)
	if err != nil {
		panic(err)
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(expMeter)),
		metric.WithResource(resources),
	)
	context.AfterFunc(ctx, func() {
		if err := meterProvider.Shutdown(ctx); err != nil {
			slog.Default().Error("error shutting down meter provider", slog.String("error", err.Error()))
		}
	})
	otel.SetMeterProvider(meterProvider)

	protoconfRoot := strings.TrimSpace(flags.Args()[0])
	protoconfServer := NewProtoconfMutationServer(protoconfRoot)
	protoconfServer.config = config

	logger.Info("starting protoconf server", "address", config.grpcAddress, "version", consts.Version, "root", protoconfRoot, "pre", config.preMutationScript, "post", config.postMutationScript)

	listener, err := net.Listen("tcp", config.grpcAddress)
	if err != nil {
		logger.Error("error listening", err)
		return 1
	}

	rpcServer := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	protoconfmutation.RegisterProtoconfMutationServiceServer(rpcServer, &legacyProtoconfMutationServer{srv: protoconfServer})
	protoconf_pb.RegisterProtoconfMutationServiceServer(rpcServer, protoconfServer)
	protoconf_pb.RegisterProtoconfMutationReportServiceServer(rpcServer, protoconfServer)

	exampleMaker := map[string]exampleFunc{}
	protoconfServer.parser.FilesResolver.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		_, err := protoregistry.GlobalFiles.FindFileByPath(fd.Path())
		if !errors.Is(err, protoregistry.NotFound) {
			return true
		}

		if fd.Services().Len() < 1 {
			return true
		}
		for i := 0; i < fd.Services().Len(); i++ {
			svc := fd.Services().Get(i)
			svcDesc := &grpc.ServiceDesc{
				ServiceName: string(svc.FullName()),
				HandlerType: (*protoconf_pb.ProtoconfMutationServiceServer)(nil),
			}
			for j := 0; j < svc.Methods().Len(); j++ {
				method := svc.Methods().Get(j)
				svcDesc.Methods = append(svcDesc.Methods, grpc.MethodDesc{
					MethodName: string(method.Name()),
					Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
						logger.Debug("Method called", "method", method.Name(), "srv", srv)
						in := dynamicpb.NewMessage(method.Input())
						if err := dec(in); err != nil {
							return nil, err
						}
						if interceptor == nil {
							return srv.(*ProtoconfMutationServer).Put(ctx, in)
						}
						info := &grpc.UnaryServerInfo{
							Server:     srv,
							FullMethod: string(method.FullName()),
						}
						handler := func(ctx context.Context, req interface{}) (interface{}, error) {
							return srv.(*ProtoconfMutationServer).Put(ctx, in)
						}
						return interceptor(ctx, in, info, handler)
					},
				})
				exampleMaker[string(method.Input().FullName())] = func(path string, msg proto.Message) standalone.Example {
					return standalone.Example{
						Name:    path,
						Service: string(svc.FullName()),
						Method:  string(method.Name()),
						Request: standalone.ExampleRequest{
							Metadata: []standalone.ExampleMetadataPair{{Name: "path", Value: path}},
							Data:     msg,
						},
					}
				}
			}
			logger.Info("Registering service", "service", svcDesc.ServiceName)
			rpcServer.RegisterService(svcDesc, protoconfServer)
		}
		return true
	})
	logger.Debug("examples", "maker", exampleMaker)
	reflectionServer := reflection.NewServer(reflection.ServerOptions{
		Services:           rpcServer,
		DescriptorResolver: protoconfServer.parser.FilesResolver,
		ExtensionResolver:  protoconfServer.parser.LocalResolver,
	})

	grpc_reflection_v1alpha.RegisterServerReflectionServer(rpcServer, reflectionServer)

	logger.Info("protoconf server running")

	httpServer := &http.Server{
		Addr: ":4333",
	}
	context.AfterFunc(ctx, func() {
		rpcServer.GracefulStop()
		httpServer.Shutdown(ctx)
	})
	go func() {
		err = rpcServer.Serve(listener)
		if err != nil {
			logger.Error("Error serving gRPC, err=%s", err)
		}
	}()
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				protoconfServer.genReflectionUI(ctx, listener, httpServer, exampleMaker)
			}
		}
	}()
	protoconfServer.genReflectionUI(ctx, listener, httpServer, exampleMaker)

	httpServer.ListenAndServe()

	return 0
}

func (c *cliCommand) Help() string {
	var b bytes.Buffer
	b.WriteString(c.Synopsis())
	b.WriteString("\n")
	flags, _ := newFlagSet()
	flags.SetOutput(&b)
	flags.Usage()
	return b.String()
}

func (c *cliCommand) Synopsis() string {
	return "Runs a server"
}

// Command is a cli.CommandFactory
func Command() (cli.Command, error) {
	return &cliCommand{}, nil
}

type ProtoconfMutationServer struct {
	protoconf_pb.UnimplementedProtoconfMutationServiceServer
	protoconf_pb.UnimplementedProtoconfMutationReportServiceServer
	config        *cliConfig
	protoconfRoot string
	parser        *parser.Parser
	reports       *sync.Map
}

func NewProtoconfMutationServer(protoconfRoot string) *ProtoconfMutationServer {
	ms := lib.NewModuleService(protoconfRoot)
	ms.LoadFromLockFile()
	parser := parser.NewParser(ms.GetProtoFilesRegistry())
	return &ProtoconfMutationServer{protoconfRoot: protoconfRoot, config: &cliConfig{}, parser: parser, reports: &sync.Map{}}
}

func (s *ProtoconfMutationServer) Put(ctx context.Context, in *dynamicpb.Message) (proto.Message, error) {
	any, err := anypb.New(in)
	if err != nil {
		return nil, err
	}
	md, _ := metadata.FromIncomingContext(ctx)
	path := md.Get("path")
	if len(path) != 1 {
		return nil, fmt.Errorf("path metadata not found")
	}
	req := &protoconf_pb.ConfigMutationRequest{
		Path: path[0],
		Value: &protoconf_pb.ProtoconfValue{
			Value: any,
		},
	}
	slog.Debug("Put", "in", in, "path", path, "metadata", md)
	return s.MutateConfig(ctx, req)
}

func (s *ProtoconfMutationServer) MutateConfig(ctx context.Context, in *protoconf_pb.ConfigMutationRequest) (*protoconf_pb.ConfigMutationResponse, error) {
	id := uuid.New()
	s.reports.Store(id, &protoconf_pb.ConfigMutationResponse{Uuid: id.String()})
	defer s.reports.Delete(id)
	log.Printf("Mutating path=%s", in.Path)
	filename := filepath.Join(s.protoconfRoot, consts.MutableConfigPath, filepath.Clean(in.Path)+consts.CompiledConfigExtension)

	resolver := s.parser.LocalResolver
	jsonData, err := protojson.MarshalOptions{Resolver: resolver, Multiline: true}.Marshal(in.Value)
	if err != nil {
		return nil, logError(fmt.Errorf("error marshaling ProtoconfValue to JSON, value=%s", in.Value))
	}

	if s.config.preMutationScript != "" {
		if err := runScript(s.config.preMutationScript, id.String()); err != nil {
			return nil, logError(fmt.Errorf("error running pre mutation script, err=%s", err))
		}
	}

	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return nil, logError(fmt.Errorf("error creating output directory %s, err: %s", filepath.Dir(filename), err))
	}

	if err := os.WriteFile(filename, []byte(jsonData), 0644); err != nil {
		return nil, logError(fmt.Errorf("error writing to file %s, err: %s", filename, err))
	}

	log.Printf("Written to %s", filename)

	if s.config.postMutationScript != "" {
		if err := runScript(s.config.postMutationScript, id.String()); err != nil {
			return nil, logError(fmt.Errorf("error running post mutation script, err=%s", err))
		}
	}

	if report, ok := s.reports.Load(id); ok {
		return report.(*protoconf_pb.ConfigMutationResponse), nil
	}
	return &protoconf_pb.ConfigMutationResponse{}, nil
}

func (s *ProtoconfMutationServer) ReportProgress(ctx context.Context, in *protoconf_pb.ConfigMutationResponse) (*protoconf_pb.ConfigMutationResponse, error) {
	if current, ok := s.reports.Load(in.Uuid); !ok {
		return nil, fmt.Errorf("report not found")
	} else {
		update := current.(*protoconf_pb.ConfigMutationResponse)
		proto.Merge(update, in)
		s.reports.Store(in.Uuid, update)
		return update, nil
	}
}

func logError(err error) error {
	log.Printf("Error: %s", err)
	return err
}

func runScript(filename string, uuid string) error {
	cmd := exec.Command(filename)
	cmd.Env = append(cmd.Env, "PROTOCONF_MUTATION_UUID="+uuid)
	_, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			log.Printf("Script error: %s", string(ee.Stderr))
		}
		return err
	}
	return nil
}

func (s *ProtoconfMutationServer) genReflectionUI(ctx context.Context, listener net.Listener, httpServer *http.Server, exampleMaker map[string]exampleFunc) error {
	cc, err := grpc.DialContext(ctx, listener.Addr().String(), grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Default().Error("error creating grpc client", "err", err)
		return err
	}

	examples := []standalone.Example{}
	filepath.WalkDir(filepath.Join(s.protoconfRoot, consts.MutableConfigPath), func(path string, info fs.DirEntry, _ error) error {
		if info.IsDir() {
			return nil
		}
		value := &protoconf_pb.ProtoconfValue{}
		err = s.parser.ReadConfig(path, value)
		if err != nil {
			logger.Error("error reading config", "path", path, "err", err)
			return nil
		}
		mt, err := s.parser.LocalResolver.FindMessageByURL(value.GetValue().GetTypeUrl())
		if err != nil {
			logger.Error("error finding message", "url", value.GetValue().GetTypeUrl(), "err", err)
			return err
		}
		dynamic := dynamicpb.NewMessage(mt.New().Descriptor())

		err = value.GetValue().UnmarshalTo(dynamic)
		if err != nil {
			logger.Error("error unmarshaling any", "err", err)
			return err
		}
		path, _ = filepath.Rel(filepath.Join(s.protoconfRoot, consts.MutableConfigPath), path)
		path = strings.TrimSuffix(path, consts.CompiledConfigExtension)
		logger.Debug("example", "path", path, "value", dynamic)
		if f, ok := exampleMaker[string(mt.Descriptor().FullName())]; ok {
			examples = append(examples, f(path, dynamic))
		}

		return nil
	})
	logger.Debug("examples", "examples", examples)
	ex, err := standalone.WithExamples(examples...)
	if err != nil {
		slog.Default().Error("error creating examples", "err", err)
		return err
	}
	h, err := standalone.HandlerViaReflection(ctx, cc, listener.Addr().String(), ex)
	if err != nil {
		slog.Default().Error("error creating reflection handler", "err", err)
		return err
	}
	httpServer.Handler = h
	return nil
}
