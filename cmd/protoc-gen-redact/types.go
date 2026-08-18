package main

// ProtoFileData defines custom data type for Proto File info needed in template
type ProtoFileData struct {
	Source  string
	Package string
	// Imports: alias -> import-path
	Imports    map[string]string
	References []string
	Messages   []*MessageData
}

// MessageData defines custom data type for Message info needed in template
type MessageData struct {
	Name      string
	WithAlias string

	Fields []*FieldData
	Oneofs []*OneofData
}

// OneofData defines custom data type for a protobuf oneof group
type OneofData struct {
	Name   string            // Go name of the oneof field in the parent struct
	Fields []*OneofFieldData // Fields within this oneof
}

// HasRedactableFields returns true if at least one field in the oneof has redaction enabled
func (o *OneofData) HasRedactableFields() bool {
	for _, f := range o.Fields {
		if f.Redact {
			return true
		}
	}
	return false
}

// OneofFieldData wraps FieldData with oneof-specific information
type OneofFieldData struct {
	*FieldData
	WrapperTypeName string // Go wrapper type name (e.g., "MessageName_FieldName")
}

// FieldData defines custom data type for Field info needed in template
type FieldData struct {
	Name string
	// Redact using RedactionValue
	Redact         bool
	RedactionValue string
	FieldGoType    string // Go type for the field (e.g., "int32", "string", "bool")

	IsMap      bool // IsMap: true for Map types
	IsRepeated bool // IsRepeated: true for Repeated types
	IsMessage  bool // IsMessage: true for Message type(& not Repeated/Map)
	IsOptional bool // IsOptional: true for optional types

	// Iterate will only be used for Repeated/Map types and it specifies
	// whether or not to iterate each entry to be redacted
	Iterate bool

	// NestedEmbedCall will only be used for Message Types and it specifies
	// whether or not the embed message should be called for redaction.
	NestedEmbedCall bool

	// EmbedSkip will only be used for Message Types and it specifies
	// whether or not the embed message should be skipped.
	EmbedSkip bool

	// EmbedMessageName: name of embed message which is in case of Repeated or
	// Map or Message type field
	EmbedMessageName          string
	EmbedMessageNameWithAlias string
}
