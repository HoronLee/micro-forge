package main

import (
	"fmt"

	redact "github.com/Servora-Kit/servora/api/gen/go/servora/redact/v3"
	pgs "github.com/lyft/protoc-gen-star/v2"
)

// processFields extracts each fields information
func (m *Module) processFields(
	field pgs.Field,
	nameWithAlias func(n pgs.Entity) string,
) *FieldData {
	// Validate field before processing
	if err := m.validateField(field); err != nil {
		m.Failf("Invalid field: %v", err)
		return nil
	}

	typ := field.Type()
	if typ == nil {
		m.failWithContext(field, "field has nil type")
		return nil
	}

	// Determine if field will be a pointer in generated Go code
	// In proto3, fields with explicit `optional` keyword become pointers
	// These fields are implemented as synthetic oneofs (proto3_optional)
	// Exception: bytes fields are always []byte, never *[]byte, even with explicit optional
	hasExplicitOptional := field.InOneOf() && field.OneOf().IsSynthetic()
	isOptional := hasExplicitOptional && typ.ProtoType() != pgs.BytesT

	flData := &FieldData{
		Name:        m.ctx.Name(field).String(),
		IsMap:       typ.IsMap(),
		IsRepeated:  typ.IsRepeated(),
		IsMessage:   typ.IsEmbed(),
		IsOptional:  isOptional,
		FieldGoType: goTypeName(typ.ProtoType()),
	}
	em := typ.Embed()
	if em == nil {
		if ele := typ.Element(); ele != nil {
			em = ele.Embed()
		}
	}
	// embed message
	if em != nil {
		flData.EmbedMessageName = m.ctx.Name(em).String()
		flData.EmbedMessageNameWithAlias = nameWithAlias(em)
	}

	_redact, fieldRules := false, &redact.FieldRules{}
	// ok := m.must(field.Extension(redact.E_Redact, &_redact))
	ok := m.must(field.Extension(redact.E_Value, &fieldRules))

	// safe field: no option is defined
	if !ok {
		return flData
	}

	// Validate rules before processing
	if err := m.validateRules(fieldRules, field); err != nil {
		m.Fail(err)
		return flData
	}

	// check for custom field rules
	if fieldRules == nil || fieldRules.Values == nil {
		// no field rules
		if !_redact {
			// and redaction is also denied
			return flData
		}
		// default rules will be used
		flData.Redact = true
		flData.RedactionValue = RedactionDefaults(
			typ.ProtoType(),
			typ.IsRepeated() || typ.IsMap(),
		)
		if typ.IsEmbed() {
			flData.NestedEmbedCall = true
		}
		return flData
	}

	// custom field rules are defined, hence prefill defaults
	flData.Redact = true
	flData.RedactionValue = RedactionDefaults(
		typ.ProtoType(),
		typ.IsRepeated() || typ.IsMap(),
	)
	// custom values
	m.redactedCustomValue(flData, field, fieldRules)
	return flData
}

func (m *Module) redactedCustomValue(
	flData *FieldData,
	field pgs.Field,
	fieldRules *redact.FieldRules,
) {
	// Validate inputs
	if flData == nil {
		m.Failf("Internal error: nil FieldData for field %s", field.FullyQualifiedName())
		return
	}
	if fieldRules == nil {
		m.Failf("Internal error: nil fieldRules for field %s", field.FullyQualifiedName())
		return
	}

	typ := field.Type()
	if typ == nil {
		m.failWithContext(field, "field type is nil")
		return
	}

	// extract rule information
	info := m.RuleInformation(fieldRules)

	// match field types & rule types with better error message
	if info.ProtoType != 0 && info.ProtoType != typ.ProtoType() {
		err := m.validateTypeMatch(field, info.ProtoType, info.ProtoLabel)
		if err != nil {
			m.Fail(err)
		} else {
			m.failWithInvalidType(field)
		}
		return // unreachable
	}
	if typ.ProtoLabel() == pgs.Repeated && info.ProtoLabel != pgs.Repeated {
		err := m.validateTypeMatch(field, info.ProtoType, info.ProtoLabel)
		if err != nil {
			m.Fail(err)
		} else {
			m.failWithInvalidType(field)
		}
		return // unreachable
	}
	if info.ProtoType != pgs.MessageT && info.ProtoLabel != pgs.Repeated {
		// simple type fields
		flData.RedactionValue = fmt.Sprintf("%v", info.RedactionValue)
		return
	}

	// if message type
	if info.ProtoType == pgs.MessageT {
		messageRule, ok := fieldRules.Values.(*redact.FieldRules_Message)
		if !ok {
			m.Failf("Invalid message rule type for field %s", field.Name())
		}
		rule := messageRule.Message
		// default value is nil
		flData.RedactionValue = `nil`
		if rule.Empty {
			// flData.RedactionValue = m.ctx.Type(field).String() + "{}"
			flData.RedactionValue = fmt.Sprintf("&%s{}", flData.EmbedMessageNameWithAlias)
			return
		}
		if rule.Nil {
			flData.RedactionValue = "nil"
			return
		}
		if rule.Skip {
			flData.EmbedSkip = true
			return
		}
		flData.NestedEmbedCall = true
		return
	}

	// else info.ProtoLabel == pgs.Repeated
	elementRule, ok := fieldRules.Values.(*redact.FieldRules_Element)
	if !ok {
		m.Failf("Invalid element rule type for field %s", field.Name())
	}
	rule := elementRule.Element
	if rule.Empty {
		if flData.EmbedMessageNameWithAlias == "" {
			flData.RedactionValue = m.ctx.Type(field).String() + "{}"
			return
		}
		if flData.IsRepeated {
			flData.RedactionValue = fmt.Sprintf("[]*%s{}", flData.EmbedMessageNameWithAlias)
			return
		}
		// map type
		key := m.ctx.Type(field).Key().String()
		flData.RedactionValue = fmt.Sprintf("map[%s]*%s{}", key, flData.EmbedMessageNameWithAlias)
		return
	}
	if rule.Nested {
		// iterate over all items and redact with defaults
		flData.Iterate = true
		flData.RedactionValue = RedactionDefaults(typ.Element().ProtoType(), false)
		if typ.Element().IsEmbed() {
			flData.NestedEmbedCall = true
		}
		return
	}
	if rules := rule.Item; rules != nil && rules.Values != nil {
		if rules.GetElement() != nil {
			// Use the improved error message
			m.failWithNestedError(field)
			return
		}
		info := m.RuleInformation(rules)
		// match types
		if info.ProtoType != typ.Element().ProtoType() {
			m.failWithInvalidType(field)
			return // unreachable
		}
		// default value is nil
		flData.Iterate = true
		flData.RedactionValue = "nil"
		if info.ProtoType != pgs.MessageT {
			// simple type fields
			flData.RedactionValue = fmt.Sprintf("%v", info.RedactionValue)
		} else {
			// message type embedded field
			messageRule, ok := rules.Values.(*redact.FieldRules_Message)
			if !ok {
				m.Failf("Invalid message rule type for field %s", field.Name())
			}
			rule := messageRule.Message
			flData.RedactionValue = `nil`
			if rule.Empty {
				// flData.RedactionValue = m.ctx.Type(field).String() + "{}"
				flData.RedactionValue = fmt.Sprintf("&%s{}", flData.EmbedMessageNameWithAlias)
				return
			}
			if rule.Nil {
				flData.RedactionValue = "nil"
				return
			}
			if rule.Skip {
				flData.EmbedSkip = true
				return
			}
			flData.NestedEmbedCall = true
		}
	}
}

// RuleInfo response type for Module.RuleInformation
type RuleInfo struct {
	RedactionValue any
	// equivalent field type information
	ProtoType  pgs.ProtoType
	ProtoLabel pgs.ProtoLabel
}

// RuleInformation returns required information from the redact.FieldRules
func (m *Module) RuleInformation(rules *redact.FieldRules) (res RuleInfo) {
	// custom rules validation and values
	switch rule := rules.Values.(type) {
	case *redact.FieldRules_Float:
		res.ProtoType = pgs.FloatT
		res.RedactionValue = rule.Float
	case *redact.FieldRules_Double:
		res.ProtoType = pgs.DoubleT
		res.RedactionValue = rule.Double
	case *redact.FieldRules_Int32:
		res.ProtoType = pgs.Int32T
		res.RedactionValue = rule.Int32
	case *redact.FieldRules_Int64:
		res.ProtoType = pgs.Int64T
		res.RedactionValue = rule.Int64
	case *redact.FieldRules_Uint32:
		res.ProtoType = pgs.UInt32T
		res.RedactionValue = rule.Uint32
	case *redact.FieldRules_Uint64:
		res.ProtoType = pgs.UInt64T
		res.RedactionValue = rule.Uint64
	case *redact.FieldRules_Sint32:
		res.ProtoType = pgs.SInt32
		res.RedactionValue = rule.Sint32
	case *redact.FieldRules_Sint64:
		res.ProtoType = pgs.SInt64
		res.RedactionValue = rule.Sint64
	case *redact.FieldRules_Fixed32:
		res.ProtoType = pgs.Fixed32T
		res.RedactionValue = rule.Fixed32
	case *redact.FieldRules_Fixed64:
		res.ProtoType = pgs.Fixed64T
		res.RedactionValue = rule.Fixed64
	case *redact.FieldRules_Sfixed32:
		res.ProtoType = pgs.SFixed32
		res.RedactionValue = rule.Sfixed32
	case *redact.FieldRules_Sfixed64:
		res.ProtoType = pgs.SFixed64
		res.RedactionValue = rule.Sfixed64
	case *redact.FieldRules_Bool:
		res.ProtoType = pgs.BoolT
		res.RedactionValue = rule.Bool
	case *redact.FieldRules_String_:
		res.ProtoType = pgs.StringT
		res.RedactionValue = fmt.Sprintf("`%v`", rule.String_)
	case *redact.FieldRules_Bytes:
		res.ProtoType = pgs.BytesT
		res.RedactionValue = fmt.Sprintf("[]byte(`%v`)", string(rule.Bytes))
	case *redact.FieldRules_Enum:
		res.ProtoType = pgs.EnumT
		res.RedactionValue = rule.Enum
	case *redact.FieldRules_Message:
		res.ProtoType = pgs.MessageT
		if rule == nil || rule.Message == nil {
			m.Fail("(redact.custom).message is nil, no option defined")
			return // unreachable
		}
	case *redact.FieldRules_Element:
		res.ProtoLabel = pgs.Repeated
		if rule == nil || rule.Element == nil {
			m.Fail("(redact.custom).element is nil, no option defined")
			return // unreachable
		}
	default:
		m.Fail("Something went wrong")
	}
	return res
}

// goTypeName returns the Go type name for a proto type
func goTypeName(pt pgs.ProtoType) string {
	switch pt {
	case pgs.Int32T:
		return "int32"
	case pgs.Int64T:
		return "int64"
	case pgs.UInt32T:
		return "uint32"
	case pgs.UInt64T:
		return "uint64"
	case pgs.SInt32:
		return "int32"
	case pgs.SInt64:
		return "int64"
	case pgs.Fixed32T:
		return "uint32"
	case pgs.Fixed64T:
		return "uint64"
	case pgs.SFixed32:
		return "int32"
	case pgs.SFixed64:
		return "int64"
	case pgs.FloatT:
		return "float32"
	case pgs.DoubleT:
		return "float64"
	case pgs.BoolT:
		return "bool"
	case pgs.StringT:
		return "string"
	case pgs.BytesT:
		return "[]byte"
	default:
		return ""
	}
}
