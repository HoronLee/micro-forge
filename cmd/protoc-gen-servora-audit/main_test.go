package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	auditv1 "github.com/Servora-Kit/servora/api/gen/go/servora/audit/v1"
	"github.com/Servora-Kit/servora/cmd/internal/plugintest"
	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/wellknownimports"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// methodSpec describes a single RPC entry to materialize on a fake service.
type methodSpec struct {
	name string
	rule *auditv1.AuditRule // nil → no method-level option
}

// serviceSpec describes a single service in the fake proto file.
type serviceSpec struct {
	name           string
	serviceDefault *auditv1.AuditRule
	methods        []methodSpec
}

// fileSpec describes a single proto file to feed the plugin.
type fileSpec struct {
	name     string
	pkg      string
	goPkg    string
	generate bool
	services []serviceSpec
}

// runPluginScenario constructs a fake protogen.Plugin from the given files,
// invokes generate(), and returns the resulting plugin plus any generation error.
func runPluginScenario(t *testing.T, files []fileSpec) (*protogen.Plugin, error) {
	t.Helper()

	deps := plugintest.DescriptorClosure(auditv1.File_servora_audit_v1_annotations_proto)

	req := &pluginpb.CodeGeneratorRequest{
		ProtoFile: deps,
	}

	for _, fs := range files {
		fp := buildFileDescriptorProto(t, fs)
		req.ProtoFile = append(req.ProtoFile, fp)
		if fs.generate {
			req.FileToGenerate = append(req.FileToGenerate, fs.name)
		}
	}

	gen, err := protogen.Options{}.New(req)
	if err != nil {
		t.Fatalf("protogen.Options.New: %v", err)
	}

	return gen, generate(gen)
}

func buildFileDescriptorProto(t *testing.T, fs fileSpec) *descriptorpb.FileDescriptorProto {
	t.Helper()

	fp := &descriptorpb.FileDescriptorProto{
		Name:       proto.String(fs.name),
		Package:    proto.String(fs.pkg),
		Syntax:     proto.String(protoreflect.Proto3.String()),
		Dependency: []string{"google/protobuf/descriptor.proto", "servora/audit/v1/annotations.proto"},
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String(fs.goPkg),
		},
	}

	for _, svc := range fs.services {
		sp := &descriptorpb.ServiceDescriptorProto{Name: proto.String(svc.name)}
		if svc.serviceDefault != nil {
			opts := &descriptorpb.ServiceOptions{}
			proto.SetExtension(opts, auditv1.E_ServiceDefault, svc.serviceDefault)
			sp.Options = opts
		}
		for _, m := range svc.methods {
			mp := &descriptorpb.MethodDescriptorProto{
				Name:       proto.String(m.name),
				InputType:  proto.String("." + fs.pkg + ".Empty"),
				OutputType: proto.String("." + fs.pkg + ".Empty"),
			}
			if m.rule != nil {
				opts := &descriptorpb.MethodOptions{}
				proto.SetExtension(opts, auditv1.E_Rule, m.rule)
				mp.Options = opts
			}
			sp.Method = append(sp.Method, mp)
		}
		fp.Service = append(fp.Service, sp)
	}

	emptyMsg := &descriptorpb.DescriptorProto{Name: proto.String("Empty")}
	fp.MessageType = append(fp.MessageType, emptyMsg)

	return fp
}

func TestMethodExtensionContract(t *testing.T) {
	desc := auditv1.E_Rule.TypeDescriptor()
	if got, want := desc.FullName(), protoreflect.FullName("servora.audit.v1.rule"); got != want {
		t.Fatalf("method extension full name = %q; want %q", got, want)
	}
	if got, want := desc.Number(), protoreflect.FieldNumber(50100); got != want {
		t.Fatalf("method extension number = %d; want %d", got, want)
	}
	if _, err := protoregistry.GlobalTypes.FindExtensionByName("servora.audit.v1.audit_rule"); err == nil {
		t.Fatal("legacy servora.audit.v1.audit_rule extension must not be registered")
	}
}

func TestLegacyMethodOptionRejectedByCompiler(t *testing.T) {
	annotations, err := os.ReadFile(filepath.Join("..", "..", "api", "protos", "servora", "audit", "v1", "annotations.proto"))
	if err != nil {
		t.Fatalf("read audit annotations proto: %v", err)
	}
	const legacy = `syntax = "proto3";

package legacy.v1;

import "servora/audit/v1/annotations.proto";

option go_package = "example.com/legacy/v1;legacyv1";

message Request {}
message Response {}

service LegacyAuditService {
  rpc Call(Request) returns (Response) {
    option (servora.audit.v1.audit_rule) = {mode: AUDIT_MODE_ENABLED};
  }
}
`

	resolver := &protocompile.SourceResolver{
		Accessor: protocompile.SourceAccessorFromMap(map[string]string{
			"servora/audit/v1/annotations.proto": string(annotations),
			"legacy/v1/legacy.proto":             legacy,
		}),
	}
	compiler := protocompile.Compiler{
		Resolver: wellknownimports.WithStandardImports(resolver),
	}
	if _, err := compiler.Compile(t.Context(), "legacy/v1/legacy.proto"); err == nil {
		t.Fatal("legacy audit_rule fixture unexpectedly compiled")
	} else if want := "servora.audit.v1.audit_rule"; !strings.Contains(err.Error(), want) {
		t.Fatalf("legacy fixture failed for the wrong reason; want %q in error: %v", want, err)
	}
}

// ── basic behaviour ──────────────────────────────────────────────────────────

func TestNoAnnotations_NoFileGenerated(t *testing.T) {
	gen, err := runPluginScenario(t, []fileSpec{
		{
			name:     "example/v1/empty.proto",
			pkg:      "example.v1",
			goPkg:    "example.com/gen/example/v1;examplev1",
			generate: true,
			services: []serviceSpec{
				{
					name: "EmptyService",
					methods: []methodSpec{
						{name: "Noop"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("generate returned unexpected error: %v", err)
	}
	files := plugintest.ResponseFiles(gen)
	if len(files) != 0 {
		t.Fatalf("expected no generated files, got: %v", plugintest.SortedKeys(files))
	}
}

func TestMethodLevelEnabled_GoesToOutput(t *testing.T) {
	gen, err := runPluginScenario(t, []fileSpec{
		{
			name:     "example/v1/greeting.proto",
			pkg:      "example.v1",
			goPkg:    "example.com/gen/example/v1;examplev1",
			generate: true,
			services: []serviceSpec{
				{
					name: "GreetingService",
					methods: []methodSpec{
						{name: "Hello", rule: &auditv1.AuditRule{
							Mode: auditv1.AuditMode_AUDIT_MODE_ENABLED,
						}},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("generate returned unexpected error: %v", err)
	}
	content := plugintest.OnlyGeneratedFile(t, plugintest.ResponseFiles(gen), "audit_rules.gen.go")

	// Must contain the operation key.
	wantOp := `"/example.v1.GreetingService/Hello"`
	if !strings.Contains(content, wantOp) {
		t.Fatalf("audit rule missing operation key %s\n--- generated ---\n%s", wantOp, content)
	}
	// Must expose the AuditRules function.
	if !strings.Contains(content, "func AuditRules()") {
		t.Errorf("generated code should contain AuditRules()\n--- generated ---\n%s", content)
	}
	// Must set AUDIT_MODE_ENABLED.
	if !strings.Contains(content, "AUDIT_MODE_ENABLED") {
		t.Errorf("generated code should reference AUDIT_MODE_ENABLED\n--- generated ---\n%s", content)
	}
	// Must NOT reference old CompiledRule or BuildEvent.
	if strings.Contains(content, "CompiledRule") {
		t.Errorf("generated code must not reference CompiledRule\n--- generated ---\n%s", content)
	}
	if strings.Contains(content, "BuildEvent") {
		t.Errorf("generated code must not reference BuildEvent\n--- generated ---\n%s", content)
	}
}

func TestMethodLevelDisabled_NotEmitted(t *testing.T) {
	gen, err := runPluginScenario(t, []fileSpec{
		{
			name:     "example/v1/greeting.proto",
			pkg:      "example.v1",
			goPkg:    "example.com/gen/example/v1;examplev1",
			generate: true,
			services: []serviceSpec{
				{
					name: "GreetingService",
					methods: []methodSpec{
						{name: "Hello", rule: &auditv1.AuditRule{
							Mode: auditv1.AuditMode_AUDIT_MODE_DISABLED,
						}},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("generate returned unexpected error: %v", err)
	}
	files := plugintest.ResponseFiles(gen)
	if len(files) != 0 {
		t.Fatalf("expected no generated files for DISABLED rule, got: %v", plugintest.SortedKeys(files))
	}
}

// ── service default / merge semantics ────────────────────────────────────────

func TestServiceDefault_MethodInherits(t *testing.T) {
	gen, err := runPluginScenario(t, []fileSpec{
		{
			name:     "example/v1/greeting.proto",
			pkg:      "example.v1",
			goPkg:    "example.com/gen/example/v1;examplev1",
			generate: true,
			services: []serviceSpec{
				{
					name: "GreetingService",
					serviceDefault: &auditv1.AuditRule{
						Mode: auditv1.AuditMode_AUDIT_MODE_ENABLED,
					},
					methods: []methodSpec{
						{name: "Hello"}, // no method-level rule → inherits ENABLED
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("generate returned unexpected error: %v", err)
	}
	content := plugintest.OnlyGeneratedFile(t, plugintest.ResponseFiles(gen), "audit_rules.gen.go")
	wantKey := `"/example.v1.GreetingService/Hello"`
	if !strings.Contains(content, wantKey) {
		t.Fatalf("inherited rule missing operation key %s\n--- generated ---\n%s", wantKey, content)
	}
}

func TestServiceDefault_MethodUnspecifiedInherits(t *testing.T) {
	gen, err := runPluginScenario(t, []fileSpec{
		{
			name:     "example/v1/greeting.proto",
			pkg:      "example.v1",
			goPkg:    "example.com/gen/example/v1;examplev1",
			generate: true,
			services: []serviceSpec{
				{
					name: "GreetingService",
					serviceDefault: &auditv1.AuditRule{
						Mode: auditv1.AuditMode_AUDIT_MODE_ENABLED,
					},
					methods: []methodSpec{
						{name: "Hello", rule: &auditv1.AuditRule{
							// Mode UNSPECIFIED → inherits service default.
						}},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("generate returned unexpected error: %v", err)
	}
	content := plugintest.OnlyGeneratedFile(t, plugintest.ResponseFiles(gen), "audit_rules.gen.go")
	if !strings.Contains(content, `"/example.v1.GreetingService/Hello"`) {
		t.Fatalf("expected operation entry for inherited method\n--- generated ---\n%s", content)
	}
}

func TestMethodOverridesServiceDefault_DisabledWins(t *testing.T) {
	gen, err := runPluginScenario(t, []fileSpec{
		{
			name:     "example/v1/greeting.proto",
			pkg:      "example.v1",
			goPkg:    "example.com/gen/example/v1;examplev1",
			generate: true,
			services: []serviceSpec{
				{
					name: "GreetingService",
					serviceDefault: &auditv1.AuditRule{
						Mode: auditv1.AuditMode_AUDIT_MODE_ENABLED,
					},
					methods: []methodSpec{
						{name: "Hello"}, // inherits ENABLED
						{name: "Healthz", rule: &auditv1.AuditRule{
							Mode: auditv1.AuditMode_AUDIT_MODE_DISABLED,
						}},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("generate returned unexpected error: %v", err)
	}
	content := plugintest.OnlyGeneratedFile(t, plugintest.ResponseFiles(gen), "audit_rules.gen.go")
	if !strings.Contains(content, `"/example.v1.GreetingService/Hello"`) {
		t.Errorf("Hello should appear (inherits ENABLED)\n--- generated ---\n%s", content)
	}
	if strings.Contains(content, `"/example.v1.GreetingService/Healthz"`) {
		t.Errorf("Healthz should NOT appear (method DISABLED overrides service ENABLED)\n--- generated ---\n%s", content)
	}
}

func TestSameShortServiceNameAcrossPackages_DoesNotShareRules(t *testing.T) {
	gen, err := runPluginScenario(t, []fileSpec{
		{
			name:     "accounts/v1/user.proto",
			pkg:      "accounts.v1",
			goPkg:    "example.com/gen/accounts/v1;accountsv1",
			generate: true,
			services: []serviceSpec{
				{
					name: "UserService",
					serviceDefault: &auditv1.AuditRule{
						Mode: auditv1.AuditMode_AUDIT_MODE_ENABLED,
					},
					methods: []methodSpec{{name: "Get"}},
				},
			},
		},
		{
			name:     "admin/v1/user.proto",
			pkg:      "admin.v1",
			goPkg:    "example.com/gen/admin/v1;adminv1",
			generate: true,
			services: []serviceSpec{
				{
					name: "UserService",
					serviceDefault: &auditv1.AuditRule{
						Mode: auditv1.AuditMode_AUDIT_MODE_DISABLED,
					},
					methods: []methodSpec{{name: "Get"}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("generate returned unexpected error: %v", err)
	}

	files := plugintest.ResponseFiles(gen)
	accounts := files["example.com/gen/accounts/v1/audit_rules.gen.go"]
	if accounts == "" {
		t.Fatalf("expected generated audit file for accounts package, got: %v", plugintest.SortedKeys(files))
	}
	if _, ok := files["example.com/gen/admin/v1/audit_rules.gen.go"]; ok {
		t.Fatalf("admin service is disabled and must not emit rules, got files: %v", plugintest.SortedKeys(files))
	}
	if !strings.Contains(accounts, `"/accounts.v1.UserService/Get"`) {
		t.Fatalf("accounts rule missing full-name operation\n--- generated ---\n%s", accounts)
	}
	if strings.Contains(accounts, `"/admin.v1.UserService/Get"`) {
		t.Fatalf("accounts output leaked admin operation\n--- generated ---\n%s", accounts)
	}
}

// ── multi-service / multi-method ─────────────────────────────────────────────

func TestMultipleMethods_AllEnabledPresent(t *testing.T) {
	gen, err := runPluginScenario(t, []fileSpec{
		{
			name:     "example/v1/svc.proto",
			pkg:      "example.v1",
			goPkg:    "example.com/gen/example/v1;examplev1",
			generate: true,
			services: []serviceSpec{
				{
					name: "SvcA",
					methods: []methodSpec{
						{name: "Create", rule: &auditv1.AuditRule{Mode: auditv1.AuditMode_AUDIT_MODE_ENABLED}},
						{name: "Delete", rule: &auditv1.AuditRule{Mode: auditv1.AuditMode_AUDIT_MODE_ENABLED}},
						{name: "Get", rule: &auditv1.AuditRule{Mode: auditv1.AuditMode_AUDIT_MODE_DISABLED}},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("generate returned unexpected error: %v", err)
	}
	content := plugintest.OnlyGeneratedFile(t, plugintest.ResponseFiles(gen), "audit_rules.gen.go")
	for _, op := range []string{
		`"/example.v1.SvcA/Create"`,
		`"/example.v1.SvcA/Delete"`,
	} {
		if !strings.Contains(content, op) {
			t.Errorf("expected operation %s in output\n--- generated ---\n%s", op, content)
		}
	}
	if strings.Contains(content, `"/example.v1.SvcA/Get"`) {
		t.Errorf("DISABLED Get should not appear\n--- generated ---\n%s", content)
	}
}

func TestGeneratedFile_HasCorrectHeader(t *testing.T) {
	gen, err := runPluginScenario(t, []fileSpec{
		{
			name:     "example/v1/svc.proto",
			pkg:      "example.v1",
			goPkg:    "example.com/gen/example/v1;examplev1",
			generate: true,
			services: []serviceSpec{
				{
					name: "Svc",
					methods: []methodSpec{
						{name: "Op", rule: &auditv1.AuditRule{Mode: auditv1.AuditMode_AUDIT_MODE_ENABLED}},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("generate returned unexpected error: %v", err)
	}
	content := plugintest.OnlyGeneratedFile(t, plugintest.ResponseFiles(gen), "audit_rules.gen.go")
	if !strings.Contains(content, "Code generated by protoc-gen-servora-audit") {
		t.Errorf("missing generated-code header\n--- generated ---\n%s", content)
	}
	if !strings.Contains(content, "DO NOT EDIT") {
		t.Errorf("missing DO NOT EDIT marker\n--- generated ---\n%s", content)
	}
}

func TestGeneratedFileCompiles(t *testing.T) {
	gen, err := runPluginScenario(t, []fileSpec{{
		name:     "example/v1/svc.proto",
		pkg:      "example.v1",
		goPkg:    "example.com/gen/example/v1;examplev1",
		generate: true,
		services: []serviceSpec{{
			name: "Service",
			serviceDefault: &auditv1.AuditRule{
				Mode: auditv1.AuditMode_AUDIT_MODE_ENABLED,
			},
			methods: []methodSpec{
				{name: "Create"},
				{name: "Healthz", rule: &auditv1.AuditRule{Mode: auditv1.AuditMode_AUDIT_MODE_DISABLED}},
			},
		}},
	}})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	source := plugintest.OnlyGeneratedFile(t, plugintest.ResponseFiles(gen), "audit_rules.gen.go")
	plugintest.AssertGeneratedGoCompiles(t, source, "examplev1")
}
