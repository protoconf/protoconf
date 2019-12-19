package generate

import (
	"bufio"
	"io"
	"log"
	"os"
	"path/filepath"

	tfplugin "github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/builder"
	"github.com/jhump/protoreflect/desc/protoprint"
	"github.com/protoconf/protoconf/pc4tf/provider_importer"
)

type Generator struct {
	Providers  map[string]*provider_importer.ProviderImporter
	ImportPath string
	OutputPath string
}

func NewGenerator(importPath, outputPath string) *Generator {
	providers := make(map[string]*provider_importer.ProviderImporter)
	return &Generator{
		ImportPath: importPath,
		OutputPath: outputPath,
		Providers:  providers,
	}
}

func (g *Generator) PopulateProviders() error {
	dirs := []string{g.ImportPath}
	log.Println(dirs)
	meta := discovery.FindPlugins("provider", dirs)
	log.Println(meta)

	for c, _ := range meta {
		config := tfplugin.ClientConfig(c)
		client, err := provider_importer.NewGRPCClient(config)
		if err != nil {
			return err
		}
		p, err := provider_importer.NewProviderImporter(c.Name, client)
		if err != nil {
			return err
		}
		g.Providers[c.Name] = p
	}

	return nil
}

func (g *Generator) Save() error {
	file := builder.NewFile("terraform/terraform.proto")
	file.SetProto3(true)
	file.SetPackageName("terraform")

	main := builder.NewMessage("Terraform")
	resources := builder.NewMessage("Resources")

	//
	main.AddNestedMessage(resources)
	main.AddField(builder.NewField("resource", builder.FieldTypeMessage(resources)).SetJsonName("resource"))

	protoFiles := []*builder.FileBuilder{file}

	for name, p := range g.Providers {
		log.Println("saving", name)
		protoFiles = append(protoFiles, p.Resources)
		protoFiles = append(protoFiles, p.Datasources)
		for _, b := range p.Provider.GetMessage("resources").GetChildren() {
			f := p.Provider.GetMessage("resources").GetField(b.GetName())
			resources.AddField(f)
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
	return &MyWriteCloser{f, bufio.NewWriter(f)}, nil
}

type MyWriteCloser struct {
	f *os.File
	*bufio.Writer
}

// https://stackoverflow.com/questions/43115699/how-to-get-a-bufio-writer-that-implements-io-writecloser
func (mwc *MyWriteCloser) Close() error {
	if err := mwc.Flush(); err != nil {
		return err
	}
	return mwc.f.Close()
}
