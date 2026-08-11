# AGENTS.md - transport/

<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-22 | Updated: 2026-05-12 -->

## 模块目的

提供服务间 transport 工具箱，覆盖 client / server 装配、连接管理与通用 middleware 支撑。TLS 配置构造归 `security/tls`。

## 当前结构

```text
transport/
├── client/                   # Dialer（grpc/http/middleware）+ endpoint 索引
└── server/                   # NewServer（grpc/http/middleware）+ endpoint/accept
                              # http/ 含 cors/swagger/health 子包
```

## 当前实现事实

- `client/` 目录承载 `grpc/`、`http/`、`middleware/` 与 `endpoint/`（按 protocol 索引 service endpoint 配置）
- `server/` 目录承载 `grpc/`、`http/`、`middleware/`、`endpoint/`（注册中心 endpoint URL 解析）、`accept/`（TCP accept 循环）
- `server/middleware/whitelist.go` 匹配 operation 白名单，而非 IP 白名单
- TLS 配置构造（`BuildServerTLS` / `BuildClientTLS`）归 `security/tls`；transport 通过 alias `svrtls` 引用

## 边界约束

- transport 包负责连接、协议和中间件装配；service 包负责 handler 与领域规则。
- client/server 子目录分别维护各自的详细装配规则。

## 常见反模式

- 在 transport 目录中写入 service handler。
- 将 operation 白名单当作网络访问控制白名单。
- 把 client/server 共性抽象与单一协议实现强耦合。

## 测试与使用

```bash
go test ./transport/...
go test ./transport/client/...
go test ./transport/server/...
```

## 维护提示

- 新 middleware 遵循 transport 通用接口与现有装配方式。
- 调整 client/server 目录时同步更新父级 `servora/AGENTS.md` 与调用方说明。
