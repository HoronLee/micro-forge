package main

import (
	pgs "github.com/lyft/protoc-gen-star/v2"
)

// Note: must() and failWithInvalidType() have been moved to errors.go
// for better error handling and context

// RedactionDefaults returns the default value that can be used for the input
// pgs.Field for redaction. Predefined reduction defaults are:
//   - `0` for any number type
//   - `"REDACTED"` for string type
//   - `nil` for byte type
//   - `0th value` for enum type
//   - `nil` map for map type
//   - `nil` for repeated field type
//   - for message type, redaction is applied inside the message type
func RedactionDefaults(typ pgs.ProtoType, isRepeated bool) string {
	// isRepeated fields is for map or slice type fields
	if isRepeated {
		return "nil"
	}
	switch typ {
	case pgs.Int32T, pgs.Int64T,
		pgs.SInt32, pgs.SInt64,
		pgs.UInt32T, pgs.UInt64T,
		pgs.SFixed32, pgs.SFixed64,
		pgs.Fixed32T, pgs.Fixed64T,
		pgs.FloatT, pgs.DoubleT, pgs.EnumT:
		return "0"
	case pgs.BoolT:
		return "false"
	case pgs.StringT:
		return `"REDACTED"`
	case pgs.BytesT, pgs.GroupT:
		return "nil"
	case pgs.MessageT:
		return `-`
	default: // repeated and map
		return "nil"
	}
}

// ToCustomRule return redact proto' field rules based on their type
func ToCustomRule(typ pgs.ProtoType, lab pgs.ProtoLabel) string {
	if lab == pgs.Repeated {
		return "(redact.custom).element.*"
	}
	switch typ {
	case pgs.FloatT:
		return "(redact.custom).float"
	case pgs.DoubleT:
		return "(redact.custom).double"
	case pgs.Int32T:
		return "(redact.custom).int32"
	case pgs.Int64T:
		return "(redact.custom).int64"
	case pgs.UInt32T:
		return "(redact.custom).uint32"
	case pgs.UInt64T:
		return "(redact.custom).uint64"
	case pgs.SInt32:
		return "(redact.custom).sint32"
	case pgs.SInt64:
		return "(redact.custom).sint64"
	case pgs.Fixed32T:
		return "(redact.custom).fixed32"
	case pgs.Fixed64T:
		return "(redact.custom).fixed64"
	case pgs.SFixed32:
		return "(redact.custom).sfixed32"
	case pgs.SFixed64:
		return "(redact.custom).sfixed64"
	case pgs.BoolT:
		return "(redact.custom).bool"
	case pgs.StringT:
		return "(redact.custom).string"
	case pgs.BytesT:
		return "(redact.custom).bytes"
	case pgs.EnumT:
		return "(redact.custom).enum"
	case pgs.MessageT:
		return "(redact.custom).message.*"
	default:
		return "(redact.redact)"
	}
}
