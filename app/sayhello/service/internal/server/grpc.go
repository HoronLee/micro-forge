package server

import (
	"crypto/tls"

	"github.com/Servora-Kit/servora/api/gen/go/conf/v1"
	sayhellov1 "github.com/Servora-Kit/servora/api/gen/go/sayhello/service/v1"
	"github.com/Servora-Kit/servora/app/sayhello/service/internal/service"
	"github.com/Servora-Kit/servora/pkg/governance/telemetry"
	"github.com/Servora-Kit/servora/pkg/logger"

	"github.com/go-kratos/kratos/contrib/middleware/validate/v2"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func NewGRPCServer(c *conf.Server, trace *conf.Trace, mtc *telemetry.Metrics, l logger.Logger, sayhello *service.SayHelloService) *grpc.Server {
	helper := logger.NewHelper(l)
	var mds []middleware.Middleware
	mds = []middleware.Middleware{recovery.Recovery()}
	if trace != nil && trace.Endpoint != "" {
		mds = append(mds, tracing.Server())
	}
	mds = append(mds,
		logging.Server(l),
		validate.ProtoValidate(),
	)
	if mtc != nil {
		mds = append(mds, metrics.Server(
			metrics.WithSeconds(mtc.Seconds),
			metrics.WithRequests(mtc.Requests),
		))
	}

	var opts = []grpc.ServerOption{
		grpc.Middleware(mds...),
	}

	if c != nil && c.Grpc != nil {
		if c.Grpc.Network != "" {
			opts = append(opts, grpc.Network(c.Grpc.Network))
		}
		if c.Grpc.Addr != "" {
			opts = append(opts, grpc.Address(c.Grpc.Addr))
		}
		if c.Grpc.Timeout != nil {
			opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
		}
	}

	if c != nil && c.Grpc != nil && c.Grpc.Tls != nil && c.Grpc.Tls.Enable {
		cert, err := tls.LoadX509KeyPair(c.Grpc.Tls.CertPath, c.Grpc.Tls.KeyPath)
		if err != nil {
			helper.Fatalf("gRPC Server TLS: Failed to load key pair: %v", err)
		}
		creds := credentials.NewTLS(&tls.Config{Certificates: []tls.Certificate{cert}})
		opts = append(opts, grpc.Options(gogrpc.Creds(creds)))
	}

	srv := grpc.NewServer(opts...)
	sayhellov1.RegisterSayHelloServiceServer(srv, sayhello)
	return srv
}
