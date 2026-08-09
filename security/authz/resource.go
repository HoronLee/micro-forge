package authz

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func resolveResource(rule compiledRule, req any) (Resource, error) {
	resource := Resource{Type: rule.resourceType}
	if resource.Type == "" {
		return resource, fmt.Errorf("authz: resource_type is empty")
	}
	if rule.resourceIDField == "" {
		return resource, fmt.Errorf("authz: resource_id_field is empty")
	}

	id, err := extractProtoField(req, rule.resourceIDField)
	if err != nil {
		return resource, err
	}
	resource.ID = id
	return resource, nil
}

func extractProtoField(req any, fieldPath string) (string, error) {
	if fieldPath == "" {
		return "", fmt.Errorf("authz: resource_id_field is empty")
	}
	msg, ok := req.(proto.Message)
	if !ok || msg == nil {
		return "", fmt.Errorf("authz: request is not a protobuf message")
	}

	segments := strings.Split(fieldPath, ".")
	current := msg.ProtoReflect()
	for index, segment := range segments {
		if segment == "" {
			return "", fmt.Errorf("authz: resource_id_field %q contains an empty segment", fieldPath)
		}
		field := current.Descriptor().Fields().ByName(protoreflect.Name(segment))
		if field == nil {
			return "", fmt.Errorf("authz: field %q not found in %s", segment, current.Descriptor().FullName())
		}
		if field.IsList() || field.IsMap() {
			return "", fmt.Errorf("authz: field %q is repeated or map", segment)
		}

		last := index == len(segments)-1
		value := current.Get(field)
		if !last {
			if field.Kind() != protoreflect.MessageKind && field.Kind() != protoreflect.GroupKind {
				return "", fmt.Errorf("authz: path segment %q is not a message", segment)
			}
			current = value.Message()
			continue
		}
		if field.Kind() == protoreflect.MessageKind || field.Kind() == protoreflect.GroupKind {
			return "", fmt.Errorf("authz: path %q terminates on a message", fieldPath)
		}
		result := value.String()
		if result == "" {
			return "", fmt.Errorf("authz: field %q is empty", fieldPath)
		}
		return result, nil
	}
	return "", fmt.Errorf("authz: resource_id_field %q is invalid", fieldPath)
}
