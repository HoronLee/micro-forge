package main

import (
	"fmt"

	pgs "github.com/lyft/protoc-gen-star/v2"

	redact "github.com/Servora-Kit/servora/api/gen/go/servora/redact/v3"
)

// Process processes the file and adds its generated code into Module.Artifacts
func (m *Module) Process(file pgs.File) {
	if err := m.validateFile(file); err != nil {
		m.Failf("Cannot process file: %v", err)
		return
	}
	defer m.recoverFromPanic(fmt.Sprintf("processing file %s", file.Name()))

	fileSkip := false
	m.must(file.Extension(redact.E_FileSkip, &fileSkip))
	if fileSkip {
		return
	}

	path2Alias, alias2Path := m.importPaths(file)
	nameWithAlias := func(n pgs.Entity) string {
		imp := m.ctx.ImportPath(n).String()
		name := m.ctx.Name(n).String()
		if alias := path2Alias[imp]; alias != "" {
			name = alias + "." + name
		}
		return name
	}

	data := &ProtoFileData{
		Source:     file.Name().String(),
		Package:    m.ctx.PackageName(file).String(),
		Imports:    alias2Path,
		References: m.references(file, nameWithAlias),
		Messages:   make([]*MessageData, 0, len(file.AllMessages())),
	}

	hasRedaction := false
	for _, msg := range file.AllMessages() {
		message := m.processMessage(msg, nameWithAlias)
		data.Messages = append(data.Messages, message)
		if message != nil {
			for _, field := range message.Fields {
				hasRedaction = hasRedaction || field.Redact
			}
			for _, oneof := range message.Oneofs {
				hasRedaction = hasRedaction || oneof.HasRedactableFields()
			}
		}
	}
	if !hasRedaction {
		return
	}

	name := m.ctx.OutputPath(file).SetExt(".redact.go")
	m.AddGeneratorTemplateFile(name.String(), m.tmpl, data)
}

// processMessage extracts all pgs.Message and their pgs.Field(s) information and
// structures them into MessageData
func (m *Module) processMessage(
	msg pgs.Message,
	nameWithAlias func(n pgs.Entity) string,
) *MessageData {
	// Validate message before processing
	if err := m.validateMessage(msg); err != nil {
		m.Failf("Cannot process message: %v", err)
		return nil
	}

	defer m.recoverFromPanic(fmt.Sprintf("processing message %s", msg.FullyQualifiedName()))

	msgData := &MessageData{
		Name:      m.ctx.Name(msg).String(),
		WithAlias: nameWithAlias(msg),
		Fields:    make([]*FieldData, 0, len(msg.Fields())*2),
	}

	for _, field := range msg.Fields() {
		// Skip fields that belong to non-synthetic oneofs;
		// they are processed as part of OneofData below
		if field.InOneOf() && !field.OneOf().IsSynthetic() {
			continue
		}
		msgData.Fields = append(msgData.Fields, m.processFields(field, nameWithAlias))
	}

	// Process non-synthetic oneofs (real oneof groups)
	for _, oneOf := range msg.RealOneOfs() {
		oneofData := &OneofData{
			Name:   m.ctx.Name(oneOf).String(),
			Fields: make([]*OneofFieldData, 0, len(oneOf.Fields())),
		}
		for _, field := range oneOf.Fields() {
			fieldData := m.processFields(field, nameWithAlias)
			oneofData.Fields = append(oneofData.Fields, &OneofFieldData{
				FieldData:       fieldData,
				WrapperTypeName: msgData.Name + "_" + fieldData.Name,
			})
		}
		msgData.Oneofs = append(msgData.Oneofs, oneofData)
	}
	return msgData
}
