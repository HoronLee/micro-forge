package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	errorsv1 "github.com/Servora-Kit/servora/api/gen/go/servora/errors/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestGenerateDefaultsToGoAndMatchesExplicitGo(t *testing.T) {
	t.Parallel()

	fixture := compatibilityFixture(500)
	defaultOutput, err := generateWithParameter(t, fixture, "paths=source_relative")
	if err != nil {
		t.Fatalf("generate with default target: %v", err)
	}
	explicitOutput, err := generateWithParameter(t, fixture, "paths=source_relative,target=go")
	if err != nil {
		t.Fatalf("generate with target=go: %v", err)
	}
	if !reflect.DeepEqual(defaultOutput, explicitOutput) {
		t.Fatalf("default target output = %v, want explicit target=go output %v", defaultOutput, explicitOutput)
	}
}

func TestGenerateRejectsUnknownOrEmptyTargetWithoutPartialOutput(t *testing.T) {
	t.Parallel()

	for _, target := range []string{"", "javascript"} {
		target := target
		t.Run(fmt.Sprintf("target=%q", target), func(t *testing.T) {
			t.Parallel()

			generated, err := generateWithParameter(
				t,
				compatibilityFixture(500),
				"paths=source_relative,target="+target,
			)
			if err == nil {
				t.Fatal("generate error = nil, want unsupported target error")
			}
			if !strings.Contains(err.Error(), "target") {
				t.Fatalf("generate error = %q, want target context", err)
			}
			if len(generated) != 0 {
				t.Fatalf("generated partial files = %v, want none", generated)
			}
		})
	}
}

func TestGenerateRejectsOutOfRangeCodeForEveryTarget(t *testing.T) {
	t.Parallel()

	for _, target := range []string{generatorTargetGo, generatorTargetTypeScript} {
		target := target
		t.Run(target, func(t *testing.T) {
			t.Parallel()

			generated, err := generateWithParameter(
				t,
				compatibilityFixture(99),
				"paths=source_relative,target="+target,
			)
			if err == nil {
				t.Fatal("generate error = nil, want HTTP status range error")
			}
			if !strings.Contains(err.Error(), "outside HTTP status range 100..599") {
				t.Fatalf("generate error = %q, want HTTP status range context", err)
			}
			if len(generated) != 0 {
				t.Fatalf("generated partial files = %v, want none", generated)
			}
		})
	}
}

func TestGenerateSkipsUnannotatedAndNestedEnumsForEveryTarget(t *testing.T) {
	t.Parallel()

	for _, target := range []string{generatorTargetGo, generatorTargetTypeScript} {
		target := target
		t.Run(target, func(t *testing.T) {
			t.Parallel()

			generated, err := generateWithParameter(
				t,
				nonTopLevelErrorFixture(),
				"paths=source_relative,target="+target,
			)
			if err != nil {
				t.Fatalf("generate target=%s: %v", target, err)
			}
			if len(generated) != 0 {
				t.Fatalf("generated files = %v, want none", generated)
			}
		})
	}
}

func TestVersionFlagPrintsStableProcessOutput(t *testing.T) {
	command := exec.Command("go", "run", ".", "--version")
	command.Env = os.Environ()
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run . --version: %v\n%s", err, output)
	}
	if got, want := string(output), "protoc-gen-go-errors devel\n"; got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
}

func TestGenerateTypeScriptGolden(t *testing.T) {
	t.Parallel()

	generated, err := generateWithParameter(
		t,
		typeScriptFixture(),
		"paths=source_relative,target=ts",
	)
	if err != nil {
		t.Fatalf("generate target=ts: %v", err)
	}
	const filename = "example/service/v1/user.errors.ts"
	if len(generated) != 1 {
		t.Fatalf("generated files = %v, want only %s", generated, filename)
	}
	content, ok := generated[filename]
	if !ok {
		t.Fatalf("generated files = %v, want %s", generated, filename)
	}
	golden, err := os.ReadFile(filepath.Join("testdata", "user.errors.ts.golden"))
	if err != nil {
		t.Fatalf("read TypeScript golden: %v", err)
	}
	if content != string(golden) {
		t.Fatalf("generated TypeScript changed:\n--- got ---\n%s\n--- want ---\n%s", content, golden)
	}
	for _, forbidden := range []string{
		"HTTP",
		"Kratos",
		"code:",
		"400",
		"402",
		"500",
		"message",
		"transport",
		"Fetch",
		"Axios",
		"ofetch",
		"retry",
		"i18n",
		"Toast",
		" | '",
		"Object.values",
		"new Set",
	} {
		if strings.Contains(content, forbidden) {
			t.Errorf("generated TypeScript contains forbidden output %q:\n%s", forbidden, content)
		}
	}
}

func generateWithParameter(
	t *testing.T,
	target *descriptorpb.FileDescriptorProto,
	parameter string,
) (map[string]string, error) {
	t.Helper()

	options, selectedTarget := newGeneratorOptions()
	request := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{target.GetName()},
		Parameter:      proto.String(parameter),
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			protodesc.ToFileDescriptorProto(descriptorpb.File_google_protobuf_descriptor_proto),
			protodesc.ToFileDescriptorProto(errorsv1.File_servora_errors_v1_errors_proto),
			target,
		},
	}
	plugin, err := options.New(request)
	if err != nil {
		return nil, fmt.Errorf("protogen.Options.New: %w", err)
	}
	generationErr := generate(plugin, *selectedTarget)
	generated := make(map[string]string, len(plugin.Response().GetFile()))
	for _, file := range plugin.Response().GetFile() {
		generated[file.GetName()] = file.GetContent()
	}
	return generated, generationErr
}

func typeScriptFixture() *descriptorpb.FileDescriptorProto {
	defaultOptions := &descriptorpb.EnumOptions{}
	proto.SetExtension(defaultOptions, errorsv1.E_DefaultCode, int32(500))
	invalidNameOptions := &descriptorpb.EnumValueOptions{}
	proto.SetExtension(invalidNameOptions, errorsv1.E_Code, int32(400))
	declinedOptions := &descriptorpb.EnumValueOptions{}
	proto.SetExtension(declinedOptions, errorsv1.E_Code, int32(402))

	return &descriptorpb.FileDescriptorProto{
		Name:       proto.String("example/service/v1/user.proto"),
		Package:    proto.String("example.service.v1"),
		Dependency: []string{"servora/errors/v1/errors.proto"},
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/gen/example/service/v1;servicev1"),
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name:    proto.String("UserErrorReason"),
				Options: defaultOptions,
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: proto.String("USER_ERROR_REASON_UNSPECIFIED"), Number: proto.Int32(0)},
					{Name: proto.String("USER_ERROR_REASON_INVALID_NAME"), Number: proto.Int32(1), Options: invalidNameOptions},
					{Name: proto.String("USER_ERROR_REASON_INTERNAL"), Number: proto.Int32(2)},
				},
			},
			{
				Name: proto.String("Role"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: proto.String("ROLE_UNSPECIFIED"), Number: proto.Int32(0)},
					{Name: proto.String("ROLE_ADMIN"), Number: proto.Int32(1)},
				},
			},
			{
				Name: proto.String("PaymentErrorReason"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: proto.String("PAYMENT_ERROR_REASON_UNSPECIFIED"), Number: proto.Int32(0)},
					{Name: proto.String("PAYMENT_ERROR_REASON_DECLINED"), Number: proto.Int32(1), Options: declinedOptions},
				},
			},
		},
	}
}
func nonTopLevelErrorFixture() *descriptorpb.FileDescriptorProto {
	defaultOptions := &descriptorpb.EnumOptions{}
	proto.SetExtension(defaultOptions, errorsv1.E_DefaultCode, int32(500))

	return &descriptorpb.FileDescriptorProto{
		Name:       proto.String("example/service/v1/plain.proto"),
		Package:    proto.String("example.service.v1"),
		Dependency: []string{"servora/errors/v1/errors.proto"},
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/gen/example/service/v1;servicev1"),
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: proto.String("Plain"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: proto.String("PLAIN_UNSPECIFIED"), Number: proto.Int32(0)},
				},
			},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("Envelope"),
				EnumType: []*descriptorpb.EnumDescriptorProto{
					{
						Name:    proto.String("NestedErrorReason"),
						Options: defaultOptions,
						Value: []*descriptorpb.EnumValueDescriptorProto{
							{Name: proto.String("NESTED_ERROR_REASON_UNSPECIFIED"), Number: proto.Int32(0)},
						},
					},
				},
			},
		},
	}
}
