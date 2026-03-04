# AGENTS.md - servora 项目根目录

<!-- Generated: 2026-02-09 | Updated: 2026-02-27 -->

## 项目概述

servora 是一个基于 Go Kratos v2 框架的微服务项目示例，展示了现代化微服务架构的最佳实践。项目采用 DDD（领域驱动设计）分层架构，使用 Buf 进行 Protobuf 管理，支持多服务独立开发和部署。

**核心价值**：
- 提供完整的微服务开发模板和最佳实践参考
- 统一的代码生成和构建流程（通过 `app.mk` 共享）
- 双协议支持（gRPC + HTTP）及自动 OpenAPI 文档生成
- 完善的服务治理（注册发现、配置中心、链路追踪）
- 生产级可观测性（OTel Collector + Jaeger + Loki + Prometheus + Grafana）

**技术栈**：
- **框架**: Kratos v2 (微服务框架)
- **API 定义**: Protobuf + Buf (现代化 Protobuf 工具链)
- **依赖注入**: Wire (编译时 DI)
- **ORM**: Ent + GORM GEN（双 ORM 并行支持）
- **前端**: Vue 3 + Vite (位于仓库根目录 `web/`)
- **服务治理**: Consul / Nacos / etcd
- **可观测性**: OTel Collector + Jaeger + Loki + Prometheus + Grafana

## 关键文件

### 构建配置
- **Makefile** - 根目录主构建文件，管理所有服务的统一构建流程
  - `make init` - 初始化开发环境（安装 buf, wire, protoc 插件等）
  - `make gen` - 生成所有代码（protobuf + wire + openapi）
  - `make build` - 构建所有服务
  - `make test` - 运行所有测试
  - `make lint` - 代码检查
- **app.mk** - 服务级通用 Makefile，被所有微服务共享
  - 定义了服务级的 `run`, `build`, `wire`, `gen.gorm`, `gen.ent` 等命令
  - 所有服务通过 `include ../../../app.mk` 复用构建逻辑
- **cmd/svr/** - servora 项目 CLI 工具（`svr gen gorm`、`svr new api` 等命令）
  - **执行约定**：`svr` 命令设计为在项目根目录执行。`svr new api` 默认输出路径 `./api/protos/` 相对于当前工作目录解析，CI 中须先 `cd` 到项目根目录再执行。

### Go 项目配置
- **go.mod** / **go.sum** - Go 模块依赖管理
- **.env.example** / **.env** - 环境变量配置

### 文档
- **README.md** - 项目主文档（包含快速开始、架构说明、使用指南）
- **CLAUDE.md** - AI 助手开发指南（构建命令、代码风格、测试模式）
- **TODO.md** - 项目待办事项（K8s 增强、技术债务等）
- **DEVELOPMENT.md** - 详细开发文档
- **LICENSE** - MIT 开源许可证

## 子目录结构

### `api/` - API 定义与代码生成
Protobuf API 定义的中心目录，包含所有 proto 文件和生成的代码。

**关键文件**：
- `buf.gen.yaml` - Buf 代码生成配置（Go protobuf 代码）
- `buf.{name}.go.gen.yaml` - Go 代码生成模板（根 Makefile 自动扫描执行）
- `buf.{name}.typescript.gen.yaml` - TypeScript 代码生成模板（根 Makefile 自动扫描执行）
- `buf.work.yaml` - Buf workspace 配置
- `buf.{service}.openapi.gen.yaml` - 各服务 OpenAPI 生成配置

**子目录**：
- `protos/` - Proto 源文件
  - `buf.yaml` - Proto 依赖配置（BSR 远程依赖）
  - `conf/v1/` - 配置定义（proto）与示例配置文件
  - `servora/service/v1/` - servora HTTP 接口（`i_*.proto` 文件）
  - `auth/service/v1/` - Auth gRPC 服务
  - `user/service/v1/` - User gRPC 服务
  - `test/service/v1/` - Test gRPC 服务
  - `sayhello/service/v1/` - SayHello 独立微服务
  - `template/service/v1/` - `svr new api` 脚手架模板（可自定义覆盖）
- `gen/go/` - 生成的 Go protobuf 代码（自动生成，不提交）

**Proto 组织规范**：
1. **HTTP 接口**：`servora/service/v1/i_*.proto`（统一包名 `servora.service.v1`）
2. **gRPC 服务**：`{service}/service/v1/{service}.proto`（独立包名）
3. **独立微服务**：完全独立的服务，可包含 HTTP 注解

### `app/` - 微服务实现
所有微服务的实现代码，每个服务独立目录。

**服务结构** (`app/{service}/service/`)：
- `cmd/server/` - 服务启动入口
  - `main.go` - 主函数
  - `wire.go` - Wire 依赖注入配置
  - `wire_gen.go` - Wire 生成的代码
- `internal/` - 内部实现（DDD 分层架构）
  - `biz/` - 业务逻辑层（UseCase）
  - `data/` - 数据访问层（Repository 实现）
  - `service/` - 服务层（API 接口实现）
  - `server/` - gRPC/HTTP 服务器配置
  - `client/` - 外部服务客户端（服务间调用）
- `configs/` - 服务配置文件（`config.yaml`）
- `bin/` - 编译输出目录
- `openapi.yaml` - 生成的 OpenAPI 文档
- `Makefile` - 服务级构建文件（include app.mk）

**当前服务**：
- `servora/service/` - 主服务（包含 auth, user, test 等模块）
- `sayhello/service/` - 独立微服务示例
- `web/` - 根目录前端应用（Vue 3 + Vite + TypeScript）

### `pkg/` - 共享库
项目内部共享的通用库，可被所有微服务复用。

**典型模块**：
- `jwt/` - JWT 认证工具
- `redis/` - Redis 客户端封装
- `logger/` - 日志工具
- `middleware/` - 通用中间件（CORS 等）
- `helpers/` - 通用辅助函数（含 bcrypt 哈希工具）

### `openspec/` - OpenSpec 规范文档
OpenSpec 变更管理系统，用于结构化跟踪架构变更和提案。

**关键文件**：
- `AGENTS.md` - OpenSpec AI 协作指南
- `changes/` - 变更提案目录（proposal → approved → deployed）

### `deployment/` - 部署配置
容器化和云原生部署配置。

**子目录**：
- `docker/` - Docker Compose 配置
- `kubernetes/` - Kubernetes 部署清单（service 级）

### `manifests/` - Kubernetes 配置
项目级 K8s 资源配置。

**子目录**：
- `certs/` - TLS 证书配置

### `docs/` - 文档
项目文档和知识库。

**子目录**：
- `knowledge/` - 知识文档（如 `k8s-service-governance.md`）

## AI Agent 工作指南

### 开发工作流

**标准开发流程**：
1. **定义 API** - 在 `api/protos/` 中编写 `.proto` 文件
2. **生成代码** - 运行 `make gen`（生成 protobuf + wire + openapi）
3. **实现业务逻辑** - 按 DDD 分层实现（biz → data → service）
4. **Wire 依赖注入** - 更新 `cmd/server/wire.go`，运行 `make wire`
5. **测试** - 运行 `make test`
6. **运行服务** - `cd app/{service}/service && make run`

### 常用命令

**根目录命令**（管理所有服务）：
```bash
make init          # 初始化开发环境
make gen           # 生成所有代码（api + wire + openapi）
make ent           # 聚合生成所有服务的 Ent 代码
make build         # 构建所有服务
make test          # 运行所有测试
make lint          # 代码检查（golangci-lint）
make clean         # 清理构建产物
make compose.build      # 构建生产镜像（servora + sayhello）
make compose.up         # 启动生产 compose 全栈（consul + db + redis + observability + services）
make compose.rebuild    # 重建生产镜像并启动生产 compose 全栈
make compose.ps         # 查看生产 compose 状态
make compose.logs       # 查看生产 compose 日志
make compose.down       # 停止生产 compose 全栈
make compose.dev.build  # 构建 Air 开发镜像（根 compose 分层）
make compose.dev.up     # 启动 Air 热重载开发容器（servora + sayhello）
make compose.dev.restart # 重启 Air 开发容器（强制触发启动时重编译）
make compose.dev.ps     # 查看 Air 开发容器状态
make compose.dev.logs   # 查看 Air 开发容器日志
make compose.dev.down   # 停止 Air 开发容器
```

# svr CLI 工具
```bash
svr gen gorm <服务名...>          # 为指定服务生成 GORM DAO/PO 代码
svr gen gorm servora --dry-run   # 预览生成路径
svr gen gorm                     # 交互式服务选择
```

`make api` 执行规则（自动扫描）：
- Go：执行 `api/buf.*.go.gen.yaml`；若无匹配文件，回退执行 `api/buf.gen.yaml`
- TypeScript：执行 `api/buf.*.typescript.gen.yaml`；若无匹配文件，跳过 TS 生成

**服务级命令**（在 `app/{service}/service/` 下执行）：
```bash
make run           # 运行服务（含代码生成）
make build         # 构建服务
make wire          # 生成 Wire 代码
make gen.gorm      # 生成 GORM GEN PO/DAO（内部调用 `svr gen gorm`）
make gen.ent       # 生成 Ent 代码
make test          # 运行测试
```

**前端命令**（在仓库根目录 `web/` 下执行）：
```bash
bun install && bun dev  # 开发服务器
bun test:unit           # Vitest 单元测试
bun test:e2e            # Playwright E2E 测试
bun lint                # ESLint 检查
```

**单独运行测试**：
```bash
# Go 测试
go test -v -run TestFunctionName ./path/to/package
go test -v ./pkg/redis/...

# 前端测试
bun test:unit src/__tests__/example.spec.ts
bun test:e2e e2e/example.spec.ts --project=chromium
```

### 代码风格规范

**Go 导入顺序**：
```go
import (
    "context"       // 1. 标准库

    "github.com/go-kratos/kratos/v2/log"  // 2. 第三方库

    authv1 "github.com/horonlee/servora/api/gen/go/auth/service/v1"  // 3. 项目内
)
```

**命名约定**：
- 接口：`UserRepo`, `AuthRepo`
- 构造函数：`NewUserUsecase`, `NewUserRepo`
- 私有类型：小写开头（`userRepo`）

**错误处理**：
使用 Kratos 错误类型（从生成的 proto 中导入）：
```go
return userv1.ErrorUserNotFound("user not found: %v", err)
return authv1.ErrorUnauthorized("user not authenticated")
```

**DDD 分层架构**：
- `service/` → `biz/` → `data/`
- 依赖方向：service 依赖 biz，biz 依赖 data（通过接口）
- 数据流：HTTP/gRPC 请求 → service → biz（业务逻辑）→ data（持久化）

**TypeScript/Vue 规范**：
- 使用 `<script setup lang="ts">` 组合式 API
- 禁止使用 `as any` 或 `@ts-ignore`
- 测试：Vitest（单元测试）、Playwright（E2E 测试）

### 测试模式

**表驱动测试**（推荐）：
```go
tests := []struct {
    name     string
    input    string
    expected bool
}{
    {"valid", "https://example.com", true},
    {"invalid", "https://bad.com", false},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        assert.Equal(t, tt.expected, isValid(tt.input))
    })
}
```

**跳过外部依赖**：
```go
client, err := redis.NewClient(cfg)
if err != nil {
    t.Skipf("redis not available: %v", err)
}
```

### 常见任务

**添加新服务**：
1. 创建目录：`mkdir -p app/newservice/service`
2. 创建 Makefile：`echo "include ../../../app.mk" > app/newservice/service/Makefile`
3. 创建 OpenAPI 配置：复制并修改 `api/buf.{service}.openapi.gen.yaml`
4. 定义 proto 文件：`api/protos/newservice/service/v1/`
5. 生成代码：`make gen`
6. 实现服务代码（参考 `app/servora/service/` 结构）

**修改 API**：
1. 编辑 `api/protos/` 中的 `.proto` 文件
2. 运行 `make gen`
3. 更新服务实现代码
4. 运行 `make test` 验证

**添加共享工具**：
1. 在 `pkg/` 下创建新模块
2. 在服务中通过 `github.com/horonlee/servora/pkg/{module}` 导入
3. 编写单元测试

**更新依赖注入**：
1. 编辑 `app/{service}/service/cmd/server/wire.go`
2. 在 `wire.Build()` 中添加新的 ProviderSet
3. 运行 `cd app/{service}/service && make wire`

## 依赖关系

### 构建依赖
- **Go 1.21+** - Go 运行时
- **Buf CLI** - Protobuf 管理（`buf generate`）
- **Wire** - 依赖注入代码生成
- **Make** - 构建工具
- **protoc 插件** - 通过 `make plugin` 安装
  - `protoc-gen-go`
  - `protoc-gen-go-grpc`
  - `protoc-gen-go-http`
  - `protoc-gen-go-errors`
  - `protoc-gen-openapi`

### 运行时依赖
- **Redis** - 缓存和会话存储（必需）
- **数据库**（可选）- MySQL / PostgreSQL / SQLite
- **服务注册中心**（可选）- Consul / Nacos / etcd
- **链路追踪**（可选）- Jaeger / Zipkin

### 前端依赖
- **Bun** - JavaScript 运行时和包管理器
- **Vue 3** - 前端框架
- **Vite** - 构建工具
- **Playwright** - E2E 测试（需 `npx playwright install`）

### 开发工具依赖
- **golangci-lint** - Go 代码检查（`make lint`）
- **Kratos CLI** - Kratos 命令行工具（`make cli` 安装）

## 注意事项

### 文件生成
- 运行 `make gen` 后会生成大量代码到 `api/gen/`，这些文件已加入 `.gitignore`
- 修改 proto 文件后必须运行 `make gen` 重新生成代码
- 修改 Wire 配置后必须运行 `make wire` 重新生成 `wire_gen.go`
- 生成的代码不应手动编辑，会在下次生成时被覆盖
- 新增 API 生成模板请使用命名：`api/buf.<name>.go.gen.yaml`、`api/buf.<name>.typescript.gen.yaml`，以便 `make api` 自动发现

### 配置管理
- 每个服务有独立的 `configs/config.yaml`
- 配置示例在 `api/protos/conf/v1/config-example.yaml`
- 支持通过环境变量覆盖配置（通过 `.env` 文件）
- 不要提交包含敏感信息的 `.env` 文件

### OpenSpec 工作流
- 重大架构变更、新功能、破坏性修改应使用 OpenSpec 流程
- 参考 `openspec/AGENTS.md` 了解详细流程
- 使用 `/openspec:proposal`, `/openspec:apply`, `/openspec:archive` 命令

### 常见陷阱
- 忘记运行 `make gen` 导致代码不同步
- 忘记运行 `make wire` 导致依赖注入失败
- 使用 Air 开发时把 `make gen` 放进 Air 启动命令，导致文件变更触发重启循环（正确流程：先宿主机 `make gen`，再 `make compose.dev.up`）
- 前端 E2E 测试需要先运行 `npx playwright install` 安装浏览器
- 测试需要外部服务时使用 `t.Skipf()` 优雅跳过
- 不要提交生成的代码（已在 `.gitignore` 中）
- Docker 构建需要在服务目录下执行，不要在根目录执行

## 快速参考

**初次使用**：
```bash
git clone https://github.com/horonlee/servora.git
cd servora
make init                                      # 安装工具
cp .env.example .env                           # 配置环境变量
make gen                                       # 生成代码
cd app/servora/service
cp ../../../api/protos/conf/v1/config-example.yaml configs/config.yaml
# 编辑 config.yaml，配置数据库和 Redis
make run                                       # 启动服务
```

**日常开发**：
```bash
# 修改 proto 文件后
make gen

# 修改 Wire 配置后
cd app/servora/service && make wire

# 运行测试
make test

# 代码检查
make lint
```

**部署**：
```bash
# 构建所有服务
make build

# 构建生产镜像
make compose.build

# K8s 部署
kubectl apply -f manifests/
kubectl apply -f app/servora/service/deployment/kubernetes/
```
