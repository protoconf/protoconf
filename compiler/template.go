package compiler

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	sprig "github.com/Masterminds/sprig/v3"
	"github.com/protoconf/protoconf/consts"
)

const DefaultTemplate = `
"""
Generated from {{ .TemplateFilename }}
"""
{{ range .Files -}}
load("mutable:{{.MutableName}}", {{.VariableName}}="value")
{{ end }}
ALL = {
{{- range .Files }}
  "{{.VariableName}}": {{.VariableName}},
{{- end }}
}
`

var variableRegex = regexp.MustCompile("[^a-zA-Z0-9]+")

type file struct {
	MutableName  string
	VariableName string
}

type input struct {
	TemplateFilename string
	Files            []file
}

func normalizeFilepath(file string) string {
	return filepath.Join(filepath.SplitList(file)...)
}

func evaluateTemplateFile(filename string) error {
	filename = normalizeFilepath(filename)
	templateInput := &input{
		TemplateFilename: filename,
		Files:            []file{},
	}
	abs, err := filepath.Abs(filepath.Join(consts.SrcPath, filename))
	if err != nil {
		return err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return fmt.Errorf("cannot read file %s, err: %v", abs, err)
	}

	dir := filepath.Join(consts.MutableConfigPath, filepath.Dir(filename))
	err = filepath.Walk(dir, func(mutablePath string, _ os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		if !strings.HasSuffix(mutablePath, consts.CompiledConfigExtension) {
			return nil
		}
		mutableName := strings.TrimSuffix(
			strings.TrimPrefix(mutablePath, consts.MutableConfigPath),
			consts.CompiledConfigExtension,
		)
		variableName := variableRegex.ReplaceAllString(mutableName, "_")
		templateInput.Files = append(templateInput.Files, file{
			MutableName:  mutableName,
			VariableName: variableName,
		})
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed while walking through %s, err: %v", dir, err)
	}

	t, err := template.New("template").Funcs(sprig.FuncMap()).Parse(string(data))
	if err != nil {
		return fmt.Errorf("failed to evaluate template, file: %s, err: %v", abs, err)
	}

	targetFile := filepath.Join(consts.SrcPath, strings.TrimSuffix(filename, `.template`))

	log.Println("writing to", targetFile)
	fileWriter, err := os.Create(targetFile)
	if err != nil {
		return fmt.Errorf("failed to open file %s for writing, err: %v", targetFile, err)
	}
	return t.Execute(fileWriter, templateInput)
}

func findTemplateFilesAndProccess() error {
	log.Println("processing template")
	return filepath.Walk(consts.SrcPath, func(filepath string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(filepath, consts.TemplateExtension) {
			log.Println("found", filepath)
			return evaluateTemplateFile(strings.TrimPrefix(filepath, consts.SrcPath))
		}
		return nil
	})
}
