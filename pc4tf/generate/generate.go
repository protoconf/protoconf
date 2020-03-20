package generate

import (
	"bufio"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"

	tfplugin "github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/builder"
	"github.com/jhump/protoreflect/desc/protoprint"

	"github.com/protoconf/protoconf/pc4tf/meta"
	"github.com/protoconf/protoconf/pc4tf/providerimporter"
)

// Generator is used to create `terrafrom.proto` file with all of its
// available resources from the provider used in the current directory
// context.
type Generator struct {
	Providers  map[string]*providerimporter.ProviderImporter
	ImportPath string
	OutputPath string
}

// NewGenerator creates a new Generator
func NewGenerator(importPath, outputPath string) *Generator {
	providers := make(map[string]*providerimporter.ProviderImporter)
	return &Generator{
		ImportPath: importPath,
		OutputPath: outputPath,
		Providers:  providers,
	}
}

// PopulateProviders finds all providers available for the runtime context
// and populate the Generator with proto schemas from `providerimporter`
func (g *Generator) PopulateProviders() error {
	dirs := []string{g.ImportPath}
	log.Println(dirs)
	meta := discovery.FindPlugins("provider", dirs)
	log.Println(meta)

	for c := range meta {
		config := tfplugin.ClientConfig(c)
		client, err := providerimporter.NewGRPCClient(config)
		if err != nil {
			return err
		}
		p, err := providerimporter.NewProviderImporter(c.Name, client)
		if err != nil {
			return err
		}
		g.Providers[c.Name] = p
	}

	return nil
}

// Save will write all protofiles to disk
func (g *Generator) Save() error {
	file := builder.NewFile("terraform/terraform.proto")
	file.SetProto3(true)
	file.SetPackageName("terraform")

	main := builder.NewMessage("Terraform")
	resources := builder.NewMessage("Resources")
	data := builder.NewMessage("Datasources")

	main.AddNestedMessage(resources)
	main.AddField(builder.NewField("resource", builder.FieldTypeMessage(resources)).SetJsonName("resource"))
	main.AddNestedMessage(data)
	main.AddField(builder.NewField("data", builder.FieldTypeMessage(data)).SetJsonName("data"))

	metaFile := meta.MetaFile()
	protoFiles := []*builder.FileBuilder{file, metaFile}

	keys := []string{}
	for name := range g.Providers {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	for _, name := range keys {
		p := g.Providers[name]
		log.Println("saving", name)
		protoFiles = append(protoFiles, p.Resources)
		protoFiles = append(protoFiles, p.Datasources)
		for _, b := range p.Provider.GetMessage("resources").GetChildren() {
			f := p.Provider.GetMessage("resources").GetField(b.GetName())
			resources.AddField(f)
		}
		for _, b := range p.Provider.GetMessage("data").GetChildren() {
			f := p.Provider.GetMessage("data").GetField(b.GetName())
			data.AddField(f)
		}
	}
	file.AddMessage(main)
	descriptors := []*desc.FileDescriptor{}
	for _, f := range protoFiles {
		d, err := f.Build()
		if err != nil {
			return err
		}
		descriptors = append(descriptors, d)
	}

	pr := &protoprint.Printer{}
	return pr.PrintProtoFiles(descriptors, g.opener)
}

func (g *Generator) opener(name string) (io.WriteCloser, error) {
	filePath := filepath.Join(g.OutputPath, name)
	dirName := filepath.Dir(filePath)
	err := os.MkdirAll(dirName, 0755)
	if err != nil {
		return nil, err
	}
	f, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	return &myWriteCloser{f, bufio.NewWriter(f)}, nil
}

type myWriteCloser struct {
	f *os.File
	*bufio.Writer
}

// https://stackoverflow.com/questions/43115699/how-to-get-a-bufio-writer-that-implements-io-writecloser
func (mwc *myWriteCloser) Close() error {
	if err := mwc.Flush(); err != nil {
		return err
	}
	return mwc.f.Close()
}
