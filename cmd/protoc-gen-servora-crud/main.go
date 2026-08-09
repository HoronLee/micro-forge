// Command protoc-gen-servora-crud generates runtime-independent CRUD companions.
package main

import (
	"flag"
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"
)

func main() {
	var target string
	flags := flag.NewFlagSet("protoc-gen-servora-crud", flag.ContinueOnError)
	flags.StringVar(&target, "target", "go", "generation target: go or ts")
	protogen.Options{ParamFunc: flags.Set}.Run(func(gen *protogen.Plugin) error {
		gen.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		return generate(gen, target)
	})
}

type fileResources struct {
	file      *protogen.File
	resources []*resourceInfo
}

type resourceCatalog struct {
	files   []fileResources
	ordered []*resourceInfo
}

func generate(gen *protogen.Plugin, target string) error {
	if target != "go" && target != "ts" {
		return fmt.Errorf("crud: unknown target %q (want go or ts)", target)
	}
	catalog, err := buildResourceCatalog(gen)
	if err != nil {
		return err
	}
	if err := validateStandardMethods(gen.Files, catalog.ordered); err != nil {
		return err
	}
	for _, group := range catalog.files {
		switch target {
		case "go":
			generateGoFile(gen, group.file, group.resources)
		case "ts":
			generateTypeScriptFile(gen, group.file, group.resources)
		}
	}
	return nil
}

func buildResourceCatalog(gen *protogen.Plugin) (resourceCatalog, error) {
	var catalog resourceCatalog
	for _, file := range gen.Files {
		if !file.Generate {
			continue
		}
		resources, err := discoverResources(file)
		if err != nil {
			return resourceCatalog{}, err
		}
		if len(resources) == 0 {
			continue
		}
		catalog.files = append(catalog.files, fileResources{file: file, resources: resources})
		catalog.ordered = append(catalog.ordered, resources...)
	}
	return catalog, nil
}
