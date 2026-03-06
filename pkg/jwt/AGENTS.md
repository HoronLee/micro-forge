<!-- Parent: ../AGENTS.md -->
# JWT 认证工具 (pkg/jwt)

**最后更新时间**: 2026-03-06

## 模块目的

提供泛型 JWT 工具：签发、解析，以及 claims 的 context 注入/提取。

## 当前实现事实

- 核心类型：`JWT[T any]`
- 配置类型：`Config{ SecretKey string }`
- `GenerateToken` / `ParseToken` 都会在运行时校验 `*T` 是否实现 `jwt.Claims`
- `NewContext` / `FromContext` 使用包内私有 key，避免 context key 冲突

## 关键文件

- `jwt.go`

## 使用示例

```go
type MyClaims struct {
    UserID int64 `json:"user_id"`
    jwt.RegisteredClaims
}

j := jwtutil.NewJWT[MyClaims](&jwtutil.Config{SecretKey: "secret"})
token, err := j.GenerateToken(&MyClaims{UserID: 1})
claims, err := j.ParseToken(token)
ctx := jwtutil.NewContext(context.Background(), claims)
```

## 注意事项

- claims 类型通常通过嵌入 `jwt.RegisteredClaims` 满足接口
- 当前目录没有独立测试文件，修改逻辑后应至少运行 `go test ./pkg/...`
