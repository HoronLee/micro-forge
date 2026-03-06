package server

import (
	"crypto/tls"

	"github.com/Servora-Kit/servora/api/gen/go/conf/v1"
	"github.com/Servora-Kit/servora/pkg/governance/telemetry"
	"github.com/Servora-Kit/servora/pkg/logger"

	"github.com/go-kratos/kratos/contrib/middleware/validate/v2"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"github.com/go-kratos/kratos/v2/middleware/ratelimit"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// GRPCMiddleware 用于 Wire 注入的中间件切片包装类型
type GRPCMiddleware []middleware.Middleware

// NewGRPCMiddleware 创建 gRPC 中间件
func NewGRPCMiddleware(
	trace *conf.Trace,
	mtc *telemetry.Metrics,
	l logger.Logger,
) GRPCMiddleware {
	grpcLogger := logger.With(l, logger.WithModule("grpc/server/servora-service"))

	var ms []middleware.Middleware
	ms = append(ms, recovery.Recovery())
	if trace != nil && trace.Endpoint != "" {
		ms = append(ms, tracing.Server())
	}
	ms = append(ms,
		logging.Server(grpcLogger),
		ratelimit.Server(),
		validate.ProtoValidate(),
	)
	if mtc != nil {
		ms = append(ms, metrics.Server(
			metrics.WithSeconds(mtc.Seconds),
			metrics.WithRequests(mtc.Requests),
		))
	}

	return ms
}

// NewGRPCServer new a gRPC server.
func NewGRPCServer(
	c *conf.Server,
	mw GRPCMiddleware,
	l logger.Logger,
) *grpc.Server {
	glog := logger.With(l, logger.WithModule("grpc/server/servora-service"))
	log := logger.NewHelper(glog)

	opts := []grpc.ServerOption{
		grpc.Middleware(mw...),
		grpc.Logger(glog),
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
			log.Fatalf("gRPC Server TLS: Failed to load key pair: %v", err)
		}
		creds := credentials.NewTLS(&tls.Config{Certificates: []tls.Certificate{cert}})
		opts = append(opts, grpc.Options(gogrpc.Creds(creds)))
	}

	srv := grpc.NewServer(opts...)

	// 注册服务

	return srv
}
