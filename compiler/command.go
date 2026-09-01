package compiler

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"

	"github.com/mitchellh/cli"
	configtool "github.com/protoconf/libprotoconf"
	"github.com/protoconf/protoconf/command"
	protoconf_compiler_config "github.com/protoconf/protoconf/compiler/config/v1"
	compilerlib "github.com/protoconf/protoconf/compiler/lib"
	"github.com/protoconf/protoconf/consts"
	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	"go.starlark.net/repl"
	"go.starlark.net/starlark"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

type cliCommand struct {
	config *protoconf_compiler_config.CompilerConfig
	flag   *flag.FlagSet
}

func (c *cliCommand) Run(args []string) int {
	err := c.flag.Parse(args)
	if err != nil {
		fmt.Fprint(os.Stderr, "failed to parse flags: ", err)
		return 2
	}

	if c.flag.NArg() < 1 {
		c.flag.Usage()
		return 1
	}
	if c.config.Cpuprofile != "" {
		f, err := os.Create(c.config.Cpuprofile)
		if err != nil {
			slog.Error("Could not create CPU profile:", "error", err)
			os.Exit(1)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			slog.Error("Could not start CPU profile:", "error", err)
			os.Exit(1)
		}
		defer pprof.StopCPUProfile()
	}

	protoconfRoot := strings.TrimSpace(c.flag.Args()[0])
	var configs []string

	if c.flag.NArg() == 1 {
		var err error
		configs, err = GetAllConfigs(protoconfRoot)
		if err != nil {
			slog.Error("Error getting all configs", "config", protoconfRoot, "error", err)
			return 1
		}
	} else {
		configs = c.flag.Args()[1:]
	}

	if c.config.CompilerAddress != "" {
		return runRemote(c.config, configs)
	}
	return runLocally(protoconfRoot, c.config, configs)
}

func runRemote(config *protoconf_compiler_config.CompilerConfig, configs []string) int {
	conn, err := grpc.NewClient(config.CompilerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("error connecting to server", "error", err)
	}
	client := protoconf_pb.NewProtoconfCompileClient(conn)
	stream, err := client.CompileFiles(context.Background(), &protoconf_pb.CompileRequest{Files: configs})
	if err != nil {
		slog.Error("error compiling files", "error", err)
		return 1
	}
	ret := 0
	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return ret
		}
		if err != nil {
			slog.Error("error receiving response", "error", err)
			return 1
		}
		if resp != nil {
			if len(resp.Errors) > 0 {
				slog.Error("Error compiling config", "path", resp.Path, "errors", resp.Errors)
				ret = 1
				continue
			}
			if config.VerboseLogging {
				slog.Error("Compiled", "path", resp.Path, "result", resp.Result)
			}
		}
	}

}

func runLocally(protoconfRoot string, config *protoconf_compiler_config.CompilerConfig, configs []string) int {
	compiler, err := compilerlib.NewCompiler(protoconfRoot, config.VerboseLogging)
	if err != nil {
		slog.Error("Failed to initialize compiler", "error", err)
		return 1
	}
	ui := &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	if config.Repl {
		REPL(compiler)
		return 0
	}

	if config.ProcessTemplates {
		if err := findTemplateFilesAndProccess(); err != nil {
			log.Fatal(err)
		}
	}
	g, _ := errgroup.WithContext(context.Background())

	for _, config := range configs {
		filename := strings.TrimSpace(config)
		g.Go(func() error {
			err := compiler.CompileFile(filename)
			if err != nil {
				ui.Error(fmt.Sprintf("Error compiling config %s:\n    %s", filename, err))
			}
			return err
		})
	}
	if err = g.Wait(); err != nil {
		// log.Println(err)
		return 1
	}

	if config.Memprofile != "" {
		f, err := os.Create(config.Memprofile)
		if err != nil {
			log.Fatal("Could not create memory profile:", err)
		}
		defer f.Close()
		runtime.GC()
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("Could not start memory profile:", err)
		}
	}
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
	return "Compile Starlark .pconf/.mpconf files into materialized protobuf configs"
}

// Command is a cli.CommandFactory
func Command() (cli.Command, error) {
	c := &cliCommand{
		config: &protoconf_compiler_config.CompilerConfig{},
	}
	lpc := configtool.NewConfig(c.config)
	lpc.SetEnvKeyPrefix("PROTOCONF_COMPILER")
	// base is the pristine factory-default snapshot handed to command.NewConfigLayerer below.
	// It is no longer mutated as an accumulator (that role now belongs to layerer.fileLayer).
	base := proto.Clone(c.config)
	lpc.Environment()
	c.flag = flag.NewFlagSet(string(c.config.ProtoReflect().Descriptor().FullName()), flag.ContinueOnError)
	lpc.PopulateFlagSet(c.flag)
	// layerer owns the accumulated config-file layer and the env/flag provenance set for
	// this Command() instance. Constructed after PopulateFlagSet (so c.flag is fully
	// populated) and before flag.Parse runs (so markExplicitFlags can observe flags parsed
	// before each -config-file occurrence via c.flag.Visit).
	layerer := command.NewConfigLayerer(base, c.flag)
	c.flag.Func("config-file", "Compiler configuration file (available formats: json, yaml, pb). Values are overridden by PROTOCONF_COMPILER_* environment variables and by command-line flags.", func(filename string) error {
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
		// into this same message; see command.ConfigLayerer for the file/env/default
		// layering, which tracks provenance explicitly rather than inferring it from value
		// comparison (08-REVIEW.md CR-01). The ordering now holds across repeated
		// -config-file flags and for message-typed fields, not only for a single file and
		// scalar fields.
		layerer.LayerConfigFile(c.config, preFile)
		return nil
	})
	return c, nil
}

func GetAllConfigs(protoconfRoot string) ([]string, error) {
	srcDir, err := filepath.Abs(filepath.Join(protoconfRoot, consts.SrcPath))
	if err != nil {
		return nil, err
	}

	var configs []string
	err = filepath.Walk(srcDir, func(path string, f os.FileInfo, err error) error {
		ext := filepath.Ext(path)
		if ext == consts.ConfigExtension || ext == consts.MultiConfigExtension {
			configs = append(configs, strings.TrimPrefix(path, srcDir))
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return configs, nil
}

func REPL(c *compilerlib.Compiler) {
	fmt.Printf("Protoconf %s\n", consts.Version)

	loader := c.GetLoader()
	thread := &starlark.Thread{
		Load: loader.Load,
	}

	repl.REPL(thread, loader.Modules)
}
