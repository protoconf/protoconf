# Coding Conventions

**Analysis Date:** 2026-03-23

## Naming Patterns

**Files:**
- Use `snake_case.go` for all Go source files: `kv_agent_impl.go`, `module_service_test.go`
- Test files use `_test.go` suffix co-located with source: `compiler_test.go` next to `compiler.go`
- Proto-generated files use `.pb.go` and `_grpc.pb.go` suffixes: `protoconf.pb.go`, `protoconf_grpc.pb.go`
- CLI entry points use `command.go` in each package: `compiler/command.go`, `server/server.go`, `agent/agent.go`

**Packages:**
- Use short, single-word lowercase names: `agent`, `compiler`, `inserter`, `server`, `consts`, `utils`
- Nested packages for versioned protos follow `v1` convention: `agent/config/v1`, `pb/protoconf/v1`
- Sub-packages under a feature use descriptive names: `compiler/lib`, `compiler/lib/parser`, `compiler/starproto`

**Functions:**
- Use `PascalCase` for exported functions: `NewCompiler()`, `RunAgent()`, `CompileFile()`
- Constructors follow `New<Type>` pattern: `NewModuleService()`, `NewProtoconfKVAgent()`, `NewProtoconfMutationServer()`
- CLI command factories follow `Command() (cli.Command, error)` signature consistently across all packages

**Variables:**
- Use `camelCase` for local variables: `protoconfRoot`, `kvStore`, `mainOutput`
- Package-level sentinel errors use `Err` prefix: `ErrBadConfigExtension`, `ErrStarlarkEval`, `ErrPreMutationScriptError`
- Constants use `PascalCase`: `AgentDefaultAddress`, `CompiledConfigExtension`, `SrcPath`

**Types:**
- Struct types use `PascalCase`: `Compiler`, `ProtoconfKVAgent`, `ProtoconfMutationServer`
- Internal CLI config structs use unexported `cliConfig` and `cliCommand` patterns
- Option functions follow `With<Name>` pattern: `WithCompiler()` in `server/server.go`

## Code Style

**Formatting:**
- `gofmt` enforced via Trunk (`.trunk/trunk.yaml`)
- `golangci-lint` enabled in Trunk for additional linting

**Linting:**
- Trunk CI with `golangci-lint@1.55.2` and `gofmt@1.20.4`
- `buf-lint@1.28.1` for protobuf linting (config: `buf.yaml`)
- No `.golangci.yml` config file detected -- uses default golangci-lint rules

## Import Organization

**Order:**
1. Standard library imports
2. Third-party imports (sorted alphabetically)
3. Internal project imports (`github.com/protoconf/protoconf/...`)

**Import Aliases:**
- Proto-generated packages use descriptive aliases to avoid collisions:
  - `protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"` -- the primary proto package
  - `protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"` -- agent config protos
  - `protoconfservice "github.com/protoconf/protoconf/agent/api/proto/v1"` -- legacy service protos
  - `protoconfmutation "github.com/protoconf/protoconf/server/api/proto/v1"` -- mutation service protos
  - `compilerlib "github.com/protoconf/protoconf/compiler/lib"` -- when `compiler` package name collides
- Side-effect imports for proto registration use blank identifier:
  ```go
  _ "github.com/bufbuild/protovalidate-go"
  _ "github.com/bufbuild/protovalidate-go/legacy"
  _ "github.com/protoconf/protoconf/pb/protoconf/v1"
  ```

**Path Aliases:**
- No Go module path aliases or `replace` directives for internal packages
- All imports use full module path: `github.com/protoconf/protoconf/...`

## Error Handling

**Patterns:**
- Use sentinel errors defined as package-level `var` with `errors.New()`:
  ```go
  var (
      ErrBadConfigExtension = errors.New("bad config extension")
      ErrStarlarkEval       = errors.New("error evaluating starlark file")
  )
  ```
  See `compiler/lib/compiler.go`, `server/server.go`

- Wrap errors using `errors.Join()` (Go 1.20+ pattern) throughout:
  ```go
  return errors.Join(ErrLoadStarlark, err)
  return errors.Join(ErrPreMutationScriptError, err)
  return errors.Join(errors.New("error setting config store"), err)
  ```

- Check errors with `errors.Is()`:
  ```go
  if !errors.Is(err, tt.wantErr) { ... }
  ```

- Use `fmt.Errorf()` for context-rich non-sentinel errors:
  ```go
  return fmt.Errorf("error marshaling ProtoconfValue to JSON, value=%v, err: %v", protoconfValue, err)
  ```

- CLI commands return integer exit codes (0 = success, 1 = error, 2 = usage error)

## Logging

**Framework:** `log/slog` (Go standard library structured logging)

**Patterns:**
- Use `slog.Default()` or create a named logger with `slog.New()`:
  ```go
  logger := slog.Default()
  logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))
  ```
- Use structured logging with typed fields:
  ```go
  slog.Info("starting protoconf agent", slog.String("address", config.GrpcAddress))
  slog.Error("error compiling file", "file", file, "error", err)
  logger.Info("Got watch request", slog.String("key", request.Path))
  ```
- Package-level logger variable in `server/server.go`: `var logger = slog.Default()`
- Agent supports JSON logging via config: `slog.NewJSONHandler(os.Stderr, loggerHandlerOptions)`
- OpenTelemetry-aware logging via `slog-otel` wrapper in `agent/agent.go`
- Legacy `log.Printf` still appears in some older code paths (e.g., `compiler/command.go` line 176)

## CLI Command Pattern

**All CLI commands follow the `mitchellh/cli` pattern:**
```go
type cliCommand struct{}

func (c *cliCommand) Run(args []string) int { ... }
func (c *cliCommand) Help() string { ... }
func (c *cliCommand) Synopsis() string { ... }

// Command is a cli.CommandFactory
func Command() (cli.Command, error) {
    return &cliCommand{}, nil
}
```
Files: `compiler/command.go`, `server/server.go`, `inserter/inserter.go`, `agent/command.go`

**Flag handling uses `newFlagSet()` helper:**
```go
func newFlagSet() (*flag.FlagSet, *cliConfig) {
    flags := flag.NewFlagSet("", flag.ExitOnError)
    config := &cliConfig{}
    flags.StringVar(&config.grpcAddress, "grpc-address", consts.ServerDefaultAddress, "Server gRPC address")
    return flags, config
}
```

## gRPC Server Pattern

**Server setup follows a consistent pattern across agent and server packages:**
1. Create store/backend connection
2. Create server implementation struct
3. Create `grpc.NewServer()` with interceptors/stats handlers
4. Register services on the gRPC server
5. Start listening and serve

**OpenTelemetry is integrated in both agent and server:**
```go
rpcServer := grpc.NewServer(
    grpc.StatsHandler(otelgrpc.NewServerHandler()),
    grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
    grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
)
```

## Functional Options Pattern

**Used for server construction:**
```go
type MutationServerOption func(*ProtoconfMutationServer)

func WithCompiler(c *lib.Compiler) func(*ProtoconfMutationServer) {
    return func(s *ProtoconfMutationServer) { s.compiler = c }
}

func NewProtoconfMutationServer(root string, opts ...MutationServerOption) *ProtoconfMutationServer { ... }
```
See `server/server.go`

## Proto Code Generation

**Generated code pattern:**
- Proto definitions live alongside generated code: `pb/protoconf/v1/protoconf.proto` generates `pb/protoconf/v1/protoconf.pb.go`
- Generation triggered via `go:generate` directive in `generate.go`:
  ```go
  //go:generate sh -xc "find . -name '*.proto' -not -path '*pb/*' -not -path '*utils/*' | xargs protoc -I=. --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative"
  ```
- Generated `.pb.go` files excluded from Codecov coverage (`codecov.yml`)

## Constants

**All project constants centralized in `consts/consts.go`:**
- File extensions: `ConfigExtension = ".pconf"`, `MultiConfigExtension = ".mpconf"`, `CompiledConfigExtension = ".materialized_JSON"`
- Directory paths: `SrcPath = "src/"`, `CompiledConfigPath = "materialized_config/"`, `MutableConfigPath = "mutable_config/"`
- Network defaults: `AgentDefaultAddress = ":4300"`, `ServerDefaultAddress = ":4301"`

## Comments

**When to Comment:**
- Exported `Command()` functions get a brief `// Command is a cli.CommandFactory` comment
- TODO comments mark incomplete test cases: `// TODO: Add test cases.`
- TODO with author attribution: `// TODO(smintz): assert the response`
- Inline comments explain non-obvious behavior

**JSDoc/TSDoc:** Not applicable (Go codebase)

## Module Design

**Exports:**
- Each package exports a single primary constructor and type
- CLI packages export a `Command()` factory function
- Internal types (`cliCommand`, `cliConfig`) remain unexported

**Barrel Files:** Not used (Go convention -- import specific packages directly)

---

*Convention analysis: 2026-03-23*
