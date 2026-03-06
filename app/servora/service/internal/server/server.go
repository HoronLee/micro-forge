package server

import (
	"github.com/google/wire"
	"github.com/Servora-Kit/servora/app/servora/service/internal/server/middleware"
	"github.com/Servora-Kit/servora/pkg/governance/registry"
	"github.com/Servora-Kit/servora/pkg/governance/telemetry"
)

// ProviderSet is server providers.
var ProviderSet = wire.NewSet(middleware.ProviderSet, registry.NewRegistrar, telemetry.NewMetrics, NewGRPCMiddleware, NewGRPCServer, NewHTTPMiddleware, NewHTTPServer)
