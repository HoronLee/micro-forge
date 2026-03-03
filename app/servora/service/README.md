# servora Service

servora 是本项目的主示例微服务，展示了基于 Kratos 框架的完整微服务实现。

## 技术选型

- **ORM**: Ent + GORM GEN（双 ORM 并行支持）
- **缓存**: Redis
- **认证**: JWT
- **API**: gRPC + HTTP 双协议

## 目录结构

```
.
├── cmd/
│   └── server/          # 服务启动入口
├── configs/             # 配置文件
├── internal/
│   ├── biz/            # 业务逻辑层
│   ├── data/           # 数据访问层
│   │   ├── ent/        # Ent 生成代码
│   │   ├── schema/     # Ent Schema 定义
│   │   ├── dao/        # GORM GEN 生成的 DAO
│   │   └── po/         # GORM GEN 生成的 PO
│   ├── server/         # gRPC/HTTP 服务器配置
│   └── service/        # Service 层实现
└── Makefile
```

## 开发命令

```shell
# 生成 GORM GEN 的 PO 和 DAO 代码
make gen.gorm

# 生成 Ent 代码
make gen.ent

# 生成 wire 依赖注入代码
make wire

# 运行服务
make run

# 构建服务
make build
```

## ORM 使用说明

- 默认运行时 ORM 为 **Ent**：在 `internal/data/schema/` 定义 Schema，执行 `make gen.ent` 生成到 `internal/data/ent/`。
- **GORM GEN** 作为并行工具链保留：执行 `make gen.gorm` 生成 `internal/data/gorm/po/` 与 `internal/data/gorm/dao/`。
- 推荐日常使用 `make gen`，会统一执行 `wire + protobuf + openapi + ent` 生成流程。

## 配置

复制示例配置并修改：

```shell
cp configs/config-example.yaml configs/config.yaml
```

主要配置项：
- 数据库连接（MySQL/PostgreSQL/SQLite）
- Redis 连接
- JWT 密钥
- 服务端口
