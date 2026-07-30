package main

import (
	"flag"
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
)

var version = "devel"

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Printf("protoc-gen-go-errors %s\n", version)
		return
	}

	options, target := newGeneratorOptions()
	options.Run(func(plugin *protogen.Plugin) error {
		return generate(plugin, *target)
	})
}

func newGeneratorOptions() (protogen.Options, *string) {
	target := generatorTargetGo
	parameterFlags := flag.NewFlagSet("protoc-gen-go-errors", flag.ContinueOnError)
	parameterFlags.StringVar(&target, "target", generatorTargetGo, "generation target: go or ts")
	return protogen.Options{ParamFunc: parameterFlags.Set}, &target
}
