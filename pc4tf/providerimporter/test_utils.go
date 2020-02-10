package providerimporter

import (
	"fmt"
	"io/ioutil"
	"log"
	"runtime"
	"strings"
	"text/template"

	getter "github.com/hashicorp/go-getter"
	"github.com/hashicorp/terraform/plugin/discovery"
)

func findPlugin(pluginType, pluginName, pluginVersion string) (*discovery.PluginMeta, error) {
	dir, err := ioutil.TempDir("", "test")
	if err != nil {
		return &discovery.PluginMeta{}, err
	}
	dirs := []string{dir}
	log.Println("dirs:", dirs)
	err = DownloadPlugin(dir, pluginName, pluginVersion)
	if err != nil {
		return &discovery.PluginMeta{}, err
	}
	meta := discovery.FindPlugins(pluginType, dirs).WithName(pluginName)
	for c := range meta {
		if c.Name == pluginName {
			return &c, nil
		}
	}
	return &discovery.PluginMeta{}, fmt.Errorf("could not find plugin %s", pluginName)
}

// DownloadPlugin will download a terraform probider if missing for testing purposes
func DownloadPlugin(dir, pluginName, pluginVersion string) error {
	templateURL := `https://releases.hashicorp.com/terraform-provider-{{.ProviderName}}/{{.Version}}/terraform-provider-{{.ProviderName}}_{{.Version}}_{{.Goos}}_{{.Goarch}}.zip`
	data := map[string]string{
		"ProviderName": pluginName,
		"Version":      pluginVersion,
		"Goos":         runtime.GOOS,
		"Goarch":       runtime.GOARCH,
	}
	t := template.Must(template.New("url").Parse(templateURL))
	builder := &strings.Builder{}
	err := t.Execute(builder, data)
	if err != nil {
		return err
	}
	return getter.Get(dir, builder.String())
}
