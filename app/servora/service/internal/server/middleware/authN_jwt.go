package middleware

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	authpb "github.com/Servora-Kit/servora/api/gen/go/auth/service/v1"
	"github.com/Servora-Kit/servora/api/gen/go/conf/v1"
	"github.com/Servora-Kit/servora/app/servora/service/internal/biz"
	"github.com/Servora-Kit/servora/app/servora/service/internal/consts"
	"github.com/Servora-Kit/servora/pkg/jwt"
)

// AuthJWT 定义认证中间件生成器函数类型
type AuthJWT func(minRole consts.UserRole) middleware.Middleware

// NewAuthMiddleware 创建认证中间件生成器
func NewAuthMiddleware(appConf *conf.App) AuthJWT {
	return func(minRole consts.UserRole) middleware.Middleware {
		return func(handler middleware.Handler) middleware.Handler {
			return func(ctx context.Context, req any) (reply any, err error) {
				tr, ok := transport.FromServerContext(ctx)
				if !ok {
					return nil, authpb.ErrorMissingToken("missing transport context")
				}
				authHeader := tr.RequestHeader().Get("Authorization")
				tokenString := strings.TrimPrefix(authHeader, "Bearer ")

				// 如果未设置 minRole（即为 0），允许无 token 访问
				if minRole == 0 && tokenString == "" {
					return handler(ctx, req)
				}

				if tokenString == "" {
					return nil, authpb.ErrorMissingToken("missing Authorization header")
				}

				// 创建JWT实例并解析Token
				jwtInstance := jwt.NewJWT[biz.UserClaims](&jwt.Config{
					SecretKey: appConf.Jwt.AccessSecret,
				})
				claims, err := jwtInstance.ParseToken(tokenString)
				if err != nil {
					return nil, authpb.ErrorUnauthorized("invalid token: %v", err)
				}

				// 验证用户角色
				var userRole consts.UserRole
				switch claims.Role {
				case "guest":
					userRole = consts.Guest
				case "user":
					userRole = consts.User
				case "admin":
					userRole = consts.Admin
				case "operator":
					userRole = consts.Operator
				default:
					return nil, authpb.ErrorUnauthorized("unknown role")
				}

				if userRole < minRole {
					return nil, authpb.ErrorUnauthorized("permission denied, you at least need %s role", minRole.String())
				}

				// 将用户claims存入context
				ctx = jwt.NewContext(ctx, claims)

				return handler(ctx, req)
			}
		}
	}
}
