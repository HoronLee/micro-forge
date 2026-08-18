package main

import (
	"fmt"
	"strconv"

	pgs "github.com/lyft/protoc-gen-star/v2"
)

// importPaths extracts all the imports of the proto files and assign them
// unique alias for imports
func (m *Module) importPaths(file pgs.File) (path2Alias, alias2Path map[string]string) {
	// Add panic recovery
	defer m.recoverFromPanic("processing import paths")

	// Validate file
	if file == nil {
		m.Fail("Cannot process imports: file is nil")
		return nil, nil
	}

	// Initialize standard imports
	path2Alias = map[string]string{
		"google.golang.org/protobuf/proto": "proto",
	}
	alias2Path = map[string]string{
		"proto": "google.golang.org/protobuf/proto",
	}

	self := m.ctx.ImportPath(file).String()

	// Validate import path
	if err := m.validateImportPath(self); err != nil {
		m.Failf("Invalid file import path: %v", err)
		return path2Alias, alias2Path
	}
	for _, imp := range file.Imports() {
		// Validate import
		if imp == nil {
			m.Debug("Skipping nil import")
			continue
		}

		path := m.ctx.ImportPath(imp).String()

		// Validate import path
		if err := m.validateImportPath(path); err != nil {
			m.Debug(fmt.Sprintf("Skipping invalid import path: %v", err))
			continue
		}

		if self == path {
			// Skip self-imports
			continue
		}
		if _, ok := path2Alias[path]; ok {
			// already exist
			continue
		}

		// Only add imports that contain messages, enums, or services that might be used
		// Skip imports that only provide annotations or are metadata-only
		hasUsableTypes := len(imp.AllMessages()) > 0 || len(imp.AllEnums()) > 0 || len(imp.Services()) > 0
		if !hasUsableTypes {
			m.Debug(fmt.Sprintf("Skipping import %s: no usable types", path))
			continue
		}

		alias := m.ctx.PackageName(imp).String()

		// Validate package name
		if err := m.validatePackageName(alias); err != nil {
			m.Debug(fmt.Sprintf("Skipping import with invalid package name %s: %v", alias, err))
			continue
		}

		_, ok := alias2Path[alias]
		cnt := 0
		for ok {
			cnt++
			_, ok = alias2Path[alias+strconv.Itoa(cnt)]
		}
		if cnt > 0 {
			alias += strconv.Itoa(cnt)
			m.Debug(fmt.Sprintf("Resolved import alias conflict: %s -> %s", path, alias))
		}
		path2Alias[path] = alias
		alias2Path[alias] = path
	}
	return
}

// references lists all the import-references from different proto packages
// to suppress any unused import errors
func (m *Module) references(file pgs.File, nameWithAlias func(n pgs.Entity) string) []string {
	if file == nil {
		return nil
	}

	var list []string
	self := m.ctx.ImportPath(file)
	for _, imp := range file.Imports() {
		if imp == nil || m.ctx.ImportPath(imp) == self {
			continue
		}
		if len(imp.AllMessages()) > 0 {
			list = append(list, nameWithAlias(imp.AllMessages()[0]))
			continue
		}
		if len(imp.AllEnums()) > 0 {
			list = append(list, nameWithAlias(imp.AllEnums()[0]))
		}
	}
	return list
}
