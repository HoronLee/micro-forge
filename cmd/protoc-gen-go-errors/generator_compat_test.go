package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	errorsv1 "github.com/Servora-Kit/servora/api/gen/go/servora/errors/v1"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestGenerateGoGolden(t *testing.T) {
	t.Parallel()

	generated := runGenerator(t, compatibilityFixture(500))
	const filename = "example/v1/errors_errors.pb.go"
	content, ok := generated[filename]
	if !ok {
		t.Fatalf("generated files = %v, want %s", generated, filename)
	}
	golden, err := os.ReadFile(filepath.Join("testdata", "errors_errors.pb.go.golden"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if content != string(golden) {
		t.Fatalf("generated Go changed:\n--- got ---\n%s\n--- want ---\n%s", content, golden)
	}
}

func TestGenerateRejectsOutOfRangeCodeWithoutPartialOutput(t *testing.T) {
	t.Parallel()

	for _, code := range []int32{99, 600} {
		code := code
		t.Run(strconv.Itoa(int(code)), func(t *testing.T) {
			t.Parallel()

			plugin := compatibilityPlugin(t, compatibilityFixture(code), "paths=source_relative")
			err := generate(plugin, generatorTargetGo)
			if err == nil {
				t.Fatal("generate error = nil, want HTTP status range error")
			}
			if !strings.Contains(err.Error(), "outside HTTP status range 100..599") {
				t.Fatalf("generate error = %q, want HTTP status range context", err)
			}
			if files := plugin.Response().GetFile(); len(files) != 0 {
				t.Fatalf("generated partial files = %v, want none", files)
			}
		})
	}
}

func compatibilityFixture(defaultCode int32) *descriptorpb.FileDescriptorProto {
	enumOptions := &descriptorpb.EnumOptions{}
	proto.SetExtension(enumOptions, errorsv1.E_DefaultCode, defaultCode)
	invalidOptions := &descriptorpb.EnumValueOptions{}
	proto.SetExtension(invalidOptions, errorsv1.E_Code, int32(400))

	return &descriptorpb.FileDescriptorProto{
		Name:       proto.String("example/v1/errors.proto"),
		Package:    proto.String("example.v1"),
		Dependency: []string{"servora/errors/v1/errors.proto"},
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String("example.com/gen/example/v1;examplev1"),
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name:    proto.String("ErrorReason"),
				Options: enumOptions,
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: proto.String("ERROR_REASON_UNSPECIFIED"), Number: proto.Int32(0)},
					{Name: proto.String("ERROR_REASON_INVALID_INPUT"), Number: proto.Int32(1), Options: invalidOptions},
					{Name: proto.String("ERROR_REASON_INTERNAL"), Number: proto.Int32(2)},
				},
			},
		},
		SourceCodeInfo: &descriptorpb.SourceCodeInfo{
			Location: []*descriptorpb.SourceCodeInfo_Location{
				{
					Path:            []int32{5, 0, 2, 1},
					Span:            []int32{0, 0, 0},
					LeadingComments: proto.String(" Invalid input is rejected.\n"),
				},
			},
		},
	}
}

func compatibilityPlugin(t *testing.T, target *descriptorpb.FileDescriptorProto, parameter string) *protogen.Plugin {
	t.Helper()

	request := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{target.GetName()},
		Parameter:      proto.String(parameter),
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			protodesc.ToFileDescriptorProto(descriptorpb.File_google_protobuf_descriptor_proto),
			protodesc.ToFileDescriptorProto(errorsv1.File_servora_errors_v1_errors_proto),
			target,
		},
	}
	plugin, err := protogen.Options{}.New(request)
	if err != nil {
		t.Fatalf("protogen.Options.New: %v", err)
	}
	return plugin
}
