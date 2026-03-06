# AGENTS.md - servora internal 实现层

<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-09 | Updated: 2026-03-06 -->

## 目录定位

`internal/` 是 `servora` 服务的核心实现层，按 `service -> biz -> data` 分层组织，并由 `server/` 负责入口装配。

## 当前结构

```text
internal/
├── biz/
│   ├── biz.go
│   ├── auth.go
│   ├── user.go
│   ├── test.go
│   └── entity/
├── consts/
├── data/
│   ├── data.go
│   ├── auth.go
│   ├── user.go
│   ├── test.go
│   ├── generate.go
│   ├── schema/
│   ├── ent/
│   └── gorm/
├── server/
│   ├── server.go
│   ├── grpc.go
│   ├── http.go
│   └── middleware/
└── service/
    ├── service.go
    ├── auth.go
    ├── user.go
    └── test.go
```

## 当前实现事实

- `biz.ProviderSet` 目前只注册 `NewAuthUsecase`、`NewUserUsecase`、`NewTestUsecase`
- 领域实体已放到 `biz/entity/`，不是旧文档里的单文件 `entity.go`
- `data.ProviderSet` 当前使用 `registry.NewDiscovery`、`NewDBClient`、`NewRedis`、`NewData`
- 运行时数据库访问默认走 Ent；GORM GEN 主要用于生成并保留框架能力
- 认证中间件文件为 `server/middleware/authN_jwt.go`

## 分层规则

- `service/`：仅做协议适配与参数转换
- `biz/`：业务规则、用例编排、仓储接口
- `data/`：Ent / Redis / transport client / 服务发现的具体实现
- `server/`：HTTP、gRPC、中间件、注册、指标装配

## 常用命令

在 `app/servora/service/` 目录执行：

```bash
make wire
make gen.ent
make gen.gorm
go test ./internal/biz/...
go test ./internal/data/...
```

## 维护提示

- 旧文档里的 `discovery.go`、`registry.go`、`metrics.go`、`AuthJWT.go` 等单文件描述已不再准确
- 若新增业务模块，通常需要同时修改 `biz/`、`data/`、`service/`，并视情况补充 `server/` 注册
- 修改 ProviderSet 后必须重新执行 `make wire`
