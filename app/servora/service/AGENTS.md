# AGENTS.md - app/servora/service 主服务

<!-- Parent: ../../AGENTS.md -->
<!-- Generated: 2026-02-09 | Updated: 2026-03-06 -->

## 目录定位

`app/servora/service/` 是主服务模块，当前同时包含：
- 服务私有 proto：`api/protos/`
- Go 后端：`cmd/` + `internal/`
- Vue 前端：`web/`
- OpenAPI 产物：`openapi.yaml`

这是一个独立 Go module，受根 `go.work` 管理。

## 当前结构

```text
app/servora/service/
├── api/
│   ├── buf.openapi.gen.yaml
│   ├── buf.typescript.gen.yaml
│   └── protos/
├── cmd/server/
├── configs/
├── internal/
├── manifests/
├── web/
├── go.mod
├── Makefile
└── openapi.yaml
```

## 代码层次

- `internal/biz/`：UseCase 与仓储接口
- `internal/biz/entity/`：领域实体
- `internal/data/`：Ent 默认实现，GORM GEN 作为并行工具链保留
- `internal/service/`：gRPC / HTTP 接口实现
- `internal/server/`：Server 与中间件装配

## Proto / OpenAPI / 前端生成

- 业务 proto 位于 `app/servora/service/api/protos/`
- TypeScript HTTP 客户端输出到 `app/servora/service/web/src/service/gen/`
- OpenAPI 模板位于 `app/servora/service/api/buf.openapi.gen.yaml`

## 常用命令

```bash
make gen
make run
make wire
make gen.ent
make gen.gorm
make openapi
cd web && bun install && bun dev
cd web && bun test:unit
cd web && bun test:e2e
cd web && bun lint
```

## 当前实现事实

- 数据层 `internal/data/data.go` 使用 Ent client 作为运行时 ORM
- `internal/data/gorm/` 目录仅保留 GORM GEN 生成物与生成器配置
- 认证中间件文件名为 `internal/server/middleware/authN_jwt.go`
- `server.ProviderSet` 组合了 `middleware.ProviderSet`、`registry.NewRegistrar`、`telemetry.NewMetrics`

## 维护提示

- 旧文档里的根目录 `web/`、`deployment/`、`config.go`、`AuthJWT.go` 等路径名已经不准确
- 修改 proto 后用根目录或服务目录的 `make gen`，修改 Wire 依赖图后执行 `make wire`
- 不要手动编辑 `openapi.yaml`、`web/src/service/gen/`、`wire_gen.go`
