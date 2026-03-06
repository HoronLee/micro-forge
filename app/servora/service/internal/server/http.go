package server

import (
	"crypto/tls"

	"github.com/go-kratos/kratos/contrib/middleware/validate/v2"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"github.com/go-kratos/kratos/v2/middleware/ratelimit"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/Servora-Kit/servora/api/gen/go/conf/v1"
	servorav1 "github.com/Servora-Kit/servora/api/gen/go/servora/service/v1"
	"github.com/Servora-Kit/servora/app/servora/service/internal/consts"
	mwinter "github.com/Servora-Kit/servora/app/servora/service/internal/server/middleware"
	"github.com/Servora-Kit/servora/app/servora/service/internal/service"
	"github.com/Servora-Kit/servora/pkg/governance/telemetry"
	"github.com/Servora-Kit/servora/pkg/logger"
	mwpkg "github.com/Servora-Kit/servora/pkg/middleware"
	"github.com/Servora-Kit/servora/pkg/middleware/cors"
)

// HTTPMiddleware 用于 Wire 注入的中间件切片包装类型
type HTTPMiddleware []middleware.Middleware

// NewHTTPMiddleware 创建 HTTP 中间件（使用白名单机制）
func NewHTTPMiddleware(
	trace *conf.Trace,
	m *telemetry.Metrics,
	l logger.Logger,
	authJWT mwinter.AuthJWT,
) HTTPMiddleware {
	httpLogger := logger.With(l, logger.WithModule("http/server/servora-service"))

	var ms []middleware.Middleware
	ms = append(ms, recovery.Recovery())
	if trace != nil && trace.Endpoint != "" {
		ms = append(ms, tracing.Server())
	}
	ms = append(ms,
		logging.Server(httpLogger),
		ratelimit.Server(),
		validate.ProtoValidate(),
	)
	if m != nil {
		ms = append(ms, metrics.Server(
			metrics.WithSeconds(m.Seconds),
			metrics.WithRequests(m.Requests),
		))
	}

	// 公开接口白名单（无需认证）
	publicWhitelist := mwpkg.NewWhiteList(mwpkg.Exact,
		servorav1.OperationAuthServiceLoginByEmailPassword,
		servorav1.OperationAuthServiceRefreshToken,
		servorav1.OperationAuthServiceSignupByEmail,
		servorav1.OperationTestServiceTest,
		servorav1.OperationTestServiceHello,
	)

	// User 级接口白名单（需要 User 权限但跳过 Admin 检查）
	userWhitelist := mwpkg.NewWhiteList(mwpkg.Exact,
		servorav1.OperationUserServiceCurrentUserInfo,
		servorav1.OperationUserServiceUpdateUser,
		servorav1.OperationAuthServiceLogout,
		servorav1.OperationTestServicePrivateTest,
	)

	// Admin 权限排除白名单 = 公开接口 ∪ User 级接口
	adminExcludeWhitelist := publicWhitelist.Merge(userWhitelist)

	ms = append(ms,
		selector.Server(authJWT(consts.User)).
			Match(publicWhitelist.MatchFunc()).
			Build(),
		selector.Server(authJWT(consts.Admin)).
			Match(adminExcludeWhitelist.MatchFunc()).
			Build(),
	)

	return ms
}

// NewHTTPServer new an HTTP server.
func NewHTTPServer(
	c *conf.Server,
	mw HTTPMiddleware,
	mtc *telemetry.Metrics,
	l logger.Logger,
	auth *service.AuthService,
	user *service.UserService,
	test *service.TestService,
) *http.Server {
	hlog := logger.With(l, logger.WithModule("http/server/servora-service"))
	log := logger.NewHelper(hlog)

	var opts = []http.ServerOption{
		http.Middleware(mw...),
		http.Logger(hlog),
	}
	if c != nil && c.Http != nil {
		if c.Http.Network != "" {
			opts = append(opts, http.Network(c.Http.Network))
		}
		if c.Http.Addr != "" {
			opts = append(opts, http.Address(c.Http.Addr))
		}
		if c.Http.Timeout != nil {
			opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
		}
		if cors.IsEnabled(c.Http.Cors) {
			opts = append(opts, http.Filter(cors.Middleware(c.Http.Cors)))
			log.Infof("CORS middleware enabled: allowed_origins=%v", cors.GetAllowedOrigins(c.Http.Cors))
		}
	}
	if c != nil && c.Http != nil && c.Http.Tls != nil && c.Http.Tls.Enable {
		if c.Http.Tls.CertPath == "" || c.Http.Tls.KeyPath == "" {
			log.Fatal("Server TLS: can't find TLS key pairs")
		}
		cert, err := tls.LoadX509KeyPair(c.Http.Tls.CertPath, c.Http.Tls.KeyPath)
		if err != nil {
			log.Fatalf("Server TLS: Failed to load key pair: %v", err)
		}
		opts = append(opts, http.TLSConfig(&tls.Config{Certificates: []tls.Certificate{cert}}))
	}

	srv := http.NewServer(opts...)

	if mtc != nil {
		srv.Handle("/metrics", mtc.Handler)
	}

	servorav1.RegisterAuthServiceHTTPServer(srv, auth)
	servorav1.RegisterUserServiceHTTPServer(srv, user)
	servorav1.RegisterTestServiceHTTPServer(srv, test)

	return srv
}
