package main

import (
	"flag"

	"github.com/Servora-Kit/servora/pkg/bootstrap"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/grpc"

	_ "go.uber.org/automaxprocs"
)

var (
	Name     = "sayhello.service"
	Version  = "v1.0.0"
	flagconf string
)

func init() {
	flag.StringVar(&flagconf, "conf", "./configs", "config path, eg: -conf config.yaml")
}

func newApp(identity bootstrap.SvcIdentity, l log.Logger, reg registry.Registrar, gs *grpc.Server) *kratos.App {
	return kratos.New(
		kratos.ID(identity.ID),
		kratos.Name(identity.Name),
		kratos.Version(identity.Version),
		kratos.Metadata(identity.Metadata),
		kratos.Logger(l),
		kratos.Server(gs),
		kratos.Registrar(reg),
	)
}

func main() {
	flag.Parse()

	err := bootstrap.BootstrapAndRun(flagconf, Name, Version, func(runtime *bootstrap.Runtime) (*kratos.App, func(), error) {
		bc := runtime.Bootstrap
		return wireApp(bc.Server, bc.Registry, bc.App, bc.Trace, bc.Metrics, runtime.Identity, runtime.Logger)
	})
	if err != nil {
		panic(err)
	}
}
