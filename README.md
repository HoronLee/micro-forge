# Servora

简体中文

servora 是一个基于 **Go Kratos v2** 的微服务快开框架，采用 **DDD 分层** 与 **契约优先（Proto First）** 的开发方式，覆盖从 API 定义、代码生成、服务开发到可观测性与容器化部署的完整链路。

## ✨ 核心能力

- **微服务模板化**：统一的服务目录约定与 `app.mk` 共享构建流程
- **Proto First**：使用 Buf 管理 Protobuf 依赖与代码生成
- **双协议接口**：同时支持 gRPC 与 HTTP（含 OpenAPI 生成）
- **DDD 分层**：`service -> biz -> data`，职责边界清晰
- **依赖注入**：使用 Wire 进行编译期依赖注入
- **数据访问**：Ent + GORM GEN 双工具链并行
- **服务治理**：支持 Consul / Nacos / etcd 注册发现与配置中心
- **可观测性**：OTel Collector + Jaeger + Loki + Prometheus + Grafana
- **开发体验**：支持 Docker Compose + Air 热重载开发

## 🧱 技术栈

- 框架：Kratos v2
- API：Protobuf + Buf
- DI：Google Wire
- ORM：Ent（主）+ GORM GEN（并行）
- 存储：MySQL / PostgreSQL / SQLite + Redis
- 前端：Vue 3 + Vite + Bun（位于根目录 `web/`）
- 观测：OTel / Jaeger / Loki / Prometheus / Grafana

## 🗂️ 项目结构

```text
.
├── api/                         # Proto 定义、Buf 配置、生成代码
│   ├── protos/
│   │   └── template/            # svr new api 脚手架模板（可自定义）
│   ├── gen/go/
│   ├── buf.gen.yaml
│   ├── buf.*.go.gen.yaml
│   ├── buf.*.typescript.gen.yaml
│   └── buf.*.openapi.gen.yaml
├── app/
│   ├── servora/service/         # 主服务（DDD 分层）
│   └── sayhello/service/        # 独立示例服务
├── cmd/
│   └── svr/                     # CLI 工具（svr gen gorm / svr new api）
├── pkg/                         # 项目共享库
├── web/                         # Vue 3 前端项目（根目录）
├── manifests/                   # 可观测性与证书等基础设施清单
├── deployment/                  # 部署相关配置
├── docker-compose.yaml          # 生产编排
├── docker-compose.dev.yaml      # 开发覆盖层（Air）
├── app.mk                       # 服务级通用 Makefile
└── Makefile                     # 根目录统一入口
```

## 🚀 快速开始

### 1) 前置要求

- Go 1.21+
- Make
- Docker / Docker Compose

### 2) 克隆仓库

```bash
git clone https://github.com/horonlee/servora.git
cd servora
```

按需修改 `app/servora/service/configs/config.yaml` 中的数据库、Redis、注册中心等配置。

### 3) 安装工具并且生成代码

```bash
make init
make gen
```

该命令会统一执行代码生成流程：`api + wire + openapi + ent`。

### 4) 容器化开发

```bash
make compose.dev
```

查看日志与停止：

```bash
make compose.dev.logs
make compose.dev.restart
make compose.dev.down
```

## 🧭 开发工作流

推荐顺序：

1. 使用 `svr new api <name>` 生成 proto 骨架（`api/protos/<name>/service/v1/`）
2. 修改/完善生成的 `.proto` 文件
3. 运行 `make gen` 同步生成代码
4. 按 DDD 分层实现：`internal/service -> internal/biz -> internal/data`
5. 若修改了 Wire 依赖图，运行 `make wire`（或直接 `make gen`）
6. 运行 `make test`、`make lint` 验证质量

## 🛠️ 常用命令

### 根目录命令

```bash
# 初始化工具
make init

# 代码生成
make gen
make api
make openapi
make wire
make ent

# 构建与质量
make build
make build_only
make test
make lint
make vet

# Compose（生产）
make compose.build
make compose.up
make compose.rebuild
make compose.ps
make compose.logs
make compose.down

# Compose（开发 Air）
make compose.dev
make compose.dev.build
make compose.dev.up
make compose.dev.restart
make compose.dev.ps
make compose.dev.logs
make compose.dev.down
```

`make api` 的模板执行约定：
- Go 代码生成自动扫描 `api/buf.*.go.gen.yaml` 并逐个执行；若未找到则回退到 `api/buf.gen.yaml`
- TypeScript 代码生成自动扫描 `api/buf.*.typescript.gen.yaml` 并逐个执行；若未找到则跳过 TS 生成

### 服务级命令（示例：`app/servora/service/`）

```bash
make run
make build
make build_only
make app
make gen
make wire
make gen.ent
make gen.gorm    # 生成 GORM DAO/PO（内部调用 svr gen gorm）
make openapi
```

### svr 命令行工具

```bash
# GORM GEN 代码生成
svr gen gorm <服务名...>          # 为指定服务生成 GORM DAO/PO
svr gen gorm servora --dry-run   # 预览生成路径（不实际生成）
svr gen gorm                     # 无参数进入交互式服务选择

# Proto API 脚手架
svr new api <name>               # 在 api/protos/ 下创建 proto 骨架
svr new api say_hello            # 支持 snake_case
svr new api billing.invoice      # 支持点分层级（生成嵌套目录）
svr new api user --output <dir>  # 指定输出根目录
svr new api user --template <dir> # 指定自定义模板目录
```

退出码：全部成功 = 0，存在失败 = 1

> **注意**：`svr` 命令须在项目根目录执行。

### 前端命令（`web/`）

```bash
cd web
bun install
bun dev
bun test:unit
bun test:e2e
bun lint
```

## 📦 配置说明

- 主服务配置：`app/servora/service/configs/config.yaml`
- 示例配置：`api/protos/conf/v1/config-example.yaml`
- 支持环境变量覆盖默认值（详见示例配置中的 `${VAR:default}` 写法）

核心配置块包括：

- `server`（HTTP/gRPC、TLS、CORS）
- `data`（数据库、Redis、客户端）
- `registry` / `discovery` / `config`（治理与配置中心）
- `trace` / `metrics`（观测）

## 🔭 可观测性

项目默认集成观测组件（Compose 生产栈）：

- Grafana: `http://localhost:3001`
- Prometheus: `http://localhost:9090`
- Jaeger: `http://localhost:16686`
- Loki: `http://localhost:3100`
- OTel Collector: `4317/4318`

## 🧪 质量与约束

- 不要手动编辑生成代码（如 `api/gen/go/`、`wire_gen.go`、`openapi.yaml`）
- 修改 Proto 后务必执行 `make gen`
- 修改 Wire 配置后务必重新生成（`make wire` 或 `make gen`）
- 新增 API 代码生成模板时请遵循命名：`api/buf.<name>.go.gen.yaml` 或 `api/buf.<name>.typescript.gen.yaml`，`make api` 会自动发现并执行

## 🤝 贡献

欢迎提交 Issue / PR。提交前请至少确保：

```bash
make lint
make test
```

## 📄 License

MIT，详见 `LICENSE`。
