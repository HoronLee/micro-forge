// Package ruleplan builds deterministic generation plans for RPC rule plugins.
package ruleplan

import (
	"fmt"
	"path"
	"sort"

	"github.com/Servora-Kit/servora/cmd/internal/optionmerge"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Config describes the annotation extensions and domain-specific callbacks.
type Config[R proto.Message] struct {
	MethodExtension  protoreflect.ExtensionType
	ServiceExtension protoreflect.ExtensionType
	ValidateDeclared func(DeclarationContext, R) error
	AcceptMerged     func(MethodContext, R) (bool, error)
}

// DeclarationContext identifies one explicitly declared service or method rule.
type DeclarationContext struct {
	File    *protogen.File
	Service *protogen.Service
	Method  *protogen.Method
}

// Operation returns the complete RPC operation for method declarations.
func (context DeclarationContext) Operation() string {
	if context.Service == nil || context.Method == nil {
		return ""
	}
	return operation(context.Service, context.Method)
}

// MethodContext identifies the method receiving a merged rule.
type MethodContext struct {
	File      *protogen.File
	Service   *protogen.Service
	Method    *protogen.Method
	Operation string
}

// Entry is one accepted operation and its merged rule.
type Entry[R proto.Message] struct {
	Operation string
	Rule      R
}

// Group contains all accepted rules generated into one Go output directory.
type Group[R proto.Message] struct {
	Directory  string
	TargetFile *protogen.File
	Entries    []Entry[R]
}

type declaredRule[R proto.Message] struct {
	rule R
	has  bool
}

type serviceRules[R proto.Message] struct {
	serviceDefault R
	methods        map[protoreflect.Name]declaredRule[R]
}

type groupState[R proto.Message] struct {
	group Group[R]
	seen  map[string]struct{}
}

// Build resolves service defaults and method rules into deterministic groups.
func Build[R proto.Message](plugin *protogen.Plugin, config Config[R]) ([]Group[R], error) {
	if plugin == nil {
		return nil, fmt.Errorf("ruleplan: nil plugin")
	}
	if config.MethodExtension == nil {
		return nil, fmt.Errorf("ruleplan: method extension is required")
	}
	if config.ServiceExtension == nil {
		return nil, fmt.Errorf("ruleplan: service extension is required")
	}

	indexed := make(map[protoreflect.FullName]*serviceRules[R])
	for _, file := range plugin.Files {
		for _, service := range file.Services {
			fullName := service.Desc.FullName()
			rules := &serviceRules[R]{methods: make(map[protoreflect.Name]declaredRule[R])}

			serviceDefault, declared, err := extensionRule[R](service.Desc.Options(), config.ServiceExtension)
			if err != nil {
				return nil, fmt.Errorf("ruleplan: %s service %s default: %w", file.Desc.Path(), fullName, err)
			}
			if declared {
				if config.ValidateDeclared != nil {
					context := DeclarationContext{File: file, Service: service}
					if err := config.ValidateDeclared(context, serviceDefault); err != nil {
						return nil, err
					}
				}
				rules.serviceDefault = serviceDefault
			}

			for _, method := range service.Methods {
				methodRule, declared, err := extensionRule[R](method.Desc.Options(), config.MethodExtension)
				if err != nil {
					return nil, fmt.Errorf("ruleplan: %s operation %s: %w", file.Desc.Path(), operation(service, method), err)
				}
				if !declared {
					continue
				}
				if config.ValidateDeclared != nil {
					context := DeclarationContext{File: file, Service: service, Method: method}
					if err := config.ValidateDeclared(context, methodRule); err != nil {
						return nil, err
					}
				}
				rules.methods[method.Desc.Name()] = declaredRule[R]{rule: methodRule, has: true}
			}
			indexed[fullName] = rules
		}
	}

	targets := make(map[string]*protogen.File)
	states := make(map[string]*groupState[R])
	for _, file := range plugin.Files {
		if !file.Generate {
			continue
		}
		directory := path.Dir(file.GeneratedFilenamePrefix)
		target := targets[directory]
		if target == nil {
			targets[directory] = file
		} else if target.GoImportPath != file.GoImportPath || target.GoPackageName != file.GoPackageName {
			return nil, fmt.Errorf(
				"ruleplan: output directory %q has conflicting Go packages %q (%s) and %q (%s)",
				directory,
				target.GoImportPath,
				target.GoPackageName,
				file.GoImportPath,
				file.GoPackageName,
			)
		}
		for _, service := range file.Services {
			rules := indexed[service.Desc.FullName()]
			if rules == nil {
				continue
			}
			for _, method := range service.Methods {
				declared := rules.methods[method.Desc.Name()]
				merged, ok := optionmerge.Merge(rules.serviceDefault, declared.rule, declared.has)
				if !ok {
					continue
				}
				op := operation(service, method)
				context := MethodContext{File: file, Service: service, Method: method, Operation: op}
				if config.AcceptMerged != nil {
					accepted, err := config.AcceptMerged(context, merged)
					if err != nil {
						return nil, err
					}
					if !accepted {
						continue
					}
				}

				state := states[directory]
				if state == nil {
					state = &groupState[R]{
						group: Group[R]{Directory: directory, TargetFile: targets[directory]},
						seen:  make(map[string]struct{}),
					}
					states[directory] = state
				}
				if _, duplicate := state.seen[op]; duplicate {
					continue
				}
				state.seen[op] = struct{}{}
				state.group.Entries = append(state.group.Entries, Entry[R]{Operation: op, Rule: merged})
			}
		}
	}

	directories := make([]string, 0, len(states))
	for directory := range states {
		directories = append(directories, directory)
	}
	sort.Strings(directories)

	groups := make([]Group[R], 0, len(directories))
	for _, directory := range directories {
		group := states[directory].group
		sort.Slice(group.Entries, func(left, right int) bool {
			return group.Entries[left].Operation < group.Entries[right].Operation
		})
		groups = append(groups, group)
	}
	return groups, nil
}

func extensionRule[R proto.Message](options proto.Message, extension protoreflect.ExtensionType) (R, bool, error) {
	var zero R
	if options == nil || !proto.HasExtension(options, extension) {
		return zero, false, nil
	}
	value := proto.GetExtension(options, extension)
	rule, ok := value.(R)
	if !ok {
		return zero, false, fmt.Errorf("extension %s returned %T", extension.TypeDescriptor().FullName(), value)
	}
	if !validRule(rule) {
		return zero, false, nil
	}
	return rule, true, nil
}

func validRule[R proto.Message](rule R) bool {
	return proto.Message(rule) != nil && rule.ProtoReflect().IsValid()
}

func operation(service *protogen.Service, method *protogen.Method) string {
	return fmt.Sprintf("/%s/%s", service.Desc.FullName(), method.Desc.Name())
}
