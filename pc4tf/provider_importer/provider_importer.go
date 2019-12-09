package provider_importer

import (
	"fmt"
	"log"
	"os/user"
	"path/filepath"
	"runtime"

	plugin "github.com/hashicorp/go-plugin"
	tfplugin "github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/hashicorp/terraform/providers"
	"github.com/jhump/protoreflect/desc/builder"
)

type ProviderImporter struct {
	Resources   *builder.FileBuilder
	Datasources *builder.FileBuilder
	Provider    *builder.FileBuilder
	config      *builder.MessageBuilder
}

func NewProviderImporter(name string, client providers.Interface) (*ProviderImporter, error) {
	p := &ProviderImporter{}

	schema_response := client.GetSchema()
	defer client.Close()

	p.Resources = NewFile(name, "resources")
	p.Datasources = NewFile(name, "data")
	p.Provider = NewFile(name, "provider")

	p.Provider.AddMessage(populateResources("resources", p.Resources, schema_response.ResourceTypes))
	p.Provider.AddMessage(populateResources("data", p.Datasources, schema_response.DataSources))

	return p, nil
}

func populateResources(name string, b *builder.FileBuilder, schema map[string]providers.Schema) *builder.MessageBuilder {
	msg := builder.NewMessage(name)
	for n, s := range schema {
		m := schemaToProtoMessage(capitalizeMessageName(n), s)
		b.TryAddMessage(m)
		f := builder.NewMapField(n, builder.FieldTypeString(), builder.FieldTypeMessage(m))
		f.SetJsonName(n)
		msg.AddField(f)
	}
	return msg
}

func findPlugin(pluginType, pluginName string) (*discovery.PluginMeta, error) {
	u, err := user.Current()
	if err != nil {
		return &discovery.PluginMeta{}, err
	}
	log.Println("homedir:", u.HomeDir)
	dir_postfix := filepath.Join(u.HomeDir, ".terraform.d", "plugins", fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH))
	dir, err := filepath.Abs(dir_postfix)
	if err != nil {
		return &discovery.PluginMeta{}, err
	}
	dirs := []string{dir}
	log.Println("dirs:", dirs)
	meta := discovery.FindPlugins(pluginType, dirs).WithName(pluginName)
	for c, _ := range meta {
		if c.Name == pluginName {
			return &c, nil
		}
	}
	return &discovery.PluginMeta{}, fmt.Errorf("could not find plugin %s", pluginName)
}

// NewGRPCClient creates a new GRPCClient instance.
func NewGRPCClient(config *plugin.ClientConfig) (providers.Interface, error) {
	// initialize a plugin client.
	pluginClient := plugin.NewClient(config)
	client, err := pluginClient.Client()
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize GRPC plugin: %s", err)
	}

	// create a new GRPCProvider.
	raw, err := client.Dispense(tfplugin.ProviderPluginName)
	if err != nil {
		return nil, fmt.Errorf("Failed to dispense GRPC plugin: %s", err)
	}

	switch provider := raw.(type) {
	// For Terraform v0.12+
	case *tfplugin.GRPCProvider:
		// To clean up the plugin process, we need to explicitly store references.
		provider.PluginClient = pluginClient

		return provider, nil

	default:
		return nil, fmt.Errorf("Failed to type cast GRPC plugin: %+v", raw)
	}
}

// newGRPCClientConfig returns a default plugin client config for Terraform v0.12+.
func newGRPCClientConfig(pluginMeta *discovery.PluginMeta) *plugin.ClientConfig {
	return tfplugin.ClientConfig(*pluginMeta)
}
