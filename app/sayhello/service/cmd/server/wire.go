//go:build wireinject
// +build wireinject

package main

import (
	"github.com/Servora-Kit/servora/api/gen/go/conf/v1"
	"github.com/Servora-Kit/servora/app/sayhello/service/internal/server"
	"github.com/Servora-Kit/servora/app/sayhello/service/internal/service"
	"github.com/Servora-Kit/servora/pkg/bootstrap"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

func wireApp(*conf.Server, *conf.Registry, *conf.App, *conf.Trace, *conf.Metrics, bootstrap.SvcIdentity, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, service.ProviderSet, newApp))
}
