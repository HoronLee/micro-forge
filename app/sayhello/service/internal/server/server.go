package server

import (
	"github.com/google/wire"
	"github.com/Servora-Kit/servora/pkg/governance/registry"
	"github.com/Servora-Kit/servora/pkg/governance/telemetry"
)

var ProviderSet = wire.NewSet(registry.NewRegistrar, telemetry.NewMetrics, NewGRPCServer)
