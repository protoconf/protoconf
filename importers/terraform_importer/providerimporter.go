package terraformimporter

import (
	"fmt"
	"log"
	"sort"
	"strings"

	plugin "github.com/hashicorp/go-plugin"
	tfplugin "github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/hashicorp/terraform/providers"
	"github.com/jhump/protoreflect/desc/builder"
	"github.com/protoconf/protoconf/importers"
	"go.uber.org/zap"
)

// ProviderImporter queries a Terraform provider binary for its schema
// and returns a proto FileBuilder
type ProviderImporter struct {
	importer *importers.Importer
	meta     discovery.PluginMeta
	logger   *zap.Logger
}

// NewProviderImporter returns a ProviderImporter
func NewProviderImporter(meta discovery.PluginMeta, client providers.Interface, importer *importers.Importer) (*ProviderImporter, error) {
	p := &ProviderImporter{importer: importer, meta: meta}
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}
	p.logger = logger.With(zap.String("provider_name", meta.Name))
	// p.logger = zap.New.WithField("provider_name", name)
	p.logger.Info("invoking provider importer")

	schemaResponse := client.GetSchema()
	defer client.Close()
	tfmsg := importer.MasterFile.GetMessage("Terraform")
	resources := tfmsg.GetNestedMessage("Resources")
	datasources := tfmsg.GetNestedMessage("Datasources")
	providers := tfmsg.GetNestedMessage("Providers")

	p.populateResources(resources, schemaResponse.ResourceTypes)
	p.populateResources(datasources, schemaResponse.DataSources)
	providerFile := resourceFile(importer, meta.Name, "provider", string(meta.Version), meta.Name)
	providerConfigMsg := p.schemaToProtoMessage(capitalizeMessageName(meta.Name), schemaResponse.Provider)
	providerConfigMsg.AddField(builder.NewField("alias", builder.FieldTypeString()))
	providerFile.AddMessage(providerConfigMsg)
	providers.AddField(builder.NewField(meta.Name, builder.FieldTypeMessage(providerFile.GetMessage(providerConfigMsg.GetName()))).SetRepeated())

	return p, nil
}

func (p *ProviderImporter) populateResources(msg *builder.MessageBuilder, schema map[string]providers.Schema) *builder.MessageBuilder {
	keys := []string{}
	for n := range schema {
		keys = append(keys, n)
	}
	sort.Strings(keys)

	for _, n := range keys {
		s := schema[n]
		family := strings.Split(n, "_")[1]
		log.Println(p.importer, p.meta.Name, strings.ToLower(msg.GetName()), string(p.meta.Version), family)
		file := resourceFile(p.importer, p.meta.Name, strings.ToLower(msg.GetName()), string(p.meta.Version), family)
		m := p.schemaToProtoMessage(capitalizeMessageName(n), s)
		file.TryAddMessage(m)
		f := builder.NewMapField(n, builder.FieldTypeString(), builder.FieldTypeMessage(file.GetMessage(m.GetName())))
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
		return nil, fmt.Errorf("failed to initialize GRPC plugin: %s", err)
	}

	// create a new GRPCProvider.
	raw, err := client.Dispense(tfplugin.ProviderPluginName)
	if err != nil {
		return nil, fmt.Errorf("failed to dispense GRPC plugin: %s", err)
	}

	switch provider := raw.(type) {
	// For Terraform v0.12+
	case *tfplugin.GRPCProvider:
		// To clean up the plugin process, we need to explicitly store references.
		provider.PluginClient = pluginClient

		return provider, nil

	default:
		return nil, fmt.Errorf("failed to type cast GRPC plugin: %+v", raw)
	}
}

// newGRPCClientConfig returns a default plugin client config for Terraform v0.12+.
func newGRPCClientConfig(pluginMeta *discovery.PluginMeta) *plugin.ClientConfig {
	return tfplugin.ClientConfig(*pluginMeta)
}
