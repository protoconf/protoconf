package golangimporter

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jhump/protoreflect/desc/builder"
	"github.com/mitchellh/go-homedir"
	assert "github.com/stretchr/testify/require"
)

func TestGolangImporter(t *testing.T) {
	t.Skip("Skipped until revisit")
	outputdir, err := ioutil.TempDir("", "go_importer_test")
	assert.NoError(t, err, "failed to create tmp folder")
	userHome, err := homedir.Dir()
	assert.NoError(t, err, "could not get homedir")

	// oldPwd := os.Getenv("OLDPWD")
	// goSdkRoot := filepath.Join(strings.SplitN(oldPwd, "sandbox", 2)[0], "external", "go_sdk")
	goSdkRoot := os.Getenv("GOROOT")

	packageID := "html/template"
	t.Log("GOROOT", goSdkRoot)

	filepath.Walk(filepath.Join(goSdkRoot), func(path string, info os.FileInfo, err error) error {
		t.Log(path)
		return nil
	})
	i, err := NewGolangImporter(
		packageID,
		outputdir,
		filepath.Join(goSdkRoot, "src"),
		"HOME="+userHome,
		"PATH="+filepath.Join(goSdkRoot, "bin")+":"+os.Getenv("PATH"),
		// "GOROOT="+goSdkRoot,
	)
	assert.NoError(t, err, "failed to create GolangImporter")
	importer := i.GetImporter()
	// filtered := importer.FilterFiles("prometheus_config", "Config")
	// importer.Files = filtered
	err = importer.SaveAll()
	assert.NoError(t, err, "failed to save files")
}

func printProto(b builder.Builder, indent int) {
	indentStr := strings.Repeat("  ", indent)
	if b.GetParent() != nil {
		log.Println(indentStr, "in:", b.GetParent().GetName(), b.GetName())
	} else {
		log.Println(indentStr, "in:", b.GetName())
	}
	for _, child := range b.GetChildren() {
		printProto(child, indent+1)
		if field, ok := child.(*builder.FieldBuilder); ok {
			log.Println(indentStr, "type:", field.GetType().GetTypeName(), field.GetType())
			subType := field.GetType()
			if subType != nil {
				log.Println(indentStr, "file:", subType.GetType().String())
			}
		}
	}
}
