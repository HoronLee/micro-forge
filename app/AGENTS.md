# AGENTS.md - app/ 微服务实现层

<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-09 | Updated: 2026-02-26 -->

## 目录概述

`app/` 是 servora 项目的微服务实现层，包含所有微服务的业务逻辑和运行时代码。每个微服务都采用 DDD（领域驱动设计）分层架构，通过 Wire 进行依赖注入，遵循清晰的职责分离原则。

**核心价值**：
- 统一的微服务架构模板（DDD 分层 + Wire DI）
- 独立部署单元（每个服务可单独构建、测试、部署）
- 清晰的领域边界（业务逻辑与基础设施分离）
- 可扩展的服务组织（通过 `app.mk` 共享构建逻辑）

## 子目录结构

### `servora/service/` - 主服务（单体微服务）
主业务服务，包含多个业务模块（auth, user, test 等），采用模块化单体架构。

**特点**：
- 多模块单体：在一个进程内组织多个业务域
- 同时支持 HTTP 和 gRPC 协议
- 生产级配置：包含 K8s manifests, Docker 配置等

**目录结构**：
```
servora/service/
├── cmd/server/           # 服务入口
│   ├── main.go          # 主函数（启动 HTTP/gRPC 服务器）
│   ├── wire.go          # Wire 依赖注入配置
│   └── wire_gen.go      # Wire 生成代码（自动生成）
├── internal/            # 内部实现（DDD 分层）
│   ├── biz/            # 业务逻辑层（UseCase）
│   │   ├── auth.go     # 认证业务逻辑
│   │   ├── user.go     # 用户业务逻辑
│   │   └── test.go     # 测试业务逻辑
│   ├── data/           # 数据访问层（Repository 实现）
│   │   ├── auth.go     # 认证数据访问
│   │   ├── user.go     # 用户数据访问
│   │   ├── data.go     # 数据层初始化（DB, Redis, Ent Client）
│   │   ├── schema/     # Ent Schema 定义
│   │   ├── ent/        # Ent 生成代码
│   │   └── gorm/po/    # GORM GEN 生成的持久化对象（并行保留）
│   ├── service/        # 服务层（API 接口实现）
│   │   ├── auth.go     # Auth gRPC 服务实现
│   │   ├── user.go     # User gRPC 服务实现
│   │   └── servora.go  # servora HTTP 接口实现
│   ├── server/         # 服务器配置
│   │   ├── grpc.go     # gRPC 服务器配置
│   │   └── http.go     # HTTP 服务器配置
│   └── consts/         # 常量定义
├── configs/            # 配置文件
│   └── config.yaml     # 服务配置（数据库、Redis、注册中心等）
├── deployment/         # 部署配置
│   ├── docker/         # Docker Compose
│   └── kubernetes/     # K8s 清单
├── manifests/          # K8s 资源（项目级）
├── bin/                # 编译输出
├── openapi.yaml        # 生成的 OpenAPI 文档
└── Makefile            # 服务构建文件（include ../../../app.mk）
```

**业务模块**：
- **auth** - 认证授权（登录、注册、JWT 验证）
- **user** - 用户管理（CRUD、个人资料）
- **test** - 测试接口（gRPC 调用示例）

### `sayhello/service/` - 独立微服务示例
最小化的独立微服务示例，展示如何构建轻量级服务。

**特点**：
- 独立部署单元（完全独立的服务进程）
- 简单的业务逻辑（Hello World 风格）
- 完整的服务结构（适合作为新服务模板）

**目录结构**：
```
sayhello/service/
├── cmd/server/           # 服务入口
│   ├── main.go
│   ├── wire.go
│   └── wire_gen.go
├── internal/
│   ├── server/          # 服务器配置
│   └── service/         # 服务实现
├── configs/
│   └── config.yaml
├── deployment/
├── bin/
└── Makefile
```

## DDD 分层架构详解

servora 采用经典的 DDD 分层架构，各层职责明确，依赖方向单向。

### 架构图

```
┌─────────────────────────────────────────────┐
│  服务层 (service/)                          │
│  - 实现 gRPC/HTTP 接口                      │
│  - 参数验证和转换                            │
│  - 调用业务逻辑层                            │
└─────────────────┬───────────────────────────┘
                  │ 依赖（通过接口）
┌─────────────────▼───────────────────────────┐
│  业务逻辑层 (biz/)                          │
│  - UseCase 实现（核心业务逻辑）             │
│  - 定义 Repository 接口                     │
│  - 编排数据访问和外部调用                    │
└─────────────────┬───────────────────────────┘
                  │ 依赖（通过接口）
┌─────────────────▼───────────────────────────┐
│  数据访问层 (data/)                         │
│  - Repository 接口实现                      │
│  - 数据库访问（Ent 为默认，GORM GEN 并行保留）│
│  - Redis 缓存                               │
│  - 外部服务客户端                            │
└─────────────────────────────────────────────┘
```

### 各层详细说明

#### 1. 服务层 (internal/service/)
**职责**：实现 proto 定义的 gRPC/HTTP 接口

**特征**：
- 一个 proto 服务对应一个 service 文件（如 `auth.proto` → `auth.go`）
- 参数验证（验证请求参数的合法性）
- DTO 转换（proto 消息 ↔ 业务实体）
- 不包含业务逻辑（只是薄的适配层）

**示例代码**：
```go
// internal/service/auth.go
type AuthService struct {
    authv1.UnimplementedAuthServer
    uc *biz.AuthUsecase  // 依赖业务逻辑层
}

func NewAuthService(uc *biz.AuthUsecase) *AuthService {
    return &AuthService{uc: uc}
}

func (s *AuthService) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginReply, error) {
    // 1. 参数验证
    if req.Username == "" || req.Password == "" {
        return nil, authv1.ErrorInvalidArgument("username and password required")
    }

    // 2. 调用业务逻辑层
    token, err := s.uc.Login(ctx, req.Username, req.Password)
    if err != nil {
        return nil, err
    }

    // 3. 返回响应
    return &authv1.LoginReply{Token: token}, nil
}
```

#### 2. 业务逻辑层 (internal/biz/)
**职责**：实现核心业务逻辑（UseCase）

**特征**：
- 定义领域模型（业务实体）
- 定义 Repository 接口（依赖倒置）
- 实现业务规则和用例流程
- 编排多个 Repository 和外部服务

**示例代码**：
```go
// internal/biz/auth.go
package biz

// 领域模型
type User struct {
    ID       uint64
    Username string
    Password string
}

// Repository 接口（由 data 层实现）
type AuthRepo interface {
    GetUserByUsername(ctx context.Context, username string) (*User, error)
    CreateUser(ctx context.Context, user *User) error
}

// UseCase
type AuthUsecase struct {
    repo AuthRepo
    jwt  *jwt.Manager
    log  *log.Helper
}

func NewAuthUsecase(repo AuthRepo, jwt *jwt.Manager, logger log.Logger) *AuthUsecase {
    return &AuthUsecase{
        repo: repo,
        jwt:  jwt,
        log:  log.NewHelper(logger),
    }
}

// 业务逻辑：登录
func (uc *AuthUsecase) Login(ctx context.Context, username, password string) (string, error) {
    // 1. 查询用户
    user, err := uc.repo.GetUserByUsername(ctx, username)
    if err != nil {
        return "", errors.Wrap(err, "user not found")
    }

    // 2. 验证密码（业务规则）
    if !hash.VerifyPassword(password, user.Password) {
        return "", errors.New("invalid password")
    }

    // 3. 生成 JWT
    token, err := uc.jwt.GenerateToken(user.ID, user.Username)
    if err != nil {
        return "", errors.Wrap(err, "failed to generate token")
    }

    return token, nil
}
```

#### 3. 数据访问层 (internal/data/)
**职责**：实现 Repository 接口，处理数据持久化

**特征**：
- 实现 biz 层定义的 Repository 接口
- 数据库访问（GORM）
- 缓存访问（Redis）
- 外部服务调用（gRPC 客户端）
- PO（持久化对象）与 DO（领域对象）转换

**示例代码**：
```go
// internal/data/auth.go
package data

import (
    "github.com/horonlee/servora/app/servora/service/internal/biz"
    "github.com/horonlee/servora/app/servora/service/internal/data/gorm/po"
)

// Repository 实现
type authRepo struct {
    data *Data  // 包含 DB, Redis 等
    log  *log.Helper
}

func NewAuthRepo(data *Data, logger log.Logger) biz.AuthRepo {
    return &authRepo{
        data: data,
        log:  log.NewHelper(logger),
    }
}

func (r *authRepo) GetUserByUsername(ctx context.Context, username string) (*biz.User, error) {
    // 1. 查询数据库（使用 GORM GEN 生成的 DAO）
    userPO, err := r.data.UserDAO.Where(r.data.UserDAO.Username.Eq(username)).First()
    if err != nil {
        return nil, err
    }

    // 2. PO → DO 转换
    return &biz.User{
        ID:       userPO.ID,
        Username: userPO.Username,
        Password: userPO.Password,
    }, nil
}

func (r *authRepo) CreateUser(ctx context.Context, user *biz.User) error {
    // 1. DO → PO 转换
    userPO := &po.User{
        Username: user.Username,
        Password: user.Password,
    }

    // 2. 插入数据库
    return r.data.UserDAO.Create(userPO)
}
```

**数据层初始化** (`internal/data/data.go`)：
```go
// internal/data/data.go
package data

import (
    "github.com/google/wire"
    "gorm.io/gorm"
    "github.com/horonlee/servora/pkg/redis"
)

// ProviderSet 是 data 层的 Wire Provider
var ProviderSet = wire.NewSet(
    NewData,
    NewAuthRepo,
    NewUserRepo,
)

// Data 包含所有基础设施依赖
type Data struct {
    DB       *gorm.DB
    Redis    *redis.Client
    UserDAO  *dao.User
    // ... 其他 DAO
}

func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
    // 初始化数据库、Redis 等
    db, err := gorm.Open(mysql.Open(c.Database.Source), &gorm.Config{})
    if err != nil {
        return nil, nil, err
    }

    rdb, err := redis.NewClient(c.Redis)
    if err != nil {
        return nil, nil, err
    }

    // 初始化 GORM GEN DAO
    dao.SetDefault(db)

    d := &Data{
        DB:      db,
        Redis:   rdb,
        UserDAO: dao.User,
    }

    cleanup := func() {
        log.Info("closing data resources")
        rdb.Close()
    }

    return d, cleanup, nil
}
```

### 层间依赖规则

**依赖方向**（单向）：
```
service → biz → data
```

**重要原则**：
1. **依赖倒置**：biz 层定义接口，data 层实现接口
2. **禁止反向依赖**：data 层不能依赖 biz 层的具体类型
3. **禁止跨层调用**：service 层不能直接调用 data 层
4. **接口隔离**：每个 UseCase 只依赖需要的 Repository 接口

**Wire 配置示例**：
```go
// cmd/server/wire.go
//go:build wireinject

package main

import (
    "github.com/google/wire"
    "github.com/horonlee/servora/app/servora/service/internal/biz"
    "github.com/horonlee/servora/app/servora/service/internal/data"
    "github.com/horonlee/servora/app/servora/service/internal/service"
    "github.com/horonlee/servora/app/servora/service/internal/server"
)

func wireApp(*conf.Server, *conf.Data, log.Logger) (*kratos.App, func(), error) {
    panic(wire.Build(
        server.ProviderSet,   // HTTP/gRPC 服务器
        service.ProviderSet,  // 服务层
        biz.ProviderSet,      // 业务逻辑层
        data.ProviderSet,     // 数据访问层
        newApp,
    ))
}
```

## Wire 依赖注入详解

### Wire 基础概念

**什么是 Wire**：
- Google 开源的编译时依赖注入工具
- 通过代码生成实现 DI（不使用反射，性能高）
- 在编译时检查依赖关系（类型安全）

**核心文件**：
- `wire.go` - Wire 配置文件（手动编写）
- `wire_gen.go` - Wire 生成的代码（自动生成，不要手动编辑）

### Provider 和 ProviderSet

**Provider**：返回依赖实例的构造函数
```go
// NewAuthUsecase 是一个 Provider
func NewAuthUsecase(repo AuthRepo, jwt *jwt.Manager, logger log.Logger) *AuthUsecase {
    return &AuthUsecase{repo: repo, jwt: jwt, log: log.NewHelper(logger)}
}
```

**ProviderSet**：一组相关 Provider 的集合
```go
// internal/biz/biz.go
package biz

var ProviderSet = wire.NewSet(
    NewAuthUsecase,
    NewUserUsecase,
    NewTestUsecase,
)
```

### Wire 配置文件

**标准模板** (`cmd/server/wire.go`)：
```go
//go:build wireinject
// +build wireinject

package main

import (
    "github.com/go-kratos/kratos/v2"
    "github.com/go-kratos/kratos/v2/log"
    "github.com/google/wire"

    "github.com/horonlee/servora/api/gen/go/conf/v1"
    "github.com/horonlee/servora/app/servora/service/internal/biz"
    "github.com/horonlee/servora/app/servora/service/internal/data"
    "github.com/horonlee/servora/app/servora/service/internal/server"
    "github.com/horonlee/servora/app/servora/service/internal/service"
)

// wireApp 构建应用依赖图
func wireApp(*conf.Server, *conf.Data, log.Logger) (*kratos.App, func(), error) {
    panic(wire.Build(
        server.ProviderSet,   // 服务器层
        service.ProviderSet,  // 服务层
        biz.ProviderSet,      // 业务逻辑层
        data.ProviderSet,     // 数据访问层
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

import (
    "github.com/go-kratos/kratos/v2"
    "github.com/go-kratos/kratos/v2/log"
    // ...
)

func wireApp(confServer *conf.Server, confData *conf.Data, logger log.Logger) (*kratos.App, func(), error) {
    // 按依赖顺序创建实例
    dataData, cleanup, err := data.NewData(confData, logger)
    if err != nil {
        return nil, nil, err
    }

    authRepo := data.NewAuthRepo(dataData, logger)
    authUsecase := biz.NewAuthUsecase(authRepo, jwtManager, logger)
    authService := service.NewAuthService(authUsecase)

    grpcServer := server.NewGRPCServer(confServer, authService, logger)
    httpServer := server.NewHTTPServer(confServer, authService, logger)

    app := newApp(logger, grpcServer, httpServer)

    return app, func() {
        cleanup()
    }, nil
}
```

### 常见使用场景

**场景 1：添加新 UseCase**
```go
// 1. 创建 UseCase 构造函数（internal/biz/newfeature.go）
func NewNewFeatureUsecase(repo NewFeatureRepo, logger log.Logger) *NewFeatureUsecase {
    return &NewFeatureUsecase{repo: repo, log: log.NewHelper(logger)}
}

// 2. 添加到 ProviderSet（internal/biz/biz.go）
var ProviderSet = wire.NewSet(
    NewAuthUsecase,
    NewUserUsecase,
    NewNewFeatureUsecase,  // 新增
)

// 3. 运行 make wire 重新生成代码
```

**场景 2：添加新 Repository**
```go
// 1. 创建 Repository 实现（internal/data/newfeature.go）
func NewNewFeatureRepo(data *Data, logger log.Logger) biz.NewFeatureRepo {
    return &newFeatureRepo{data: data, log: log.NewHelper(logger)}
}

// 2. 添加到 ProviderSet（internal/data/data.go）
var ProviderSet = wire.NewSet(
    NewData,
    NewAuthRepo,
    NewNewFeatureRepo,  // 新增
)

// 3. 运行 make wire
```

**场景 3：注入外部依赖**
```go
// 1. 创建外部依赖的 Provider（如 JWT Manager）
func NewJWTManager(c *conf.JWT) (*jwt.Manager, error) {
    return jwt.NewManager(c.Secret, c.Expiration)
}

// 2. 添加到某个 ProviderSet（通常在 data 层）
var ProviderSet = wire.NewSet(
    NewData,
    NewJWTManager,  // 新增外部依赖
    NewAuthRepo,
)

// 3. 在 UseCase 中使用
func NewAuthUsecase(repo AuthRepo, jwt *jwt.Manager, logger log.Logger) *AuthUsecase {
    return &AuthUsecase{repo: repo, jwt: jwt}
}
```

### Wire 最佳实践

**1. 每层一个 ProviderSet**
```go
// internal/data/data.go
var ProviderSet = wire.NewSet(NewData, NewAuthRepo, NewUserRepo)

// internal/biz/biz.go
var ProviderSet = wire.NewSet(NewAuthUsecase, NewUserUsecase)

// internal/service/service.go
var ProviderSet = wire.NewSet(NewAuthService, NewUserService)

// internal/server/server.go
var ProviderSet = wire.NewSet(NewGRPCServer, NewHTTPServer)
```

**2. 使用接口绑定**
```go
// 当返回接口类型时，明确指定绑定
var ProviderSet = wire.NewSet(
    NewAuthRepo,
    wire.Bind(new(biz.AuthRepo), new(*authRepo)),  // 绑定接口到实现
)
```

**3. 处理多个同类型依赖**
```go
// 使用命名 Provider
func NewRedisClient(c *conf.Redis) (*redis.Client, error) { ... }
func NewCacheRedis(c *conf.Redis) (*redis.Client, error) { ... }

// 或使用结构体包装
type CacheRedis struct {
    *redis.Client
}
```

**4. 清理资源**
```go
// 返回 cleanup 函数
func NewData(c *conf.Data) (*Data, func(), error) {
    // 初始化...
    cleanup := func() {
        db.Close()
        redis.Close()
    }
    return data, cleanup, nil
}

// Wire 会自动组合所有 cleanup 函数
```

## AI Agent 工作指南

### 添加新服务

**完整流程**：

1. **创建服务目录结构**
```bash
# 创建服务目录
mkdir -p app/newservice/service/{cmd/server,internal/{biz,data,service,server},configs,deployment}

# 创建 Makefile
echo 'include ../../../app.mk' > app/newservice/service/Makefile
```

2. **定义 Proto API**
```bash
# 创建 proto 文件
mkdir -p api/protos/newservice/service/v1
# 编写 newservice.proto（定义服务接口）
```

3. **配置代码生成**
```bash
# 复制并修改 OpenAPI 生成配置
cp api/buf.servora.openapi.gen.yaml api/buf.newservice.openapi.gen.yaml
# 修改 out 路径为 app/newservice/service/openapi.yaml
```

4. **生成代码**
```bash
make gen  # 生成 protobuf + openapi
```

5. **实现服务代码**（参考 `app/sayhello/service/` 作为模板）

**实现顺序**：
```
数据层 → 业务层 → 服务层 → 服务器层 → 入口文件
```

**最小化实现清单**：
- [ ] `internal/data/data.go` - 数据层初始化 + ProviderSet
- [ ] `internal/biz/biz.go` - UseCase 实现 + ProviderSet
- [ ] `internal/service/service.go` - API 实现 + ProviderSet
- [ ] `internal/server/server.go` - gRPC/HTTP 服务器 + ProviderSet
- [ ] `cmd/server/main.go` - 主函数
- [ ] `cmd/server/wire.go` - Wire 配置
- [ ] `configs/config.yaml` - 服务配置

6. **Wire 依赖注入**
```bash
cd app/newservice/service
make wire  # 生成 wire_gen.go
```

7. **配置和测试**
```bash
# 复制配置示例
cp ../../../api/protos/conf/v1/config-example.yaml configs/config.yaml
# 编辑 config.yaml（配置数据库、Redis 等）

# 运行服务
make run

# 运行测试
make test
```

### 添加新业务模块（在现有服务内）

**场景**：在 `servora` 服务中添加新模块（如 `product`）

**步骤**：

1. **定义 Proto API**
```proto
// api/protos/product/service/v1/product.proto
syntax = "proto3";

package product.service.v1;

import "google/api/annotations.proto";
import "errors/errors.proto";

option go_package = "github.com/horonlee/servora/api/gen/go/product/service/v1;v1";

service Product {
  rpc CreateProduct(CreateProductRequest) returns (CreateProductReply);
  rpc GetProduct(GetProductRequest) returns (GetProductReply);
}

message CreateProductRequest {
  string name = 1;
  double price = 2;
}

message CreateProductReply {
  uint64 id = 1;
}

// 定义错误
enum ErrorReason {
  PRODUCT_NOT_FOUND = 0;
  INVALID_PRODUCT_DATA = 1;
}
```

2. **生成代码**
```bash
make gen
```

3. **实现数据层**
```go
// internal/data/product.go
package data

type productRepo struct {
    data *Data
    log  *log.Helper
}

func NewProductRepo(data *Data, logger log.Logger) biz.ProductRepo {
    return &productRepo{data: data, log: log.NewHelper(logger)}
}

func (r *productRepo) CreateProduct(ctx context.Context, p *biz.Product) error {
    // 实现数据库操作
    return nil
}
```

4. **实现业务层**
```go
// internal/biz/product.go
package biz

type Product struct {
    ID    uint64
    Name  string
    Price float64
}

type ProductRepo interface {
    CreateProduct(ctx context.Context, p *Product) error
    GetProduct(ctx context.Context, id uint64) (*Product, error)
}

type ProductUsecase struct {
    repo ProductRepo
    log  *log.Helper
}

func NewProductUsecase(repo ProductRepo, logger log.Logger) *ProductUsecase {
    return &ProductUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (uc *ProductUsecase) CreateProduct(ctx context.Context, name string, price float64) (uint64, error) {
    // 业务逻辑
    p := &Product{Name: name, Price: price}
    return 0, uc.repo.CreateProduct(ctx, p)
}
```

5. **实现服务层**
```go
// internal/service/product.go
package service

type ProductService struct {
    productv1.UnimplementedProductServer
    uc *biz.ProductUsecase
}

func NewProductService(uc *biz.ProductUsecase) *ProductService {
    return &ProductService{uc: uc}
}

func (s *ProductService) CreateProduct(ctx context.Context, req *productv1.CreateProductRequest) (*productv1.CreateProductReply, error) {
    id, err := s.uc.CreateProduct(ctx, req.Name, req.Price)
    if err != nil {
        return nil, err
    }
    return &productv1.CreateProductReply{Id: id}, nil
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

7. **注册到 gRPC 服务器**
```go
// internal/server/grpc.go
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
cd app/servora/service
make wire
```

### 前端开发（Vue 3）

**位置**：仓库根目录 `web/`

**常用命令**：
```bash
cd web

# 安装依赖
bun install

# 开发服务器
bun dev

# 构建生产版本
bun build

# 单元测试（Vitest）
bun test:unit
bun test:unit src/__tests__/component.spec.ts  # 单个文件

# E2E 测试（Playwright）
npx playwright install  # 首次安装浏览器
bun test:e2e
bun test:e2e e2e/login.spec.ts --project=chromium  # 单个测试

# 代码检查
bun lint
bun format
```

**项目结构**：
```
web/
├── src/
│   ├── components/       # Vue 组件
│   ├── views/           # 页面组件
│   ├── router/          # Vue Router
│   ├── stores/          # Pinia 状态管理
│   ├── api/             # API 客户端
│   └── __tests__/       # 单元测试
├── e2e/                 # E2E 测试
├── public/              # 静态资源
├── vite.config.ts       # Vite 配置
├── playwright.config.ts # Playwright 配置
└── package.json
```

**TypeScript 规范**：
- 使用 `<script setup lang="ts">` 组合式 API
- 禁止使用 `as any` 或 `@ts-ignore`
- 所有组件必须类型化
- API 调用必须定义接口类型

### 使用 Ent / GORM GEN 双 ORM

**场景**：为数据库表生成类型安全的 DAO

**步骤**：

1. **配置 GORM GEN**

通过 `svr gen gorm` 命令生成 GORM DAO/PO 代码，该工具由项目 `cmd/svr/` 提供：

```bash
# 为指定服务生成 GORM DAO/PO
svr gen gorm servora

# 预览生成路径（不实际生成）
svr gen gorm servora --dry-run

# 无参数进入交互式服务选择
svr gen gorm
```

2. **运行生成**
```bash
cd app/servora/service
make gen.gorm  # 内部调用 svr gen gorm
make gen.ent  # 生成 Ent 代码（schema -> ent）
```

3. **使用生成的 DAO**
```go
// internal/data/data.go
    import "github.com/horonlee/servora/app/servora/service/internal/data/gorm/po"

type Data struct {
    DB      *gorm.DB
    UserDAO *po.User    // 生成的 DAO
}

func NewData(c *conf.Data) (*Data, error) {
    db, _ := gorm.Open(mysql.Open(c.Database.Source))
    po.SetDefault(db)  // 初始化 DAO

    return &Data{
        DB:      db,
        UserDAO: po.User,  // 单例 DAO
    }, nil
}

// internal/data/user.go
func (r *userRepo) GetByID(ctx context.Context, id uint64) (*biz.User, error) {
    // 使用类型安全的查询
    u, err := r.data.UserDAO.Where(r.data.UserDAO.ID.Eq(id)).First()
    if err != nil {
        return nil, err
    }

    return &biz.User{
        ID:       u.ID,
        Username: u.Username,
    }, nil
}
```

### 测试开发

**单元测试模板**：
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

**集成测试（跳过外部依赖）**：
```go
func TestUserRepo_Create(t *testing.T) {
    // 尝试连接数据库
    db, err := gorm.Open(mysql.Open("test:test@tcp(localhost:3306)/test"))
    if err != nil {
        t.Skipf("database not available: %v", err)
        return
    }

    // 运行测试
    repo := NewUserRepo(&Data{DB: db}, nil)
    // ...
}
```

### 部署配置

**Compose 镜像构建**：
```bash
cd /Users/horonlee/projects/go/servora
make compose.build  # 构建生产镜像（servora + sayhello）
```

**Kubernetes 部署**：
```bash
# 部署服务
kubectl apply -f deployment/kubernetes/

# 查看状态
kubectl get pods -l app=servora
kubectl logs -f deployment/servora
```

## 常见任务速查

### 开发工作流
```bash
# 1. 修改 proto 文件
vim api/protos/auth/service/v1/auth.proto

# 2. 生成代码
make gen

# 3. 修改 Wire 配置（如有必要）
vim app/servora/service/cmd/server/wire.go

# 4. 重新生成 Wire
cd app/servora/service && make wire

# 5. 运行服务
make run

# 6. 运行测试
make test
```

### 调试技巧
```bash
# 查看生成的 Wire 代码
cat app/servora/service/cmd/server/wire_gen.go

# 检查 Wire 依赖图
cd app/servora/service/cmd/server
wire show

# 验证 gRPC 接口
grpcurl -plaintext localhost:9000 list
grpcurl -plaintext localhost:9000 auth.service.v1.Auth/Login

# 查看 OpenAPI 文档
cat app/servora/service/openapi.yaml
```

### 性能优化
- 使用 Redis 缓存热点数据
- 数据库查询优化（索引、批量操作）
- gRPC 连接池复用
- HTTP/2 多路复用

### 错误处理最佳实践
```go
// 使用 Kratos 错误类型
import userv1 "github.com/horonlee/servora/api/gen/go/user/service/v1"

// 标准错误
return userv1.ErrorUserNotFound("user %d not found", id)

// 自定义错误
return errors.Wrapf(err, "failed to create user")

// 错误传播（保留堆栈）
if err != nil {
    return nil, errors.Wrap(err, "database query failed")
}
```

## 注意事项

### 代码生成
- 修改 `.proto` 文件后必须运行 `make gen`
- 修改 `wire.go` 后必须运行 `make wire`
- 生成的代码（`wire_gen.go`, `api/gen/`）不要手动编辑
- 生成的代码已在 `.gitignore` 中，不应提交

### 依赖注入
- 每个构造函数应返回接口类型（而非具体类型）
- 使用 `wire.Bind()` 显式绑定接口到实现
- 避免循环依赖（Wire 会在编译时检测）
- 清理资源通过返回 `cleanup func()` 实现

### 性能和可靠性
- 数据库连接池配置（`max_open_conns`, `max_idle_conns`）
- Redis 连接超时和重试策略
- gRPC 超时和截止时间（context deadline）
- 使用 context 传递请求上下文

### 安全性
- 永远不要在日志中打印敏感信息（密码、token）
- 使用参数化查询防止 SQL 注入
- JWT secret 必须从配置文件读取，不要硬编码
- CORS 配置要严格限制允许的 origin

## 依赖关系

**上游依赖**（本目录依赖的其他目录）：
- `../api/gen/go/` - 生成的 protobuf Go 代码（proto 接口定义）
- `../pkg/` - 共享库（jwt, redis, logger 等工具）
- `../api/protos/conf/v1/` - 配置文件定义（config.proto）

**下游依赖**（依赖本目录的其他目录）：
- `../deployment/` - 部署配置（需要编译好的服务二进制）
- `../manifests/` - Kubernetes 配置（需要 Docker 镜像）

**外部依赖**：
- Kratos v2 框架
- Ent + GORM GEN（双 ORM）
- Wire（依赖注入）
- Redis（缓存）
- 数据库驱动（MySQL/PostgreSQL/SQLite）

## 快速参考

**启动 servora 服务**：
```bash
cd app/servora/service
cp ../../../api/protos/conf/v1/config-example.yaml configs/config.yaml
# 编辑 configs/config.yaml（配置数据库和 Redis）
make run
```

**启动 sayhello 服务**：
```bash
cd app/sayhello/service
make run
```

**创建新服务**：
```bash
# 1. 创建目录
mkdir -p app/myservice/service

# 2. 复制模板
cp -r app/sayhello/service/* app/myservice/service/

# 3. 修改 import 路径和包名
# 4. 定义 proto API
# 5. make gen && make wire && make run
```
