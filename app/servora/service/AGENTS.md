# AGENTS.md - app/servora/service/ 主服务实现

<!-- Parent: ../../AGENTS.md -->
<!-- Generated: 2026-02-09 | Updated: 2026-02-26 -->

## 目录概述

`app/servora/service/` 是 servora 项目的主微服务实现，采用模块化单体架构（Modular Monolith），在一个服务进程内组织多个业务域（auth、user、test）。这是一个生产级的微服务实现，展示了完整的 DDD 分层架构、Wire 依赖注入、前后端分离等最佳实践。

**核心特点**：
- 多业务模块单体：在单一进程内组织多个领域模块，降低分布式复杂度
- 完整的 DDD 分层：严格遵循业务逻辑层（biz）、数据访问层（data）、服务层（service）三层架构
- 前后端分离：前端应用位于仓库根目录 `web/`
- 双协议支持：同时提供 gRPC 和 HTTP 接口，自动生成 OpenAPI 文档
- 生产级配置：完整的 Kubernetes、Docker、配置中心等部署方案

## 目录结构

```
app/servora/service/
├── cmd/                    # 服务入口和工具
│   ├── server/            # 主服务启动入口
│   │   ├── main.go        # 主函数（启动 HTTP/gRPC 服务器）
│   │   ├── config.go      # 配置加载逻辑
│   │   ├── wire.go        # Wire 依赖注入配置（手动编辑）
│   │   └── wire_gen.go    # Wire 生成代码（自动生成，不要编辑）
├── internal/              # 内部实现（DDD 分层架构）
│   ├── biz/              # 业务逻辑层（UseCase 层）
│   │   ├── biz.go        # ProviderSet 定义
│   │   ├── entity.go     # 领域实体定义
│   │   ├── auth.go       # 认证业务逻辑
│   │   ├── user.go       # 用户业务逻辑
│   │   ├── test.go       # 测试业务逻辑
│   │   └── README.md     # Biz 层说明
│   ├── data/             # 数据访问层（Repository 层）
│   │   ├── data.go       # Data 初始化 + ProviderSet
│   │   ├── discovery.go  # 服务发现客户端
│   │   ├── auth.go       # Auth Repository 实现
│   │   ├── user.go       # User Repository 实现
│   │   ├── test.go       # Test Repository 实现
│   │   ├── schema/       # Ent Schema 定义
│   │   ├── ent/          # Ent 生成代码
│   │   ├── gorm/po/      # GORM GEN 生成的持久化对象（PO）
│   │   │   └── *.gen.go  # 自动生成的数据模型
│   │   ├── gorm/dao/     # GORM GEN 生成的 DAO（并行保留）
│   │   │   ├── gen.go    # DAO 生成配置
│   │   │   └── *.gen.go  # 自动生成的查询接口
│   │   └── README.md     # Data 层说明
│   ├── service/          # 服务层（API 接口层）
│   │   ├── service.go    # ProviderSet 定义
│   │   ├── auth.go       # Auth gRPC 服务实现
│   │   ├── user.go       # User gRPC 服务实现
│   │   ├── test.go       # Test gRPC 服务实现
│   │   └── README.md     # Service 层说明
│   ├── server/           # 服务器配置
│   │   ├── server.go     # ProviderSet + 服务器工厂
│   │   ├── grpc.go       # gRPC 服务器配置（端口、中间件、注册）
│   │   ├── http.go       # HTTP 服务器配置（端口、路由、CORS）
│   │   ├── registry.go   # 服务注册中心配置（Consul/Nacos/etcd）
│   │   ├── metrics.go    # Prometheus 指标采集
│   │   └── middleware/   # 中间件
│   │       ├── middleware.go  # 中间件集合
│   │       └── AuthJWT.go     # JWT 认证中间件
│   └── consts/           # 常量定义
│       └── user.go       # 用户相关常量
├── configs/             # 配置文件
│   └── config.yaml      # 服务配置（数据库、Redis、注册中心等）
├── deployment/          # 部署配置
│   ├── docker/         # Docker 部署资源（configs/logs）
│   └── kubernetes/     # Kubernetes 清单
│       ├── deployment.yaml
│       ├── service.yaml
│       └── configmap.yaml
├── manifests/          # K8s 项目级资源（SQL 初始化脚本等）
│   └── init.sql
├── bin/                # 编译输出目录
│   └── server          # 编译后的服务二进制
├── logs/               # 日志输出目录
├── Dockerfile          # Docker 镜像构建文件
├── openapi.yaml        # 生成的 OpenAPI 文档（自动生成）
├── Makefile            # 服务构建文件（include ../../../app.mk）
└── README.md           # 服务说明文档
```

## DDD 分层架构详解

servora 采用经典的 DDD（领域驱动设计）三层架构，层间依赖单向、职责清晰。

### 架构图

```
┌─────────────────────────────────────────────────────────┐
│  客户端层（Client）                                      │
│  - gRPC 客户端                                           │
│  - HTTP 客户端（浏览器、移动端）                          │
│  - 其他微服务                                            │
└─────────────────┬───────────────────────────────────────┘
                  │ 调用
┌─────────────────▼───────────────────────────────────────┐
│  服务层（internal/service/）                             │
│  - 实现 proto 定义的 gRPC/HTTP 接口                      │
│  - 参数验证和格式转换（DTO ↔ proto）                     │
│  - 调用业务逻辑层（不包含业务逻辑）                       │
│  - 错误处理和响应封装                                     │
└─────────────────┬───────────────────────────────────────┘
                  │ 依赖（通过接口）
┌─────────────────▼───────────────────────────────────────┐
│  业务逻辑层（internal/biz/）                             │
│  - UseCase 实现（核心业务逻辑）                          │
│  - 定义领域模型（Entity、Value Object）                  │
│  - 定义 Repository 接口（依赖倒置）                      │
│  - 编排数据访问和外部服务调用                             │
│  - 业务规则验证                                          │
└─────────────────┬───────────────────────────────────────┘
                  │ 依赖（通过接口）
┌─────────────────▼───────────────────────────────────────┐
│  数据访问层（internal/data/）                            │
│  - 实现 biz 层定义的 Repository 接口                     │
│  - 数据库访问（Ent 默认 + GORM GEN 并行保留）             │
│  - Redis 缓存操作                                        │
│  - 外部服务客户端（gRPC 服务间调用）                      │
│  - PO（持久化对象）↔ DO（领域对象）转换                  │
└─────────────────┬───────────────────────────────────────┘
                  │ 访问
┌─────────────────▼───────────────────────────────────────┐
│  基础设施层（Infrastructure）                            │
│  - MySQL / PostgreSQL / SQLite（数据库）                │
│  - Redis（缓存）                                         │
│  - 外部微服务（gRPC）                                    │
│  - 服务注册中心（Consul / Nacos / etcd）                 │
└─────────────────────────────────────────────────────────┘
```

### 1. 服务层（internal/service/）

**职责**：实现 proto 定义的 gRPC/HTTP 接口，作为外部世界与业务逻辑的适配层。

**特征**：
- 一个 proto 服务对应一个 service 文件（如 `auth.proto` → `auth.go`）
- 薄适配层，不包含业务逻辑
- 参数验证（验证请求参数的合法性）
- DTO 转换（proto 消息 ↔ 业务实体）
- 错误处理（将业务错误转换为 gRPC 错误码）

**代码示例**：
```go
// internal/service/auth.go
package service

import (
    "context"

    authv1 "github.com/horonlee/servora/api/gen/go/auth/service/v1"
    "github.com/horonlee/servora/app/servora/service/internal/biz"
)

type AuthService struct {
    authv1.UnimplementedAuthServer  // 嵌入未实现的服务器（前向兼容）
    uc *biz.AuthUsecase              // 依赖业务逻辑层
}

func NewAuthService(uc *biz.AuthUsecase) *AuthService {
    return &AuthService{uc: uc}
}

// Login 实现登录接口
func (s *AuthService) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginReply, error) {
    // 1. 参数验证
    if req.Username == "" || req.Password == "" {
        return nil, authv1.ErrorInvalidArgument("username and password required")
    }

    // 2. 调用业务逻辑层
    token, err := s.uc.Login(ctx, req.Username, req.Password)
    if err != nil {
        return nil, err  // 业务错误直接返回
    }

    // 3. 构造响应
    return &authv1.LoginReply{Token: token}, nil
}
```

**ProviderSet 定义**（`internal/service/service.go`）：
```go
package service

import "github.com/google/wire"

// ProviderSet is service providers.
var ProviderSet = wire.NewSet(
    NewAuthService,
    NewUserService,
    NewTestService,
)
```

### 2. 业务逻辑层（internal/biz/）

**职责**：实现核心业务逻辑（UseCase），包含领域模型和业务规则。

**特征**：
- 定义领域模型（Entity、Value Object）
- 定义 Repository 接口（依赖倒置原则）
- 实现业务规则和用例流程
- 编排多个 Repository 和外部服务
- 与具体技术实现无关（不依赖数据库、框架等）

**代码示例**：
```go
// internal/biz/auth.go
package biz

import (
    "context"
    "errors"

    "github.com/go-kratos/kratos/v2/log"
    "github.com/horonlee/servora/pkg/hash"
    "github.com/horonlee/servora/pkg/jwt"
)

// User 领域模型（Domain Object）
type User struct {
    ID       uint64
    Username string
    Password string  // 哈希后的密码
    Email    string
}

// AuthRepo Repository 接口（由 data 层实现）
type AuthRepo interface {
    GetUserByUsername(ctx context.Context, username string) (*User, error)
    CreateUser(ctx context.Context, user *User) error
}

// AuthUsecase 认证业务逻辑
type AuthUsecase struct {
    repo AuthRepo       // Repository 接口
    jwt  *jwt.Manager   // JWT 管理器
    log  *log.Helper
}

func NewAuthUsecase(repo AuthRepo, jwt *jwt.Manager, logger log.Logger) *AuthUsecase {
    return &AuthUsecase{
        repo: repo,
        jwt:  jwt,
        log:  log.NewHelper(logger),
    }
}

// Login 登录业务逻辑
func (uc *AuthUsecase) Login(ctx context.Context, username, password string) (string, error) {
    // 1. 查询用户（通过 Repository 接口）
    user, err := uc.repo.GetUserByUsername(ctx, username)
    if err != nil {
        return "", errors.New("user not found")
    }

    // 2. 验证密码（业务规则）
    if !hash.VerifyPassword(password, user.Password) {
        return "", errors.New("invalid password")
    }

    // 3. 生成 JWT Token
    token, err := uc.jwt.GenerateToken(user.ID, user.Username)
    if err != nil {
        return "", errors.New("failed to generate token")
    }

    uc.log.Infof("user %s logged in successfully", username)
    return token, nil
}

// Register 注册业务逻辑
func (uc *AuthUsecase) Register(ctx context.Context, username, password string) error {
    // 1. 检查用户是否存在（业务规则）
    if _, err := uc.repo.GetUserByUsername(ctx, username); err == nil {
        return errors.New("user already exists")
    }

    // 2. 密码哈希（业务规则）
    hashedPassword, err := hash.HashPassword(password)
    if err != nil {
        return err
    }

    // 3. 创建用户
    user := &User{
        Username: username,
        Password: hashedPassword,
    }
    return uc.repo.CreateUser(ctx, user)
}
```

**ProviderSet 定义**（`internal/biz/biz.go`）：
```go
package biz

import "github.com/google/wire"

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
    NewAuthUsecase,
    NewUserUsecase,
    NewTestUsecase,
)
```

### 3. 数据访问层（internal/data/）

**职责**：实现 Repository 接口，处理数据持久化和外部服务调用。

**特征**：
- 实现 biz 层定义的 Repository 接口
- 数据库访问（默认使用 Ent，保留 GORM GEN 工具链）
- 缓存访问（使用 Redis）
- 外部服务调用（gRPC 客户端）
- PO（持久化对象）与 DO（领域对象）转换

**代码示例**：
```go
// internal/data/auth.go
package data

import (
    "context"

    "github.com/go-kratos/kratos/v2/log"
    "github.com/horonlee/servora/app/servora/service/internal/biz"
    "github.com/horonlee/servora/app/servora/service/internal/data/gorm/po"
    "gorm.io/gorm"
)

// authRepo Repository 实现
type authRepo struct {
    data *Data
    log  *log.Helper
}

// NewAuthRepo 创建 Auth Repository（返回接口类型）
func NewAuthRepo(data *Data, logger log.Logger) biz.AuthRepo {
    return &authRepo{
        data: data,
        log:  log.NewHelper(logger),
    }
}

// GetUserByUsername 通过用户名查询用户
func (r *authRepo) GetUserByUsername(ctx context.Context, username string) (*biz.User, error) {
    // 1. 使用 GORM GEN 生成的 DAO 查询数据库
    userPO, err := r.data.query.User.
        WithContext(ctx).
        Where(r.data.query.User.Username.Eq(username)).
        First()

    if err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, errors.New("user not found")
        }
        r.log.Errorf("query user by username failed: %v", err)
        return nil, err
    }

    // 2. PO（持久化对象）→ DO（领域对象）转换
    return &biz.User{
        ID:       userPO.ID,
        Username: userPO.Username,
        Password: userPO.Password,
        Email:    userPO.Email,
    }, nil
}

// CreateUser 创建用户
func (r *authRepo) CreateUser(ctx context.Context, user *biz.User) error {
    // 1. DO → PO 转换
    userPO := &po.User{
        Username: user.Username,
        Password: user.Password,
        Email:    user.Email,
    }

    // 2. 使用 GORM GEN DAO 插入数据库
    if err := r.data.query.User.WithContext(ctx).Create(userPO); err != nil {
        r.log.Errorf("create user failed: %v", err)
        return err
    }

    // 3. 回填生成的 ID
    user.ID = userPO.ID
    return nil
}
```

**数据层初始化**（`internal/data/data.go`）：
```go
package data

import (
    "github.com/google/wire"
    "gorm.io/gorm"

    dao "github.com/horonlee/servora/app/servora/service/internal/data/gorm/dao"
    "github.com/horonlee/servora/pkg/redis"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
    NewDiscovery,  // 服务发现
    NewDB,         // 数据库连接
    NewRedis,      // Redis 连接
    NewData,       // Data 初始化
    NewAuthRepo,   // Auth Repository
    NewUserRepo,   // User Repository
    NewTestRepo,   // Test Repository
)

// Data 包含所有基础设施依赖
type Data struct {
    query  *dao.Query       // GORM GEN 生成的查询接口
    log    *log.Helper
    client client.Client    // gRPC 客户端（服务间调用）
    redis  *redis.Client    // Redis 客户端
}

// NewData 初始化 Data
func NewData(db *gorm.DB, c *conf.Data, logger log.Logger,
             client client.Client, redisClient *redis.Client) (*Data, func(), error) {

    // 初始化 GORM GEN DAO
    dao.SetDefault(db)

    d := &Data{
        query:  dao.Q,  // 全局查询对象
        log:    log.NewHelper(logger),
        client: client,
        redis:  redisClient,
    }

    // 清理函数（关闭资源）
    cleanup := func() {
        log.NewHelper(logger).Info("closing data resources")
    }

    return d, cleanup, nil
}

// NewDB 创建数据库连接
func NewDB(cfg *conf.Data, logger log.Logger) (*gorm.DB, error) {
    // 支持 MySQL、PostgreSQL、SQLite
    var dialector gorm.Dialector
    switch strings.ToLower(cfg.Database.GetDriver()) {
    case "mysql":
        dialector = mysql.Open(cfg.Database.GetSource())
    case "sqlite":
        dialector = sqlite.Open(cfg.Database.GetSource())
    case "postgres", "postgresql":
        dialector = postgres.Open(cfg.Database.GetSource())
    default:
        return nil, errors.New("unsupported db driver")
    }

    // 连接数据库（带重试）
    db, err := gorm.Open(dialector, &gorm.Config{
        Logger: gormLogger,
    })
    return db, err
}

// NewRedis 创建 Redis 连接
func NewRedis(cfg *conf.Data, logger log.Logger) (*redis.Client, func(), error) {
    redisConfig := redis.NewConfigFromProto(cfg.Redis)
    return redis.NewClient(redisConfig, logger)
}
```

### 层间依赖规则

**依赖方向（单向）**：
```
service → biz → data → infrastructure
```

**核心原则**：
1. **依赖倒置**：biz 层定义接口，data 层实现接口（面向接口编程）
2. **禁止反向依赖**：data 层不能依赖 biz 层的具体类型（只能依赖接口）
3. **禁止跨层调用**：service 层不能直接调用 data 层
4. **接口隔离**：每个 UseCase 只依赖需要的 Repository 接口

**示例**：
```go
// ✅ 正确：biz 层定义接口，data 层实现
// internal/biz/auth.go
type AuthRepo interface {
    GetUserByUsername(ctx context.Context, username string) (*User, error)
}

// internal/data/auth.go
type authRepo struct { /* ... */ }
func NewAuthRepo(data *Data, logger log.Logger) biz.AuthRepo {
    return &authRepo{data: data}
}

// ❌ 错误：biz 层依赖 data 层的具体类型
// internal/biz/auth.go
import "github.com/horonlee/servora/app/servora/service/internal/data"
type AuthUsecase struct {
    repo *data.AuthRepo  // 错误！依赖了具体类型
}
```

## Wire 依赖注入工作流

Wire 是 Google 开源的编译时依赖注入工具，通过代码生成实现 DI，避免运行时反射。

### Wire 核心概念

**Provider（提供者）**：返回依赖实例的构造函数
```go
// NewAuthUsecase 是一个 Provider
func NewAuthUsecase(repo AuthRepo, jwt *jwt.Manager, logger log.Logger) *AuthUsecase {
    return &AuthUsecase{repo: repo, jwt: jwt, log: log.NewHelper(logger)}
}
```

**ProviderSet（提供者集合）**：一组相关 Provider 的集合
```go
// internal/biz/biz.go
var ProviderSet = wire.NewSet(
    NewAuthUsecase,
    NewUserUsecase,
    NewTestUsecase,
)
```

### Wire 配置文件

**标准模板**（`cmd/server/wire.go`）：
```go
//go:build wireinject
// +build wireinject

package main

import (
    "github.com/google/wire"
    "github.com/go-kratos/kratos/v2"
    "github.com/go-kratos/kratos/v2/log"

    "github.com/horonlee/servora/api/gen/go/conf/v1"
    "github.com/horonlee/servora/app/servora/service/internal/biz"
    "github.com/horonlee/servora/app/servora/service/internal/data"
    "github.com/horonlee/servora/app/servora/service/internal/server"
    "github.com/horonlee/servora/app/servora/service/internal/service"
    "github.com/horonlee/servora/pkg/transport/client"
)

// wireApp 构建应用依赖图
func wireApp(
    *conf.Server,     // 服务器配置
    *conf.Discovery,  // 服务发现配置
    *conf.Registry,   // 注册中心配置
    *conf.Data,       // 数据源配置
    *conf.App,        // 应用配置
    *conf.Trace,      // 链路追踪配置
    *conf.Metrics,    // 指标配置
    log.Logger,       // 日志实例
) (*kratos.App, func(), error) {
    panic(wire.Build(
        server.ProviderSet,   // 服务器层（gRPC/HTTP Server）
        service.ProviderSet,  // 服务层（API 实现）
        biz.ProviderSet,      // 业务逻辑层（UseCase）
        data.ProviderSet,     // 数据访问层（Repository）
        client.ProviderSet,   // gRPC 客户端
        newApp,               // 应用构造函数
    ))
}
```

**关键点**：
1. `//go:build wireinject` - 构建标签（只在生成时使用）
2. `panic(wire.Build(...))` - Wire 指令（会被替换为实际代码）
3. 参数是依赖的输入，返回值是依赖的输出
4. Wire 会自动解析依赖关系并生成代码

### 生成代码示例

运行 `make wire` 后，Wire 会生成 `wire_gen.go`：

```go
// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject

package main

func wireApp(
    confServer *conf.Server,
    confDiscovery *conf.Discovery,
    confRegistry *conf.Registry,
    confData *conf.Data,
    confApp *conf.App,
    confTrace *conf.Trace,
    confMetrics *conf.Metrics,
    logger log.Logger,
) (*kratos.App, func(), error) {
    // Wire 按依赖顺序创建实例

    // 1. 数据层
    db, err := data.NewDB(confData, logger)
    if err != nil {
        return nil, nil, err
    }
    redisClient, cleanup1, err := data.NewRedis(confData, logger)
    if err != nil {
        return nil, nil, err
    }
    grpcClient := client.NewClient(confDiscovery, logger)
    dataData, cleanup2, err := data.NewData(db, confData, logger, grpcClient, redisClient)
    if err != nil {
        cleanup1()
        return nil, nil, err
    }

    // 2. Repository
    authRepo := data.NewAuthRepo(dataData, logger)
    userRepo := data.NewUserRepo(dataData, logger)

    // 3. 业务逻辑层
    jwtManager := jwt.NewManager(confApp.Jwt.Secret, confApp.Jwt.Expiration)
    authUsecase := biz.NewAuthUsecase(authRepo, jwtManager, logger)
    userUsecase := biz.NewUserUsecase(userRepo, logger)

    // 4. 服务层
    authService := service.NewAuthService(authUsecase)
    userService := service.NewUserService(userUsecase)

    // 5. 服务器
    grpcServer := server.NewGRPCServer(confServer, authService, userService, logger)
    httpServer := server.NewHTTPServer(confServer, authService, userService, logger)
    registrar := server.NewRegistrar(confRegistry)

    // 6. 应用
    app := newApp(logger, registrar, grpcServer, httpServer)

    // 组合所有 cleanup 函数
    return app, func() {
        cleanup2()
        cleanup1()
    }, nil
}
```

### 常见使用场景

**场景 1：添加新 UseCase**
```bash
# 1. 创建 UseCase 构造函数（internal/biz/product.go）
func NewProductUsecase(repo ProductRepo, logger log.Logger) *ProductUsecase {
    return &ProductUsecase{repo: repo, log: log.NewHelper(logger)}
}

# 2. 添加到 ProviderSet（internal/biz/biz.go）
var ProviderSet = wire.NewSet(
    NewAuthUsecase,
    NewUserUsecase,
    NewProductUsecase,  // 新增
)

# 3. 运行 make wire 重新生成代码
cd /Users/horonlee/projects/go/servora/app/servora/service
make wire
```

**场景 2：添加新 Repository**
```bash
# 1. 定义接口（internal/biz/product.go）
type ProductRepo interface {
    CreateProduct(ctx context.Context, p *Product) error
}

# 2. 实现 Repository（internal/data/product.go）
func NewProductRepo(data *Data, logger log.Logger) biz.ProductRepo {
    return &productRepo{data: data, log: log.NewHelper(logger)}
}

# 3. 添加到 ProviderSet（internal/data/data.go）
var ProviderSet = wire.NewSet(
    NewData,
    NewAuthRepo,
    NewProductRepo,  // 新增
)

# 4. 运行 make wire
make wire
```

**场景 3：注入外部依赖**
```bash
# 1. 创建外部依赖的 Provider（如 JWT Manager）
func NewJWTManager(c *conf.App) (*jwt.Manager, error) {
    return jwt.NewManager(c.Jwt.Secret, c.Jwt.Expiration)
}

# 2. 在 wire.go 中添加到 Build 列表
panic(wire.Build(
    server.ProviderSet,
    service.ProviderSet,
    biz.ProviderSet,
    data.ProviderSet,
    NewJWTManager,  // 新增
    newApp,
))

# 3. 在 UseCase 中使用
func NewAuthUsecase(repo AuthRepo, jwt *jwt.Manager, logger log.Logger) *AuthUsecase {
    return &AuthUsecase{repo: repo, jwt: jwt}
}
```

### Wire 完整工作流程

```bash
# 1. 编辑 Wire 配置文件
vim cmd/server/wire.go

# 2. 运行 Wire 代码生成
cd /Users/horonlee/projects/go/servora/app/servora/service
make wire

# 3. 查看生成的代码（验证依赖关系）
cat cmd/server/wire_gen.go

# 4. 编译并运行服务
make run
```

## 前端项目（web/）

### 项目概述

仓库根目录 `web/` 包含一个独立的 Vue 3 + Vite 前端应用，采用现代化的前端技术栈。

**技术栈**：
- **框架**：Vue 3（Composition API）
- **构建工具**：Vite
- **语言**：TypeScript
- **状态管理**：Pinia
- **路由**：Vue Router
- **测试**：Vitest（单元测试）、Playwright（E2E 测试）
- **代码风格**：ESLint + Prettier

### 目录结构

```
web/
├── src/
│   ├── components/       # Vue 组件
│   ├── views/           # 页面组件
│   ├── router/          # Vue Router 路由配置
│   ├── stores/          # Pinia 状态管理
│   ├── api/             # API 客户端封装
│   ├── assets/          # 静态资源（图片、样式）
│   ├── __tests__/       # Vitest 单元测试
│   ├── App.vue          # 根组件
│   └── main.ts          # 应用入口
├── e2e/                 # Playwright E2E 测试
│   └── example.spec.ts
├── public/              # 公共静态资源
├── vite.config.ts       # Vite 配置
├── playwright.config.ts # Playwright 配置
├── tsconfig.json        # TypeScript 配置
├── package.json         # 依赖配置
└── README.md            # 前端文档
```

### 常用命令

```bash
cd /Users/horonlee/projects/go/servora/web

# 安装依赖
bun install

# 开发服务器（热重载）
bun dev

# 构建生产版本
bun build

# 单元测试（Vitest）
bun test:unit
bun test:unit src/__tests__/component.spec.ts  # 运行单个测试

# E2E 测试（Playwright）
npx playwright install  # 首次安装浏览器
bun test:e2e
bun test:e2e e2e/login.spec.ts --project=chromium  # 运行单个测试

# 代码检查
bun lint
bun format

# 预览生产构建
bun preview
```

### TypeScript 规范

**组件示例**（使用 `<script setup lang="ts">`）：
```vue
<!-- src/components/UserProfile.vue -->
<script setup lang="ts">
import { ref, computed } from 'vue'

// 定义接口
interface User {
  id: number
  username: string
  email: string
}

// Props 类型
interface Props {
  userId: number
}

const props = defineProps<Props>()

// 状态
const user = ref<User | null>(null)
const loading = ref(false)

// 计算属性
const displayName = computed(() => {
  return user.value?.username || 'Guest'
})

// 方法（必须类型化）
async function fetchUser(): Promise<void> {
  loading.value = true
  try {
    const response = await fetch(`/api/users/${props.userId}`)
    user.value = await response.json() as User
  } catch (error) {
    console.error('Failed to fetch user:', error)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="user-profile">
    <h2>{{ displayName }}</h2>
  </div>
</template>
```

**禁止使用的模式**：
```typescript
// ❌ 禁止使用 as any
const data = response as any

// ❌ 禁止使用 @ts-ignore
// @ts-ignore
const value = obj.unknownProperty

// ✅ 正确：使用类型断言或类型守卫
const data = response as User
if ('username' in obj) {
  const value = obj.username
}
```

### API 客户端示例

```typescript
// src/api/auth.ts
import type { LoginRequest, LoginResponse } from './types'

export const authApi = {
  async login(req: LoginRequest): Promise<LoginResponse> {
    const response = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(req),
    })

    if (!response.ok) {
      throw new Error(`Login failed: ${response.statusText}`)
    }

    return response.json() as Promise<LoginResponse>
  },
}
```

## GORM GEN 使用指南

GORM GEN 是 GORM 的代码生成器，自动生成类型安全的 DAO 和 PO。

### 配置和生成

**生成方式**：
```bash
# 在项目根目录运行
svr gen gorm servora

# 或在服务目录下
make gen.gorm
```

> `make gen.gorm` 内部实际调用 `svr gen gorm servora` 来执行代码生成。
>
> 更多用法：
> ```bash
> # 预览将要生成的路径，不实际执行生成
> svr gen gorm servora --dry-run
>
> # 交互式选择服务（无参数时进入交互模式）
> svr gen gorm
> ```

### 使用生成的 DAO

```go
// internal/data/user.go
func (r *userRepo) GetByID(ctx context.Context, id uint64) (*biz.User, error) {
    // 使用类型安全的查询
    u, err := r.data.query.User.
        WithContext(ctx).
        Where(r.data.query.User.ID.Eq(id)).
        First()

    if err != nil {
        return nil, err
    }

    return &biz.User{
        ID:       u.ID,
        Username: u.Username,
    }, nil
}
```

## AI Agent 工作指南

### 添加新业务模块

**完整流程**（以添加 `product` 模块为例）：

1. **定义 Proto API**
```bash
# 创建 proto 文件
mkdir -p /Users/horonlee/projects/go/servora/api/protos/product/service/v1

# 编写 product.proto（定义 gRPC 服务）
```

2. **生成代码**
```bash
cd /Users/horonlee/projects/go/servora
make gen
```

3. **实现数据层**（`internal/data/product.go`）
```go
package data

type productRepo struct {
    data *Data
    log  *log.Helper
}

func NewProductRepo(data *Data, logger log.Logger) biz.ProductRepo {
    return &productRepo{data: data, log: log.NewHelper(logger)}
}
```

4. **实现业务层**（`internal/biz/product.go`）
```go
package biz

type ProductRepo interface {
    CreateProduct(ctx context.Context, p *Product) error
}

type ProductUsecase struct {
    repo ProductRepo
    log  *log.Helper
}

func NewProductUsecase(repo ProductRepo, logger log.Logger) *ProductUsecase {
    return &ProductUsecase{repo: repo, log: log.NewHelper(logger)}
}
```

5. **实现服务层**（`internal/service/product.go`）
```go
package service

type ProductService struct {
    productv1.UnimplementedProductServer
    uc *biz.ProductUsecase
}

func NewProductService(uc *biz.ProductUsecase) *ProductService {
    return &ProductService{uc: uc}
}
```

6. **更新 ProviderSet**
```go
// internal/data/data.go
var ProviderSet = wire.NewSet(
    NewData,
    NewAuthRepo,
    NewProductRepo,  // 新增
)

// internal/biz/biz.go
var ProviderSet = wire.NewSet(
    NewAuthUsecase,
    NewProductUsecase,  // 新增
)

// internal/service/service.go
var ProviderSet = wire.NewSet(
    NewAuthService,
    NewProductService,  // 新增
)
```

7. **注册到 gRPC 服务器**（`internal/server/grpc.go`）
```go
func NewGRPCServer(
    c *conf.Server,
    authSvc *service.AuthService,
    productSvc *service.ProductService,  // 新增参数
    logger log.Logger,
) *grpc.Server {
    // ...
    authv1.RegisterAuthServer(srv, authSvc)
    productv1.RegisterProductServer(srv, productSvc)  // 注册服务
    return srv
}
```

8. **重新生成 Wire 代码**
```bash
cd /Users/horonlee/projects/go/servora/app/servora/service
make wire
```

9. **运行和测试**
```bash
make run
```

### 开发工作流速查

```bash
# 1. 修改 proto 文件
vim /Users/horonlee/projects/go/servora/api/protos/auth/service/v1/auth.proto

# 2. 生成代码（在项目根目录）
cd /Users/horonlee/projects/go/servora
make gen

# 3. 实现业务逻辑（biz → data → service）

# 4. 重新生成 Wire（在服务目录）
cd /Users/horonlee/projects/go/servora/app/servora/service
make wire

# 5. 运行服务
make run

# 6. 运行测试
make test
```

### 调试技巧

```bash
# 查看生成的 Wire 代码
cat /Users/horonlee/projects/go/servora/app/servora/service/cmd/server/wire_gen.go

# 检查 Wire 依赖图
cd /Users/horonlee/projects/go/servora/app/servora/service/cmd/server
wire show

# 验证 gRPC 接口
grpcurl -plaintext localhost:9000 list
grpcurl -plaintext localhost:9000 auth.service.v1.Auth/Login

# 查看 OpenAPI 文档
cat /Users/horonlee/projects/go/servora/app/servora/service/openapi.yaml

# 查看日志
tail -f /Users/horonlee/projects/go/servora/app/servora/service/logs/servora.service.log
```

## 测试开发

### 单元测试模板

```go
// internal/biz/auth_test.go
package biz

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

// Mock Repository
type mockAuthRepo struct {
    mock.Mock
}

func (m *mockAuthRepo) GetUserByUsername(ctx context.Context, username string) (*User, error) {
    args := m.Called(ctx, username)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*User), args.Error(1)
}

func TestAuthUsecase_Login(t *testing.T) {
    tests := []struct {
        name     string
        username string
        password string
        wantErr  bool
    }{
        {"valid credentials", "admin", "password123", false},
        {"invalid password", "admin", "wrong", true},
        {"user not found", "unknown", "password", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            repo := new(mockAuthRepo)
            uc := NewAuthUsecase(repo, nil, nil)

            // Setup mock expectations
            if tt.username == "admin" {
                repo.On("GetUserByUsername", mock.Anything, tt.username).
                    Return(&User{Username: "admin", Password: "hashed"}, nil)
            } else {
                repo.On("GetUserByUsername", mock.Anything, tt.username).
                    Return(nil, errors.New("not found"))
            }

            _, err := uc.Login(context.Background(), tt.username, tt.password)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### 集成测试（跳过外部依赖）

```go
func TestUserRepo_Create(t *testing.T) {
    db, err := gorm.Open(mysql.Open("test:test@tcp(localhost:3306)/test"))
    if err != nil {
        t.Skipf("database not available: %v", err)
        return
    }

    repo := NewUserRepo(&Data{query: dao.Q}, nil)
    // 运行测试...
}
```

## 常见任务速查

### 启动服务

```bash
cd /Users/horonlee/projects/go/servora/app/servora/service

# 配置文件（首次）
cp ../../../api/protos/conf/v1/config-example.yaml configs/config.yaml
# 编辑 configs/config.yaml（配置数据库和 Redis）

# 运行服务
make run
```

### 前端开发

```bash
cd /Users/horonlee/projects/go/servora/web

bun install
bun dev
```

### 部署

```bash
# Compose 构建（仓库根目录）
cd /Users/horonlee/projects/go/servora
make compose.build

# Kubernetes 部署
kubectl apply -f deployment/kubernetes/
```

## 注意事项

### 代码生成
- 修改 `.proto` 文件后必须运行根目录的 `make gen`
- 修改 `wire.go` 后必须运行服务目录的 `make wire`
- 生成的代码（`wire_gen.go`, `ent/`, `gorm/dao/*.gen.go`, `gorm/po/*.gen.go`）不要手动编辑

### 依赖注入
- 每个构造函数应返回接口类型（而非具体类型）
- 使用 `wire.Bind()` 显式绑定接口到实现
- 避免循环依赖（Wire 会在编译时检测）

### 性能和可靠性
- 数据库连接池配置（`max_open_conns`, `max_idle_conns`）
- Redis 连接超时和重试策略
- gRPC 超时和截止时间（context deadline）

### 安全性
- 永远不要在日志中打印敏感信息（密码、token）
- 使用参数化查询防止 SQL 注入
- JWT secret 必须从配置文件读取

## 依赖关系

**上游依赖**（本目录依赖的其他目录）：
- `/Users/horonlee/projects/go/servora/api/gen/go/` - 生成的 protobuf Go 代码
- `/Users/horonlee/projects/go/servora/pkg/` - 共享库（jwt, redis, logger 等）
- `/Users/horonlee/projects/go/servora/api/protos/conf/v1/` - 配置文件定义

**下游依赖**（依赖本目录的其他目录）：
- `/Users/horonlee/projects/go/servora/deployment/` - 部署配置

**外部依赖**：
- Kratos v2 框架
- Ent + GORM GEN（双 ORM）
- Wire
- Redis
- 数据库驱动（MySQL/PostgreSQL/SQLite）
