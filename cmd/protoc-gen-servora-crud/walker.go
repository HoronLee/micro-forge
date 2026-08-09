package main

import (
	"github.com/Servora-Kit/servora/cmd/internal/protoreach"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type walkedField struct {
	Field  *protogen.Field
	Path   string
	Symbol string
}

func walkMessageFields(root *protogen.Message, visit func(walkedField) (bool, error)) error {
	if root == nil || visit == nil {
		return nil
	}
	ancestors := make(map[protoreflect.FullName]struct{})
	var walk func(*protogen.Message, string, string) error
	walk = func(message *protogen.Message, pathPrefix, symbolPrefix string) error {
		name := message.Desc.FullName()
		if _, cycle := ancestors[name]; cycle {
			return nil
		}
		ancestors[name] = struct{}{}
		defer delete(ancestors, name)

		for _, field := range message.Fields {
			path := string(field.Desc.Name())
			if pathPrefix != "" {
				path = pathPrefix + "." + path
			}
			symbol := symbolPrefix + goExportedName(string(field.Desc.Name()))
			descend, err := visit(walkedField{Field: field, Path: path, Symbol: symbol})
			if err != nil {
				return err
			}
			if !descend || field.Message == nil || field.Desc.IsList() || field.Desc.IsMap() || protoreach.IsWellKnown(field.Message.Desc) {
				continue
			}
			if err := walk(field.Message, path, symbol); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(root, "", "")
}
