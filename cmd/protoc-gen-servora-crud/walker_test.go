package main

import (
	"reflect"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestWalkMessageFieldsTerminatesCycles(t *testing.T) {
	tests := []struct {
		name     string
		messages []*descriptorpb.DescriptorProto
		root     string
		want     []string
	}{
		{
			name: "self reference",
			messages: []*descriptorpb.DescriptorProto{
				messageDescriptor("Node",
					scalarField("value", 1),
					messageField("child", 2, ".test.v1.Node"),
				),
			},
			root: "Node",
			want: []string{"value", "child"},
		},
		{
			name: "mutual reference",
			messages: []*descriptorpb.DescriptorProto{
				messageDescriptor("A", scalarField("a_value", 1), messageField("b", 2, ".test.v1.B")),
				messageDescriptor("B", scalarField("b_value", 1), messageField("a", 2, ".test.v1.A")),
			},
			root: "A",
			want: []string{"a_value", "b", "b.b_value", "b.a"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := protogenMessage(t, test.messages, test.root)
			var paths []string
			err := walkMessageFields(root, func(field walkedField) (bool, error) {
				paths = append(paths, field.Path)
				return true, nil
			})
			if err != nil {
				t.Fatalf("walkMessageFields: %v", err)
			}
			if !reflect.DeepEqual(paths, test.want) {
				t.Fatalf("paths = %v, want %v", paths, test.want)
			}
		})
	}
}

func TestWalkMessageFieldsKeepsDiamondPathsAndSupportsPruning(t *testing.T) {
	messages := []*descriptorpb.DescriptorProto{
		messageDescriptor("Root",
			messageField("left", 1, ".test.v1.Node"),
			messageField("right", 2, ".test.v1.Node"),
			messageField("pruned", 3, ".test.v1.Node"),
		),
		messageDescriptor("Node", scalarField("leaf", 1)),
	}
	root := protogenMessage(t, messages, "Root")
	var paths []string
	err := walkMessageFields(root, func(field walkedField) (bool, error) {
		paths = append(paths, field.Path)
		return field.Path != "pruned", nil
	})
	if err != nil {
		t.Fatalf("walkMessageFields: %v", err)
	}
	want := []string{"left", "left.leaf", "right", "right.leaf", "pruned"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths = %v, want %v", paths, want)
	}
}

func protogenMessage(t *testing.T, messages []*descriptorpb.DescriptorProto, name string) *protogen.Message {
	t.Helper()
	file := &descriptorpb.FileDescriptorProto{
		Name:        new("test/v1/walker.proto"),
		Package:     new("test.v1"),
		Syntax:      new("proto3"),
		Options:     &descriptorpb.FileOptions{GoPackage: new("example.com/test/v1;testv1")},
		MessageType: messages,
	}
	plugin, err := protogen.Options{}.New(&pluginpb.CodeGeneratorRequest{
		ProtoFile:      []*descriptorpb.FileDescriptorProto{file},
		FileToGenerate: []string{file.GetName()},
	})
	if err != nil {
		t.Fatalf("protogen.Options.New: %v", err)
	}
	for _, message := range plugin.Files[0].Messages {
		if message.Desc.Name() == protoreflectName(name) {
			return message
		}
	}
	t.Fatalf("message %s not found", name)
	return nil
}

func protoreflectName(value string) protoreflect.Name {
	return protoreflect.Name(value)
}

func messageDescriptor(name string, fields ...*descriptorpb.FieldDescriptorProto) *descriptorpb.DescriptorProto {
	return &descriptorpb.DescriptorProto{Name: new(name), Field: fields}
}

func scalarField(name string, number int32) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:   new(name),
		Number: new(number),
		Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
	}
}

func messageField(name string, number int32, typeName string) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name:     new(name),
		Number:   new(number),
		Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
		TypeName: new(typeName),
	}
}
