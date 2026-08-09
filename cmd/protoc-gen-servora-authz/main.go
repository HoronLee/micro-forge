// Command protoc-gen-servora-authz translates servora authz proto annotations
// into a Go file (`authz_rules.gen.go`) consumed by the runtime to enforce
// authorization on RPC methods.
//
// Merge semantics (matches authn / audit):
//   - method-level rule with mode != AUTHZ_MODE_UNSPECIFIED replaces the
//     service-level default in its entirety,
//   - method-level rule absent (or mode == AUTHZ_MODE_UNSPECIFIED) inherits
//     the service-level default,
//   - only methods whose merged mode != AUTHZ_MODE_UNSPECIFIED appear in the
//     generated map (NONE is preserved so callers can express "explicitly
//     skip" rather than "no rule"; the runtime decides what to do with NONE).
//
// Cross-file template scanning: rules are gathered from ALL input files
// (including non-generated dependencies). Templates are keyed by proto
// fully-qualified service name so same-named services in different packages do
// not share rules. Generated output groups by output directory so each
// directory yields one authz_rules.gen.go covering the services declared in it.
package main

import (
	"fmt"
	"path"
	"strings"

	authzpb "github.com/Servora-Kit/servora/api/gen/go/servora/authz/v1"
	"github.com/Servora-Kit/servora/cmd/internal/ruleplan"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/pluginpb"
)

func main() {
	protogen.Options{}.Run(func(gen *protogen.Plugin) error {
		gen.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		return generate(gen)
	})
}

// generate is the testable entry point.
func generate(gen *protogen.Plugin) error {
	groups, err := ruleplan.Build(gen, ruleplan.Config[*authzpb.AuthzRule]{
		MethodExtension:  authzpb.E_Rule,
		ServiceExtension: authzpb.E_ServiceDefault,
		ValidateDeclared: validateDeclaredRule,
		AcceptMerged: func(context ruleplan.MethodContext, rule *authzpb.AuthzRule) (bool, error) {
			if err := validateCheckRule(context.Operation, rule, context.Method.Input.Desc); err != nil {
				return false, err
			}
			return true, nil
		},
	})
	if err != nil {
		return err
	}
	for _, group := range groups {
		file := group.TargetFile
		generated := gen.NewGeneratedFile(
			path.Join(group.Directory, "authz_rules.gen.go"),
			file.GoImportPath,
		)
		generateFile(generated, file.GoPackageName, group.Entries)
	}
	return nil
}

func validateDeclaredRule(context ruleplan.DeclarationContext, rule *authzpb.AuthzRule) error {
	if rule == nil {
		return nil
	}
	switch rule.GetMode() {
	case authzpb.AuthzMode_AUTHZ_MODE_UNSPECIFIED,
		authzpb.AuthzMode_AUTHZ_MODE_NONE,
		authzpb.AuthzMode_AUTHZ_MODE_CHECK:
		return nil
	default:
		method := "<service_default>"
		if context.Method != nil {
			method = string(context.Method.Desc.Name())
		}
		return fmt.Errorf(
			"authz: %s service %s method %s: unknown AuthzMode %d",
			context.File.Desc.Path(), context.Service.Desc.FullName(), method, rule.GetMode(),
		)
	}
}

func validateCheckRule(operation string, rule *authzpb.AuthzRule, input protoreflect.MessageDescriptor) error {
	if rule.GetMode() != authzpb.AuthzMode_AUTHZ_MODE_CHECK {
		return nil
	}
	action := rule.GetAction()
	if strings.TrimSpace(action) == "" {
		return fmt.Errorf("authz: operation %q CHECK rule requires action", operation)
	}
	if action != strings.TrimSpace(action) {
		return fmt.Errorf("authz: operation %q action must not contain leading or trailing whitespace", operation)
	}
	resourceType := rule.GetResourceType()
	if strings.TrimSpace(resourceType) == "" {
		return fmt.Errorf("authz: operation %q CHECK rule requires resource_type", operation)
	}
	if resourceType != strings.TrimSpace(resourceType) {
		return fmt.Errorf("authz: operation %q resource_type must not contain leading or trailing whitespace", operation)
	}
	fieldPath := rule.GetResourceIdField()
	if strings.TrimSpace(fieldPath) == "" {
		return fmt.Errorf("authz: operation %q CHECK rule requires resource_id_field", operation)
	}
	if fieldPath != strings.TrimSpace(fieldPath) {
		return fmt.Errorf("authz: operation %q resource_id_field must not contain leading or trailing whitespace", operation)
	}

	current := input
	segments := strings.Split(fieldPath, ".")
	for index, segment := range segments {
		if segment == "" {
			return fmt.Errorf("authz: operation %q resource_id_field %q contains an empty segment", operation, fieldPath)
		}
		field := current.Fields().ByName(protoreflect.Name(segment))
		if field == nil {
			return fmt.Errorf("authz: operation %q resource_id_field %q: field %q not found in %s", operation, fieldPath, segment, current.FullName())
		}
		if field.IsList() || field.IsMap() {
			return fmt.Errorf("authz: operation %q resource_id_field %q: field %q is repeated or map", operation, fieldPath, segment)
		}

		last := index == len(segments)-1
		if !last {
			if field.Kind() != protoreflect.MessageKind && field.Kind() != protoreflect.GroupKind {
				return fmt.Errorf("authz: operation %q resource_id_field %q: intermediate field %q is not a message", operation, fieldPath, segment)
			}
			current = field.Message()
			continue
		}
		if field.Kind() == protoreflect.MessageKind || field.Kind() == protoreflect.GroupKind {
			return fmt.Errorf("authz: operation %q resource_id_field %q terminates on a message", operation, fieldPath)
		}
	}
	return nil
}

func generateFile(g *protogen.GeneratedFile, pkgName protogen.GoPackageName, rules []ruleplan.Entry[*authzpb.AuthzRule]) {
	authzPkg := protogen.GoImportPath("github.com/Servora-Kit/servora/api/gen/go/servora/authz/v1")
	protoPkg := protogen.GoImportPath("google.golang.org/protobuf/proto")

	g.P("// Code generated by protoc-gen-servora-authz. DO NOT EDIT.")
	g.P()
	g.P("package ", pkgName)
	g.P()

	authzRule := g.QualifiedGoIdent(protogen.GoIdent{GoName: "AuthzRule", GoImportPath: authzPkg})
	protoClone := g.QualifiedGoIdent(protogen.GoIdent{GoName: "Clone", GoImportPath: protoPkg})

	g.P("// _authzRules is the immutable backing store for AuthzRules.")
	g.P("var _authzRules = map[string]*", authzRule, "{")
	for _, entry := range rules {
		rule := entry.Rule
		modeIdent := g.QualifiedGoIdent(protogen.GoIdent{
			GoName:       "AuthzMode_" + rule.GetMode().String(),
			GoImportPath: authzPkg,
		})
		g.P(fmt.Sprintf("\t%q: {", entry.Operation))
		g.P("\t\tMode: ", modeIdent, ",")
		if rule.GetAction() != "" {
			g.P(fmt.Sprintf("\t\tAction: %q,", rule.GetAction()))
		}
		if rule.GetResourceType() != "" {
			g.P(fmt.Sprintf("\t\tResourceType: %q,", rule.GetResourceType()))
		}
		if rule.GetResourceIdField() != "" {
			g.P(fmt.Sprintf("\t\tResourceIdField: %q,", rule.GetResourceIdField()))
		}
		g.P("\t},")
	}
	g.P("}")
	g.P()
	g.P("// AuthzRules returns a copy of the authorization rules for this package.")
	g.P("// Each call returns a fresh map and cloned rule messages.")
	g.P("func AuthzRules() map[string]*", authzRule, " {")
	g.P("	m := make(map[string]*", authzRule, ", len(_authzRules))")
	g.P("	for k, v := range _authzRules {")
	g.P("		m[k] = ", protoClone, "(v).(*", authzRule, ")")
	g.P("	}")
	g.P("	return m")
	g.P("}")
}
