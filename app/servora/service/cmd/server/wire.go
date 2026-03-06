//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"github.com/Servora-Kit/servora/api/gen/go/conf/v1"
	"github.com/Servora-Kit/servora/app/servora/service/internal/biz"
	"github.com/Servora-Kit/servora/app/servora/service/internal/data"
	"github.com/Servora-Kit/servora/app/servora/service/internal/server"
	"github.com/Servora-Kit/servora/app/servora/service/internal/service"
	"github.com/Servora-Kit/servora/pkg/bootstrap"
	"github.com/Servora-Kit/servora/pkg/transport/client"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Discovery, *conf.Registry, *conf.Data, *conf.App, *conf.Trace, *conf.Metrics, bootstrap.SvcIdentity, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, client.ProviderSet, newApp))
}
