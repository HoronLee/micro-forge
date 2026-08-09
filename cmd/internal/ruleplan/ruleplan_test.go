package ruleplan

import (
	"strings"
	"testing"

	auditv1 "github.com/Servora-Kit/servora/api/gen/go/servora/audit/v1"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

type testMethod struct {
	name string
	rule *auditv1.AuditRule
}

type testService struct {
	name           string
	serviceDefault *auditv1.AuditRule
	methods        []testMethod
}

type testFile struct {
	name     string
	pkg      string
	goPkg    string
	generate bool
	services []testService
}

func TestBuildMergesFiltersGroupsAndSorts(t *testing.T) {
	plugin := newTestPlugin(t, []testFile{
		{
			name: "example/v1/first.proto", pkg: "example.v1", goPkg: "example.com/gen/example/v1;examplev1", generate: true,
			services: []testService{{
				name:           "FirstService",
				serviceDefault: &auditv1.AuditRule{Mode: auditv1.AuditMode_AUDIT_MODE_ENABLED},
				methods: []testMethod{
					{name: "Zulu"},
					{name: "Alpha"},
					{name: "Healthz", rule: &auditv1.AuditRule{Mode: auditv1.AuditMode_AUDIT_MODE_DISABLED}},
				},
			}},
		},
		{
			name: "example/v1/second.proto", pkg: "example.v1", goPkg: "example.com/gen/example/v1;examplev1", generate: true,
			services: []testService{{
				name:    "SecondService",
				methods: []testMethod{{name: "Create", rule: &auditv1.AuditRule{Mode: auditv1.AuditMode_AUDIT_MODE_ENABLED}}},
			}},
		},
		{
			name: "admin/v1/service.proto", pkg: "admin.v1", goPkg: "example.com/gen/admin/v1;adminv1", generate: true,
			services: []testService{{
				name:           "FirstService",
				serviceDefault: &auditv1.AuditRule{Mode: auditv1.AuditMode_AUDIT_MODE_DISABLED},
				methods:        []testMethod{{name: "Alpha"}},
			}},
		},
	})

	groups, err := Build(plugin, Config[*auditv1.AuditRule]{
		MethodExtension:  auditv1.E_Rule,
		ServiceExtension: auditv1.E_ServiceDefault,
		AcceptMerged: func(_ MethodContext, rule *auditv1.AuditRule) (bool, error) {
			return rule.GetMode() == auditv1.AuditMode_AUDIT_MODE_ENABLED, nil
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("groups = %d, want 1", len(groups))
	}
	group := groups[0]
	if group.Directory != "example/v1" {
		t.Fatalf("directory = %q, want example/v1", group.Directory)
	}
	want := []string{
		"/example.v1.FirstService/Alpha",
		"/example.v1.FirstService/Zulu",
		"/example.v1.SecondService/Create",
	}
	if len(group.Entries) != len(want) {
		t.Fatalf("entries = %d, want %d", len(group.Entries), len(want))
	}
	for index, operation := range want {
		if group.Entries[index].Operation != operation {
			t.Fatalf("entry[%d] = %q, want %q", index, group.Entries[index].Operation, operation)
		}
	}
}

func TestBuildMethodUnspecifiedInheritsWholeServiceRule(t *testing.T) {
	plugin := newTestPlugin(t, []testFile{{
		name: "example/v1/service.proto", pkg: "example.v1", goPkg: "example.com/gen/example/v1;examplev1", generate: true,
		services: []testService{{
			name:           "Service",
			serviceDefault: &auditv1.AuditRule{Mode: auditv1.AuditMode_AUDIT_MODE_ENABLED},
			methods:        []testMethod{{name: "Get", rule: &auditv1.AuditRule{Mode: auditv1.AuditMode_AUDIT_MODE_UNSPECIFIED}}},
		}},
	}})

	groups, err := Build(plugin, Config[*auditv1.AuditRule]{
		MethodExtension:  auditv1.E_Rule,
		ServiceExtension: auditv1.E_ServiceDefault,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(groups) != 1 || len(groups[0].Entries) != 1 {
		t.Fatalf("groups = %#v, want one entry", groups)
	}
	if got := groups[0].Entries[0].Rule.GetMode(); got != auditv1.AuditMode_AUDIT_MODE_ENABLED {
		t.Fatalf("mode = %v, want ENABLED", got)
	}
}

func TestBuildNoEffectiveRulesReturnsNoGroups(t *testing.T) {
	plugin := newTestPlugin(t, []testFile{{
		name: "example/v1/service.proto", pkg: "example.v1", goPkg: "example.com/gen/example/v1;examplev1", generate: true,
		services: []testService{{name: "Service", methods: []testMethod{{name: "Get"}}}},
	}})
	groups, err := Build(plugin, Config[*auditv1.AuditRule]{
		MethodExtension:  auditv1.E_Rule,
		ServiceExtension: auditv1.E_ServiceDefault,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(groups) != 0 {
		t.Fatalf("groups = %d, want 0", len(groups))
	}
}

func TestBuildRejectsConflictingGoPackagesInDirectory(t *testing.T) {
	plugin := newTestPlugin(t, []testFile{
		{
			name: "example/v1/first.proto", pkg: "example.v1", goPkg: "example.com/one;one", generate: true,
			services: []testService{{
				name:    "First",
				methods: []testMethod{{name: "Get", rule: &auditv1.AuditRule{Mode: auditv1.AuditMode_AUDIT_MODE_ENABLED}}},
			}},
		},
		{
			name: "example/v1/second.proto", pkg: "example.v1", goPkg: "example.com/two;two", generate: true,
			services: []testService{{
				name:    "Second",
				methods: []testMethod{{name: "Get"}},
			}},
		},
	})

	_, err := Build(plugin, Config[*auditv1.AuditRule]{
		MethodExtension:  auditv1.E_Rule,
		ServiceExtension: auditv1.E_ServiceDefault,
	})
	if err == nil {
		t.Fatal("Build succeeded with conflicting Go packages")
	}
	for _, want := range []string{"example/v1", "example.com/one", "example.com/two"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q missing %q", err, want)
		}
	}
}

func TestBuildValidatesDeclaredRulesBeforeMerge(t *testing.T) {
	plugin := newTestPlugin(t, []testFile{{
		name: "example/v1/service.proto", pkg: "example.v1", goPkg: "example.com/gen/example/v1;examplev1", generate: true,
		services: []testService{{
			name:    "Service",
			methods: []testMethod{{name: "Get", rule: &auditv1.AuditRule{Mode: auditv1.AuditMode_AUDIT_MODE_ENABLED}}},
		}},
	}})
	_, err := Build(plugin, Config[*auditv1.AuditRule]{
		MethodExtension:  auditv1.E_Rule,
		ServiceExtension: auditv1.E_ServiceDefault,
		ValidateDeclared: func(context DeclarationContext, _ *auditv1.AuditRule) error {
			if context.Method != nil {
				return testError("declared " + context.Operation())
			}
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "declared /example.v1.Service/Get") {
		t.Fatalf("error = %v", err)
	}
}

type testError string

func TestBuildRejectsMissingInputs(t *testing.T) {
	plugin := newTestPlugin(t, nil)
	tests := []struct {
		name   string
		plugin *protogen.Plugin
		config Config[*auditv1.AuditRule]
		want   string
	}{
		{name: "nil plugin", want: "nil plugin"},
		{name: "missing method extension", plugin: plugin, config: Config[*auditv1.AuditRule]{ServiceExtension: auditv1.E_ServiceDefault}, want: "method extension"},
		{name: "missing service extension", plugin: plugin, config: Config[*auditv1.AuditRule]{MethodExtension: auditv1.E_Rule}, want: "service extension"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Build(test.plugin, test.config)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestValidRuleRejectsNilAndTypedNil(t *testing.T) {
	var nilRule *auditv1.AuditRule
	if validRule(nilRule) {
		t.Fatal("typed-nil rule is valid")
	}
	if !validRule(&auditv1.AuditRule{}) {
		t.Fatal("zero-value protobuf rule is invalid")
	}
}

func (err testError) Error() string { return string(err) }

func newTestPlugin(t *testing.T, files []testFile) *protogen.Plugin {
	t.Helper()
	req := &pluginpb.CodeGeneratorRequest{
		ProtoFile: descriptorClosure(auditv1.File_servora_audit_v1_annotations_proto),
		Parameter: proto.String("paths=source_relative"),
	}
	for _, file := range files {
		req.ProtoFile = append(req.ProtoFile, testFileDescriptor(file))
		if file.generate {
			req.FileToGenerate = append(req.FileToGenerate, file.name)
		}
	}
	plugin, err := protogen.Options{}.New(req)
	if err != nil {
		t.Fatalf("protogen.Options.New: %v", err)
	}
	return plugin
}

func testFileDescriptor(file testFile) *descriptorpb.FileDescriptorProto {
	messageName := "Empty"
	if len(file.services) > 0 {
		messageName = file.services[0].name + "Request"
	}
	descriptor := &descriptorpb.FileDescriptorProto{
		Name:        proto.String(file.name),
		Package:     proto.String(file.pkg),
		Syntax:      proto.String("proto3"),
		Dependency:  []string{"google/protobuf/descriptor.proto", "servora/audit/v1/annotations.proto"},
		Options:     &descriptorpb.FileOptions{GoPackage: proto.String(file.goPkg)},
		MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String(messageName)}},
	}
	for _, service := range file.services {
		serviceDescriptor := &descriptorpb.ServiceDescriptorProto{Name: proto.String(service.name)}
		if service.serviceDefault != nil {
			options := &descriptorpb.ServiceOptions{}
			proto.SetExtension(options, auditv1.E_ServiceDefault, service.serviceDefault)
			serviceDescriptor.Options = options
		}
		for _, method := range service.methods {
			methodDescriptor := &descriptorpb.MethodDescriptorProto{
				Name:       proto.String(method.name),
				InputType:  proto.String("." + file.pkg + "." + messageName),
				OutputType: proto.String("." + file.pkg + "." + messageName),
			}
			if method.rule != nil {
				options := &descriptorpb.MethodOptions{}
				proto.SetExtension(options, auditv1.E_Rule, method.rule)
				methodDescriptor.Options = options
			}
			serviceDescriptor.Method = append(serviceDescriptor.Method, methodDescriptor)
		}
		descriptor.Service = append(descriptor.Service, serviceDescriptor)
	}
	return descriptor
}

func descriptorClosure(root protoreflect.FileDescriptor) []*descriptorpb.FileDescriptorProto {
	seen := make(map[string]bool)
	var descriptors []*descriptorpb.FileDescriptorProto
	var visit func(protoreflect.FileDescriptor)
	visit = func(file protoreflect.FileDescriptor) {
		if seen[file.Path()] {
			return
		}
		seen[file.Path()] = true
		imports := file.Imports()
		for index := range imports.Len() {
			visit(imports.Get(index).FileDescriptor)
		}
		descriptors = append(descriptors, protodesc.ToFileDescriptorProto(file))
	}
	visit(root)
	return descriptors
}
