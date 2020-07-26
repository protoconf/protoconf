package terraformimporter

import (
	"log"
	"sort"

	tfplugin "github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/jhump/protoreflect/desc/builder"

	"github.com/protoconf/protoconf/importers"
	"github.com/protoconf/protoconf/importers/terraform_importer/meta"
)

// Generator is used to create `terrafrom.proto` file with all of its
// available resources from the provider used in the current directory
// context.
type Generator struct {
	Providers  map[string]*ProviderImporter
	ImportPath string
	Importer   *importers.Importer
}

// NewGenerator creates a new Generator
func NewGenerator(importPath, outputPath string) *Generator {
	providers := make(map[string]*ProviderImporter)
	return &Generator{
		ImportPath: importPath,
		Importer:   importers.NewImporter("terraform/terraform.proto", outputPath),
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
		client, err := NewGRPCClient(config)
		if err != nil {
			return err
		}
		p, err := NewProviderImporter(c.Name, client)
		if err != nil {
			return err
		}
		g.Providers[c.Name] = p
	}

	return nil
}

// Save will write all protofiles to disk
func (g *Generator) Save() error {
	file := g.Importer.MasterFile
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
	for _, f := range protoFiles {
		g.Importer.RegisterFile(f)
	}

	return g.Importer.SaveAll()

}
