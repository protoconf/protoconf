package server

import (
	"bytes"
	"context"
	"crypto/subtle"
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
	configtool "github.com/protoconf/libprotoconf"
	protoconfservice "github.com/protoconf/protoconf/agent/api/proto/v1"
	"github.com/protoconf/protoconf/command"
	"github.com/protoconf/protoconf/compiler"
	"github.com/protoconf/protoconf/compiler/lib"
	"github.com/protoconf/protoconf/compiler/lib/parser"
	"github.com/protoconf/protoconf/consts"
	"github.com/protoconf/protoconf/observability"
	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	protoconfmutation "github.com/protoconf/protoconf/server/api/proto/v1"
	protoconf_server_config "github.com/protoconf/protoconf/server/config/v1"
	"github.com/protoconf/protoconf/utils"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
)

var logger = slog.Default()

type cliCommand struct {
	config *protoconf_server_config.ServerConfig
	flag   *flag.FlagSet
}

// bearerTokenInterceptor returns a gRPC unary interceptor that validates Bearer tokens.
// When expectedToken is empty, all requests pass through (backward compatible).
func bearerTokenInterceptor(expectedToken string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if expectedToken == "" {
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
		token := strings.TrimPrefix(values[0], "Bearer ")
		if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}
		return handler(ctx, req)
	}
}

type exampleFunc func(path string, msg proto.Message) standalone.Example

func (c *cliCommand) Run(args []string) int {
	return c.run(context.Background(), args)
}

func (c *cliCommand) run(ctx context.Context, args []string) int {
	err := c.flag.Parse(args)
	if err != nil {
		fmt.Fprint(os.Stderr, "failed to parse flags: ", err)
		return 2
	}

	if c.flag.NArg() < 1 {
		c.flag.Usage()
		return 1
	}
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	shutdown, err := observability.Init(ctx, "protoconf")
	if err != nil {
		slog.Warn("OTel init failed, continuing without telemetry", "error", err)
	}
	context.AfterFunc(ctx, func() {
		if err := shutdown(ctx); err != nil {
			slog.Default().Error("error shutting down OTel providers", "error", err)
		}
	})

	protoconfRoot := strings.TrimSpace(c.flag.Args()[0])
	protoconfServer, err := NewProtoconfMutationServer(protoconfRoot)
	if err != nil {
		slog.Error("failed to create mutation server", "error", err)
		return 1
	}
	protoconfServer.config = c.config
	protoconfServer.PreMutationScript = c.config.PreMutationScript
	protoconfServer.PostMutationScript = c.config.PostMutationScript

	if err := validateScriptPath(c.config.PreMutationScript); err != nil {
		slog.Error("invalid pre-mutation script", "error", err)
		return 1
	}
	if err := validateScriptPath(c.config.PostMutationScript); err != nil {
		slog.Error("invalid post-mutation script", "error", err)
		return 1
	}

	logger.Info("starting protoconf server", "address", c.config.GrpcAddress, "version", consts.Version, "root", protoconfRoot, "pre", c.config.PreMutationScript, "post", c.config.PostMutationScript)

	serverOpts := []grpc.ServerOption{grpc.StatsHandler(otelgrpc.NewServerHandler())}

	tlsCfg, err := utils.BuildTLSConfig(utils.TLSFiles{
		CertFile: c.config.TlsCert,
		KeyFile:  c.config.TlsKey,
		CAFile:   c.config.TlsCa,
	})
	if err != nil {
		slog.Error("failed to build TLS config", "error", err)
		return 1
	}
	if tlsCfg != nil {
		serverOpts = append(serverOpts, grpc.Creds(credentials.NewTLS(tlsCfg)))
		logger.Info("gRPC server TLS enabled")
	} else {
		slog.Warn("gRPC server running without TLS -- connections are not encrypted")
	}

	serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(bearerTokenInterceptor(c.config.AuthToken)))
	if c.config.AuthToken == "" {
		slog.Warn("mutation server running without authentication -- requests are not authenticated")
	}

	rpcServer := grpc.NewServer(serverOpts...)
	protoconfServer.Init(rpcServer)

	logger.Info("protoconf server running")

	httpServer := &http.Server{
		Addr: c.config.GrpcAddress,
	}
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				protoconfServer.GenReflectionUI(ctx, rpcServer, httpServer)
			}
		}
	}()
	protoconfServer.GenReflectionUI(ctx, rpcServer, httpServer)
	context.AfterFunc(ctx, func() {
		httpServer.Shutdown(ctx)
	})

	httpServer.ListenAndServe()

	return 0
}

func (c *cliCommand) Help() string {
	var b bytes.Buffer
	b.WriteString(c.Synopsis())
	b.WriteString("\n")
	c.flag.SetOutput(&b)
	c.flag.Usage()
	return b.String()
}

func (c *cliCommand) Synopsis() string {
	return "Start the mutation server that accepts config changes via gRPC"
}

// Command is a cli.CommandFactory
func Command() (cli.Command, error) {
	c := &cliCommand{
		config: &protoconf_server_config.ServerConfig{
			GrpcAddress: consts.ServerDefaultAddress,
		},
	}
	lpc := configtool.NewConfig(c.config)
	lpc.SetEnvKeyPrefix("PROTOCONF_SERVER")
	// base is the factory-default snapshot the config-file handler needs to tell a value
	// explicitly supplied by an env var or flag apart from one that is merely a built-in
	// default (see command.LayerConfigFile).
	base := proto.Clone(c.config)
	lpc.Environment()
	c.flag = flag.NewFlagSet(string(c.config.ProtoReflect().Descriptor().FullName()), flag.ContinueOnError)
	lpc.PopulateFlagSet(c.flag)
	c.flag.Func("config-file", "Server configuration file (available formats: json, yaml, pb). Values are overridden by PROTOCONF_SERVER_* environment variables and by command-line flags.", func(filename string) error {
		b, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("failed to read config file: %v", err)
		}
		preFile := proto.Clone(c.config)
		err = lpc.Unmarshal(filename, b)
		if err != nil {
			return fmt.Errorf("failed to parse config file: %v", err)
		}
		// Precedence implemented here: flags > env vars > config file > proto defaults
		// (PCLI-09). Flags win because flag.Parse runs after lpc.Environment() and writes
		// into this same message; see command.LayerConfigFile for the file/env/default
		// layering.
		command.LayerConfigFile(c.config, base, preFile)
		return nil
	})
	return c, nil
}

type ProtoconfMutationServer struct {
	protoconf_pb.UnimplementedProtoconfMutationServiceServer
	protoconf_pb.UnimplementedProtoconfMutationReportServiceServer
	config             *protoconf_server_config.ServerConfig
	protoconfRoot      string
	parser             *parser.Parser
	reports            *sync.Map
	exampleMaker       map[string]exampleFunc
	PreMutationScript  string
	PostMutationScript string
	compiler           *lib.Compiler
}

type MutationServerOption func(*ProtoconfMutationServer)

func WithCompiler(c *lib.Compiler) func(*ProtoconfMutationServer) {
	return func(s *ProtoconfMutationServer) {
		s.compiler = c
	}
}

func NewProtoconfMutationServer(protoconfRoot string, opts ...MutationServerOption) (*ProtoconfMutationServer, error) {
	ms, err := lib.NewModuleService(protoconfRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to create module service: %w", err)
	}
	ms.LoadFromLockFile()
	parser := parser.NewParserWithDescriptorRegistry(ms.GetProtoRegistry())
	parser.FilesResolver.RegisterFile(grpc_reflection_v1.File_grpc_reflection_v1_reflection_proto)
	parser.FilesResolver.RegisterFile(grpc_reflection_v1alpha.File_grpc_reflection_v1alpha_reflection_proto)
	parser.FilesResolver.RegisterFile(grpc_health_v1.File_grpc_health_v1_health_proto)
	parser.FilesResolver.RegisterFile(protoconfmutation.File_server_api_proto_v1_protoconf_mutation_proto)
	parser.FilesResolver.RegisterFile(protoconf_pb.File_protoconf_v1_protoconf_proto)
	parser.FilesResolver.RegisterFile(protoconfservice.File_agent_api_proto_v1_protoconf_service_proto)
	s := &ProtoconfMutationServer{protoconfRoot: protoconfRoot, config: &protoconf_server_config.ServerConfig{}, parser: parser, reports: &sync.Map{}}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
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

func (s *ProtoconfMutationServer) Init(rpcServer *grpc.Server) {
	protoconfmutation.RegisterProtoconfMutationServiceServer(rpcServer, &legacyProtoconfMutationServer{srv: s})
	protoconf_pb.RegisterProtoconfMutationServiceServer(rpcServer, s)
	protoconf_pb.RegisterProtoconfMutationReportServiceServer(rpcServer, s)

	s.exampleMaker = map[string]exampleFunc{}
	s.parser.FilesResolver.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
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
				if method.Output().FullName() != protoreflect.FullName("protoconf.v1.ConfigMutationResponse") {
					continue
				}
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
				s.exampleMaker[string(method.Input().FullName())] = func(path string, msg proto.Message) standalone.Example {
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
			if len(svcDesc.Methods) < 1 {
				continue
			}
			logger.Info("Registering service", "service", svcDesc.ServiceName)
			rpcServer.RegisterService(svcDesc, s)
		}
		return true
	})
	logger.Debug("examples", "maker", s.exampleMaker)
	reflectionServer := reflection.NewServerV1(reflection.ServerOptions{
		Services:           rpcServer,
		DescriptorResolver: s.parser.FilesResolver,
		ExtensionResolver:  s.parser.LocalResolver,
	})

	grpc_reflection_v1.RegisterServerReflectionServer(rpcServer, reflectionServer)
	grpc_reflection_v1alpha.RegisterServerReflectionServer(rpcServer, reflection.NewServer(reflection.ServerOptions{
		Services:           rpcServer,
		DescriptorResolver: s.parser.FilesResolver,
		ExtensionResolver:  s.parser.LocalResolver,
	}))
}

func (s *ProtoconfMutationServer) StoreReport(id string, f func(*protoconf_pb.ConfigMutationResponse) *protoconf_pb.ConfigMutationResponse) {
	item, _ := s.reports.LoadOrStore(id, &protoconf_pb.ConfigMutationResponse{})
	item = f(item.(*protoconf_pb.ConfigMutationResponse))
	s.reports.Store(id, item)
}

var (
	ErrPreMutationScriptError  = errors.New("error running pre-mutation script")
	ErrPostMutationScriptError = errors.New("error running post-mutation script")
	ErrInternalCompilerError   = errors.New("error compiling configs")
)

func (s *ProtoconfMutationServer) MutateConfig(ctx context.Context, in *protoconf_pb.ConfigMutationRequest) (*protoconf_pb.ConfigMutationResponse, error) {
	id := uuid.NewString()
	s.reports.Store(id, &protoconf_pb.ConfigMutationResponse{Uuid: id})
	defer s.reports.Delete(id)
	slog.Info("Mutating path", "path", in.Path)
	filename := filepath.Join(s.protoconfRoot, consts.MutableConfigPath, filepath.Clean(in.Path)+consts.CompiledConfigExtension)

	resolver := s.parser.LocalResolver
	jsonData, err := protojson.MarshalOptions{Resolver: resolver, Multiline: true}.Marshal(in.Value)
	if err != nil {
		return nil, logError(fmt.Errorf("error marshaling ProtoconfValue to JSON, value=%s, err=%s", in.Value, err))
	}

	if s.PreMutationScript != "" {
		t := time.Now()
		if err := s.runScript(s.PreMutationScript, id, s.config.AuthToken, in.GetScriptMetadata()); err != nil {
			return nil, logError(errors.Join(ErrPreMutationScriptError, err))
		}
		s.StoreReport(id, func(cmr *protoconf_pb.ConfigMutationResponse) *protoconf_pb.ConfigMutationResponse {
			cmr.PreScriptDuration = durationpb.New(time.Since(t))
			return cmr
		})
	}

	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return nil, logError(fmt.Errorf("error creating output directory %s, err: %s", filepath.Dir(filename), err))
	}

	if err := os.WriteFile(filename, []byte(jsonData), 0644); err != nil {
		return nil, logError(fmt.Errorf("error writing to file %s, err: %s", filename, err))
	}

	slog.Info("Written to", "filename", filename)

	if s.compiler != nil {
		t := time.Now()
		files, err := compiler.GetAllConfigs(s.protoconfRoot)
		if err != nil {
			return nil, err
		}
		g, _ := errgroup.WithContext(ctx)
		for _, f := range files {
			file := f
			g.Go(func() error {
				lt := time.Now()
				err := s.compiler.CompileFile(file)
				if err != nil {
					slog.Error("error compiling file", "file", file, "error", err)
				} else {
					slog.Info("compiled", "file", file, "duration", time.Since(lt))
				}
				return err
			})

		}
		if err = g.Wait(); err != nil {
			return nil, errors.Join(ErrInternalCompilerError, err)
		}
		s.StoreReport(id, func(cmr *protoconf_pb.ConfigMutationResponse) *protoconf_pb.ConfigMutationResponse {
			cmr.CompileDuration = durationpb.New(time.Since(t))
			return cmr
		})
	}

	if s.PostMutationScript != "" {
		t := time.Now()
		if err := s.runScript(s.PostMutationScript, id, s.config.AuthToken, in.GetScriptMetadata()); err != nil {
			return nil, logError(errors.Join(ErrPostMutationScriptError, err))
		}
		s.StoreReport(id, func(cmr *protoconf_pb.ConfigMutationResponse) *protoconf_pb.ConfigMutationResponse {
			cmr.PostScriptDuration = durationpb.New(time.Since(t))
			return cmr
		})
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
	slog.Error("error occurred", "error", err)
	return err
}

// validateScriptPath checks that a script path is safe to use at server startup.
// It returns nil if path is empty (no script configured).
// It rejects paths containing "..", non-existent files, directories, and non-executable files.
func validateScriptPath(path string) error {
	if path == "" {
		return nil
	}
	if strings.Contains(path, "..") {
		return fmt.Errorf("script path must not contain '..': %q", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("script path does not exist: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("script path is a directory, not a file: %q", path)
	}
	if info.Mode()&0111 == 0 {
		return fmt.Errorf("script path is not executable: %q", path)
	}
	return nil
}

func (s *ProtoconfMutationServer) runScript(filename, uuid, authToken, scriptMetadata string) error {
	// Defense-in-depth: re-check file existence before exec (file may have been removed after startup validation).
	if _, err := os.Stat(filename); err != nil {
		return fmt.Errorf("script no longer exists: %w", err)
	}
	cmd := exec.Command(filename)
	cmd.Env = append(cmd.Env,
		"PROTOCONF_MUTATION_UUID="+uuid,
		"PROTOCONF_COMPILER_ADDR="+s.config.GrpcAddress, // Legacy env var name kept for script backward compatibility
		"PROTOCONF_AUTH_TOKEN="+authToken,
		"PROTOCONF_SCRIPT_METADATA="+scriptMetadata,
	)
	_, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			slog.Error("Script error", "error", string(ee.Stderr))
		}
		return err
	}
	return nil
}

func (s *ProtoconfMutationServer) GenReflectionUI(ctx context.Context, rpcServer *grpc.Server, httpServer *http.Server) error {
	examples := []standalone.Example{}
	filepath.WalkDir(filepath.Join(s.protoconfRoot, consts.MutableConfigPath), func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
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
		if f, ok := s.exampleMaker[string(mt.Descriptor().FullName())]; ok {
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

	buffer := 1024 * 1024
	lis := bufconn.Listen(buffer)
	_ = health.NewServer()
	go func() {
		context.AfterFunc(ctx, func() {
			slog.Info("shutting down rpc server")
			rpcServer.GracefulStop()
		})
		if err := rpcServer.Serve(lis); err != nil {
			log.Printf("error serving server: %v", err)
		}
	}()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("error connecting to server", "error", err)
	}
	ui, err := standalone.HandlerViaReflection(ctx, conn, httpServer.Addr, ex)
	if err != nil {
		slog.Default().Error("error creating reflection handler", "err", err)
		return err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			rpcServer.ServeHTTP(w, r)
		} else {
			ui.ServeHTTP(w, r)
		}
	})
	httpServer.Handler = h2c.NewHandler(mux, &http2.Server{})
	return nil
}
