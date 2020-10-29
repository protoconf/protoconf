package terraformimporter

import (
	"fmt"
	"sort"

	plugin "github.com/hashicorp/go-plugin"
	tfplugin "github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/hashicorp/terraform/providers"
	"github.com/jhump/protoreflect/desc/builder"
	"go.uber.org/zap"
)

// ProviderImporter queries a Terraform provider binary for its schema
// and returns a proto FileBuilder
type ProviderImporter struct {
	Resources   *builder.FileBuilder
	Datasources *builder.FileBuilder
	Provider    *builder.FileBuilder
	config      *builder.MessageBuilder
	logger      *zap.Logger
}

// NewProviderImporter returns a ProviderImporter
func NewProviderImporter(name string, client providers.Interface) (*ProviderImporter, error) {
	p := &ProviderImporter{}
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}
	p.logger = logger.With(zap.String("provider_name", name))
	// p.logger = zap.New.WithField("provider_name", name)
	p.logger.Info("invoking provider importer")

	schemaResponse := client.GetSchema()
	defer client.Close()

	p.Resources = NewFile(name, "resources")
	p.Datasources = NewFile(name, "data")
	p.Provider = NewFile(name, "provider")

	p.Provider.AddMessage(p.populateResources("resources", p.Resources, schemaResponse.ResourceTypes))
	p.Provider.AddMessage(p.populateResources("data", p.Datasources, schemaResponse.DataSources))
	providerConfigMsg := p.schemaToProtoMessage(capitalizeMessageName(name), schemaResponse.Provider)
	providerConfigMsg.AddField(builder.NewField("alias", builder.FieldTypeString()))
	p.Provider.AddMessage(providerConfigMsg)
	providersMsg := builder.NewMessage("provider")
	providersMsg.AddField(builder.NewField(name, builder.FieldTypeMessage(providerConfigMsg)).SetRepeated())
	p.Provider.AddMessage(providersMsg)

	// p.Provider.AddMessage(p.populateResources("provider", p.Provider, map[string]providers.Schema{name: schemaResponse.Provider}))

	return p, nil
}

func (p *ProviderImporter) populateResources(name string, b *builder.FileBuilder, schema map[string]providers.Schema) *builder.MessageBuilder {
	log := p.logger.With(zap.String("file", name))
	log.Info("populating resources")
	p.logger = log
	msg := builder.NewMessage(name)
	keys := []string{}
	for n := range schema {
		keys = append(keys, n)
	}
	sort.Strings(keys)

	for _, n := range keys {
		s := schema[n]
		m := p.schemaToProtoMessage(capitalizeMessageName(n), s)
		b.TryAddMessage(m)
		f := builder.NewMapField(n, builder.FieldTypeString(), builder.FieldTypeMessage(m))
		f.SetJsonName(n)
		msg.AddField(f)
	}
	return msg
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
