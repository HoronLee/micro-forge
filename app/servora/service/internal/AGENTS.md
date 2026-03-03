# AGENTS.md - internal/ DDD 分层架构实现

<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-09 | Updated: 2026-02-26 -->

## 目录概述

`app/servora/service/internal/` 是 servora 服务的核心业务实现层，采用严格的 DDD（领域驱动设计）三层架构。这个目录包含所有内部实现代码，通过清晰的层次划分实现业务逻辑与基础设施的解耦。

**核心价值**：
- 严格的层次分离：biz（业务逻辑）、data（数据访问）、service（接口适配）
- 依赖倒置原则：biz 层定义接口，data 层实现接口
- 可测试性：每层都可以独立测试，通过 mock 实现
- 可维护性：职责清晰，修改影响范围可控

## 目录结构

```
internal/
├── biz/                    # 业务逻辑层（UseCase 层）
│   ├── biz.go             # ProviderSet 定义
│   ├── entity.go          # 领域实体定义（共享的 Domain Object）
│   ├── auth.go            # 认证业务逻辑（AuthUsecase + AuthRepo 接口）
│   ├── user.go            # 用户业务逻辑（UserUsecase + UserRepo 接口）
│   ├── test.go            # 测试业务逻辑（TestUsecase）
│   └── README.md          # Biz 层说明文档
├── data/                  # 数据访问层（Repository 实现层）
│   ├── data.go            # Data 初始化 + ProviderSet（DB, Redis 连接）
│   ├── discovery.go       # 服务发现客户端（gRPC 服务间调用）
│   ├── auth.go            # AuthRepo 接口实现
│   ├── user.go            # UserRepo 接口实现
│   ├── test.go            # TestRepo 接口实现
│   ├── auth_property_test.go  # 属性测试示例
│   ├── schema/            # Ent Schema 定义（手写）
│   ├── ent/               # Ent 生成代码
│   ├── gorm/po/           # GORM GEN 生成的持久化对象（PO）
│   │   └── *.gen.go       # 自动生成的数据模型
│   ├── gorm/dao/          # GORM GEN 生成的 DAO（并行保留）
│   │   ├── gen.go         # DAO 生成配置
│   │   └── *.gen.go       # 自动生成的查询接口
│   └── README.md          # Data 层说明文档
├── service/               # 服务层（gRPC/HTTP 接口实现层）
│   ├── service.go         # ProviderSet 定义
│   ├── auth.go            # Auth gRPC 服务实现
│   ├── user.go            # User gRPC 服务实现
│   ├── test.go            # Test gRPC 服务实现
│   └── README.md          # Service 层说明文档
├── server/                # 服务器配置层
│   ├── server.go          # ProviderSet + 服务器工厂
│   ├── grpc.go            # gRPC 服务器配置（端口、中间件、注册）
│   ├── http.go            # HTTP 服务器配置（端口、路由、CORS）
│   ├── registry.go        # 服务注册中心配置（Consul/Nacos/etcd）
│   ├── metrics.go         # Prometheus 指标采集
│   └── middleware/        # 中间件
│       ├── middleware.go  # 中间件集合
│       └── AuthJWT.go     # JWT 认证中间件
└── consts/                # 常量定义
    └── user.go            # 用户相关常量
```

## DDD 三层架构详解

### 架构图

```
┌─────────────────────────────────────────────────────────────┐
│  外部调用层（External Clients）                              │
│  - gRPC 客户端                                               │
│  - HTTP 客户端（浏览器、移动端）                              │
│  - 其他微服务                                                 │
└─────────────────┬───────────────────────────────────────────┘
                  │ RPC/HTTP 调用
┌─────────────────▼───────────────────────────────────────────┐
│  服务层（internal/service/）                                 │
│  职责：实现 proto 定义的接口，参数验证和转换                  │
│  - AuthService, UserService, TestService                     │
│  - DTO ↔ proto 消息转换                                      │
│  - 错误处理和响应封装                                         │
└─────────────────┬───────────────────────────────────────────┘
                  │ 依赖（通过接口）
┌─────────────────▼───────────────────────────────────────────┐
│  业务逻辑层（internal/biz/）                                 │
│  职责：核心业务逻辑，领域模型，业务规则                       │
│  - AuthUsecase, UserUsecase, TestUsecase                     │
│  - 定义 Repository 接口（AuthRepo, UserRepo）                │
│  - 定义领域实体（User, Entity）                              │
│  - 编排数据访问和外部服务调用                                 │
└─────────────────┬───────────────────────────────────────────┘
                  │ 依赖（通过接口）
┌─────────────────▼───────────────────────────────────────────┐
│  数据访问层（internal/data/）                                │
│  职责：Repository 接口实现，数据持久化，外部服务调用          │
│  - authRepo, userRepo, testRepo（实现 biz 层定义的接口）     │
│  - Ent 默认仓储 + GORM GEN DAO 并行保留                        │
│  - Redis 缓存操作                                            │
│  - gRPC 客户端（服务间调用）                                  │
│  - PO（持久化对象）↔ DO（领域对象）转换                      │
└─────────────────┬───────────────────────────────────────────┘
                  │ 访问
┌─────────────────▼───────────────────────────────────────────┐
│  基础设施层（Infrastructure）                                │
│  - MySQL / PostgreSQL / SQLite（数据库）                     │
│  - Redis（缓存）                                             │
│  - 外部微服务（gRPC）                                         │
│  - 服务注册中心（Consul / Nacos / etcd）                      │
└─────────────────────────────────────────────────────────────┘
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

**正确示例**：
```go
// ✅ biz 层定义接口
// internal/biz/auth.go
type AuthRepo interface {
    GetUserByUsername(ctx context.Context, username string) (*User, error)
}

// ✅ data 层实现接口
// internal/data/auth.go
type authRepo struct { data *Data; log *log.Helper }

func NewAuthRepo(data *Data, logger log.Logger) biz.AuthRepo {
    return &authRepo{data: data, log: log.NewHelper(logger)}
}
```

**错误示例**：
```go
// ❌ 错误：biz 层依赖 data 层的具体类型
// internal/biz/auth.go
import "github.com/horonlee/servora/app/servora/service/internal/data"

type AuthUsecase struct {
    repo *data.AuthRepo  // 错误！依赖了具体类型
}
```

## 各层详细说明

### 1. 业务逻辑层（biz/）

**职责**：实现核心业务逻辑（UseCase），定义领域模型和 Repository 接口。

**关键文件**：
- `biz.go` - Wire ProviderSet 定义
- `entity.go` - 共享的领域实体定义
- `auth.go` - 认证业务逻辑（登录、注册、Token 验证）
- `user.go` - 用户业务逻辑（CRUD、个人资料管理）
- `test.go` - 测试业务逻辑（gRPC 调用示例）

**典型结构**：
```go
// internal/biz/auth.go
package biz

// 1. 定义领域模型（Domain Object）
type User struct {
    ID       uint64
    Username string
    Password string  // 哈希后的密码
    Email    string
}

// 2. 定义 Repository 接口（依赖倒置）
type AuthRepo interface {
    GetUserByUsername(ctx context.Context, username string) (*User, error)
    CreateUser(ctx context.Context, user *User) error
}

// 3. 定义 UseCase
type AuthUsecase struct {
    repo AuthRepo       // Repository 接口
    jwt  *jwt.Manager   // 外部依赖
    log  *log.Helper
}

func NewAuthUsecase(repo AuthRepo, jwt *jwt.Manager, logger log.Logger) *AuthUsecase {
    return &AuthUsecase{
        repo: repo,
        jwt:  jwt,
        log:  log.NewHelper(logger),
    }
}

// 4. 实现业务逻辑方法
func (uc *AuthUsecase) Login(ctx context.Context, username, password string) (string, error) {
    // 业务规则：查询用户
    user, err := uc.repo.GetUserByUsername(ctx, username)
    if err != nil {
        return "", errors.New("user not found")
    }

    // 业务规则：验证密码
    if !hash.VerifyPassword(password, user.Password) {
        return "", errors.New("invalid password")
    }

    // 业务规则：生成 Token
    token, err := uc.jwt.GenerateToken(user.ID, user.Username)
    if err != nil {
        return "", errors.New("failed to generate token")
    }

    uc.log.Infof("user %s logged in successfully", username)
    return token, nil
}
```

**ProviderSet 定义**：
```go
// internal/biz/biz.go
package biz

import "github.com/google/wire"

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
    NewAuthUsecase,
    NewUserUsecase,
    NewTestUsecase,
)
```

**特点**：
- 不依赖任何具体的技术实现（数据库、框架等）
- 所有外部依赖通过接口注入
- 业务规则集中在这一层
- 容易编写单元测试（通过 mock Repository）

### 2. 数据访问层（data/）

**职责**：实现 biz 层定义的 Repository 接口，处理数据持久化和外部服务调用。

**关键文件**：
- `data.go` - Data 初始化（DB, Redis 连接）+ ProviderSet
- `discovery.go` - 服务发现客户端配置
- `auth.go` - AuthRepo 接口实现
- `user.go` - UserRepo 接口实现
- `test.go` - TestRepo 接口实现
- `schema/` - Ent Schema 定义（手写）
- `ent/` - Ent 生成代码（自动生成，不要手动编辑）
- `gorm/po/` - GORM GEN 生成的持久化对象（自动生成，不要手动编辑）
- `gorm/dao/` - GORM GEN 生成的 DAO（自动生成，不要手动编辑）

**典型结构**：
```go
// internal/data/auth.go
package data

import (
    "context"

    "github.com/go-kratos/kratos/v2/log"
    "github.com/horonlee/servora/app/servora/service/internal/biz"
    "gorm.io/gorm"
)

// Repository 实现
type authRepo struct {
    data *Data
    log  *log.Helper
}

// 返回接口类型（而非具体类型）
func NewAuthRepo(data *Data, logger log.Logger) biz.AuthRepo {
    return &authRepo{
        data: data,
        log:  log.NewHelper(logger),
    }
}

// 实现接口方法
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

**Data 初始化**：
```go
// internal/data/data.go
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
```

**特点**：
- 实现 biz 层定义的接口
- 使用 GORM GEN 实现类型安全的数据库操作
- 处理 PO（持久化对象）与 DO（领域对象）的转换
- 封装 Redis 缓存逻辑
- 封装 gRPC 客户端调用（服务间通信）

### 3. 服务层（service/）

**职责**：实现 proto 定义的 gRPC/HTTP 接口，作为外部世界与业务逻辑的适配层。

**关键文件**：
- `service.go` - Wire ProviderSet 定义
- `auth.go` - Auth gRPC 服务实现
- `user.go` - User gRPC 服务实现
- `test.go` - Test gRPC 服务实现

**典型结构**：
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

**ProviderSet 定义**：
```go
// internal/service/service.go
package service

import "github.com/google/wire"

// ProviderSet is service providers.
var ProviderSet = wire.NewSet(
    NewAuthService,
    NewUserService,
    NewTestService,
)
```

**特点**：
- 薄适配层，不包含业务逻辑
- 一个 proto 服务对应一个 service 文件
- 参数验证（基本的合法性检查）
- DTO 转换（proto 消息 ↔ 业务实体）
- 错误处理（将业务错误转换为 gRPC 错误码）

### 4. 服务器配置层（server/）

**职责**：配置和启动 gRPC/HTTP 服务器，注册服务，配置中间件。

**关键文件**：
- `server.go` - ProviderSet + 服务器工厂
- `grpc.go` - gRPC 服务器配置（端口、中间件、服务注册）
- `http.go` - HTTP 服务器配置（端口、路由、CORS）
- `registry.go` - 服务注册中心配置
- `metrics.go` - Prometheus 指标采集
- `middleware/` - 中间件实现（JWT 认证、日志、限流等）

**典型结构**：
```go
// internal/server/grpc.go
package server

import (
    "github.com/go-kratos/kratos/v2/middleware/recovery"
    "github.com/go-kratos/kratos/v2/transport/grpc"

    authv1 "github.com/horonlee/servora/api/gen/go/auth/service/v1"
    userv1 "github.com/horonlee/servora/api/gen/go/user/service/v1"
    "github.com/horonlee/servora/app/servora/service/internal/service"
)

// NewGRPCServer 创建 gRPC 服务器
func NewGRPCServer(
    c *conf.Server,
    authSvc *service.AuthService,
    userSvc *service.UserService,
    logger log.Logger,
) *grpc.Server {
    var opts = []grpc.ServerOption{
        grpc.Middleware(
            recovery.Recovery(),  // 异常恢复
            logging.Server(logger),  // 日志
        ),
    }

    if c.Grpc.Network != "" {
        opts = append(opts, grpc.Network(c.Grpc.Network))
    }
    if c.Grpc.Addr != "" {
        opts = append(opts, grpc.Address(c.Grpc.Addr))
    }
    if c.Grpc.Timeout != nil {
        opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
    }

    srv := grpc.NewServer(opts...)

    // 注册服务
    authv1.RegisterAuthServer(srv, authSvc)
    userv1.RegisterUserServer(srv, userSvc)

    return srv
}
```

**ProviderSet 定义**：
```go
// internal/server/server.go
package server

import "github.com/google/wire"

// ProviderSet is server providers.
var ProviderSet = wire.NewSet(
    NewGRPCServer,
    NewHTTPServer,
    NewRegistrar,
)
```

### 5. 常量层（consts/）

**职责**：定义项目中使用的常量。

**关键文件**：
- `user.go` - 用户相关常量（状态、角色、权限等）

**典型结构**：
```go
// internal/consts/user.go
package consts

const (
    UserStatusActive   = 1
    UserStatusInactive = 0
    UserStatusDeleted  = -1

    UserRoleAdmin = "admin"
    UserRoleUser  = "user"
)
```

## AI Agent 工作指南

### 添加新业务模块

**场景**：在现有服务中添加新的业务模块（如 `product`）

**实现顺序**：数据层 → 业务层 → 服务层 → 服务器注册

**步骤**：

1. **定义 Proto API**（在 `api/protos/` 目录）
```bash
# 创建并编辑 proto 文件
mkdir -p /Users/horonlee/projects/go/servora/api/protos/product/service/v1
vim /Users/horonlee/projects/go/servora/api/protos/product/service/v1/product.proto

# 生成代码
cd /Users/horonlee/projects/go/servora
make gen
```

2. **实现数据层**
```go
// internal/data/product.go
package data

import (
    "context"
    "github.com/horonlee/servora/app/servora/service/internal/biz"
)

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

// 更新 internal/data/data.go 的 ProviderSet
var ProviderSet = wire.NewSet(
    NewData,
    NewAuthRepo,
    NewProductRepo,  // 新增
)
```

3. **实现业务层**
```go
// internal/biz/product.go
package biz

import "context"

// 定义领域模型
type Product struct {
    ID    uint64
    Name  string
    Price float64
}

// 定义 Repository 接口
type ProductRepo interface {
    CreateProduct(ctx context.Context, p *Product) error
    GetProduct(ctx context.Context, id uint64) (*Product, error)
}

// 定义 UseCase
type ProductUsecase struct {
    repo ProductRepo
    log  *log.Helper
}

func NewProductUsecase(repo ProductRepo, logger log.Logger) *ProductUsecase {
    return &ProductUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (uc *ProductUsecase) CreateProduct(ctx context.Context, name string, price float64) (uint64, error) {
    p := &Product{Name: name, Price: price}
    return 0, uc.repo.CreateProduct(ctx, p)
}

// 更新 internal/biz/biz.go 的 ProviderSet
var ProviderSet = wire.NewSet(
    NewAuthUsecase,
    NewProductUsecase,  // 新增
)
```

4. **实现服务层**
```go
// internal/service/product.go
package service

import (
    "context"
    productv1 "github.com/horonlee/servora/api/gen/go/product/service/v1"
    "github.com/horonlee/servora/app/servora/service/internal/biz"
)

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

// 更新 internal/service/service.go 的 ProviderSet
var ProviderSet = wire.NewSet(
    NewAuthService,
    NewProductService,  // 新增
)
```

5. **注册到 gRPC 服务器**
```go
// 修改 internal/server/grpc.go
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

6. **重新生成 Wire 代码**
```bash
cd /Users/horonlee/projects/go/servora/app/servora/service
make wire
```

7. **运行和测试**
```bash
make run
```

### 添加新中间件

**场景**：添加新的 HTTP 或 gRPC 中间件

**步骤**：

1. **创建中间件实现**
```go
// internal/server/middleware/ratelimit.go
package middleware

import (
    "context"
    "github.com/go-kratos/kratos/v2/middleware"
)

func RateLimit(limiter *rate.Limiter) middleware.Middleware {
    return func(handler middleware.Handler) middleware.Handler {
        return func(ctx context.Context, req interface{}) (interface{}, error) {
            if !limiter.Allow() {
                return nil, errors.New("rate limit exceeded")
            }
            return handler(ctx, req)
        }
    }
}
```

2. **在服务器中使用**
```go
// internal/server/grpc.go
func NewGRPCServer(...) *grpc.Server {
    var opts = []grpc.ServerOption{
        grpc.Middleware(
            recovery.Recovery(),
            middleware.RateLimit(limiter),  // 新增
        ),
    }
    // ...
}
```

### 使用 GORM GEN 生成 DAO

**场景**：为新的数据库表生成类型安全的 DAO

**步骤**：

1. **运行生成**
```bash
# 在服务目录下运行
cd /Users/horonlee/projects/go/servora/app/servora/service
make gen.gorm

# 或通过根目录 svr CLI 工具直接调用
svr gen gorm servora

# 预览将要生成的路径（不实际执行）
svr gen gorm servora --dry-run
```

3. **使用生成的 DAO**
```go
// internal/data/product.go
func (r *productRepo) GetByID(ctx context.Context, id uint64) (*biz.Product, error) {
    p, err := r.data.query.Product.
        WithContext(ctx).
        Where(r.data.query.Product.ID.Eq(id)).
        First()

    if err != nil {
        return nil, err
    }

    return &biz.Product{
        ID:    p.ID,
        Name:  p.Name,
        Price: p.Price,
    }, nil
}
```

### 编写单元测试

**场景**：为业务逻辑层编写单元测试

**步骤**：

1. **创建 Mock Repository**
```go
// internal/biz/auth_test.go
package biz

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

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

func (m *mockAuthRepo) CreateUser(ctx context.Context, user *User) error {
    args := m.Called(ctx, user)
    return args.Error(0)
}
```

2. **编写测试用例**
```go
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

3. **运行测试**
```bash
cd /Users/horonlee/projects/go/servora/app/servora/service
go test ./internal/biz/... -v
```

## 常见任务速查

### 开发工作流

```bash
# 1. 修改 proto 文件（在 api/protos/ 目录）
vim /Users/horonlee/projects/go/servora/api/protos/auth/service/v1/auth.proto

# 2. 生成代码（在项目根目录）
cd /Users/horonlee/projects/go/servora
make gen

# 3. 实现业务逻辑（biz → data → service）
# 编辑 internal/biz/, internal/data/, internal/service/ 下的文件

# 4. 更新 ProviderSet（如果添加了新模块）
# 编辑 internal/biz/biz.go, internal/data/data.go, internal/service/service.go

# 5. 重新生成 Wire（在服务目录）
cd /Users/horonlee/projects/go/servora/app/servora/service
make wire

# 6. 运行服务
make run

# 7. 运行测试
make test
```

### 调试技巧

```bash
# 查看生成的 Wire 代码
cat /Users/horonlee/projects/go/servora/app/servora/service/cmd/server/wire_gen.go

# 检查 Wire 依赖图
cd /Users/horonlee/projects/go/servora/app/servora/service/cmd/server
wire show

# 运行单个包的测试
go test -v ./internal/biz/
go test -v -run TestAuthUsecase_Login ./internal/biz/

# 查看数据库查询日志（在 configs/config.yaml 中配置）
# 设置 database.log_level: debug
```

## 注意事项

### 代码组织
- 每个业务模块一个文件（如 `auth.go`, `user.go`）
- 所有 ProviderSet 定义在 `<layer>.go` 文件中（如 `biz.go`, `data.go`）
- 生成的代码（`gorm/po/*.gen.go`, `gorm/dao/*.gen.go`）不要手动编辑

### 依赖注入
- 构造函数必须返回接口类型（而非具体类型）
- 每个新增的构造函数都要添加到对应层的 ProviderSet
- 修改 ProviderSet 后必须运行 `make wire`

### 错误处理
- 使用 Kratos 错误类型（从 proto 生成的错误）
- 示例：`authv1.ErrorUnauthorized("user not authenticated")`
- 在 data 层捕获基础设施错误，转换为业务错误

### 测试
- biz 层使用 mock Repository 进行单元测试
- data 层使用真实数据库进行集成测试（连接失败时 skip）
- service 层测试 DTO 转换和参数验证

## 依赖关系

**上游依赖**（本目录依赖的其他目录）：
- `/Users/horonlee/projects/go/servora/api/gen/go/` - 生成的 protobuf Go 代码
- `/Users/horonlee/projects/go/servora/pkg/` - 共享库（jwt, redis, logger, hash 等）

**下游依赖**（依赖本目录的其他目录）：
- `/Users/horonlee/projects/go/servora/app/servora/service/cmd/` - 服务入口（使用 Wire 生成的代码）

**外部依赖**：
- Kratos v2 框架
- GORM + GORM GEN
- Wire（依赖注入）
- Redis
- 数据库驱动（MySQL/PostgreSQL/SQLite）
