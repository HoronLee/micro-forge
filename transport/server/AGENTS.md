# AGENTS.md - transport/server/

<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-05-21 -->

## 模块定位

`transport/server` 提供服务端 transport 装配：统一 `Server` 接口、gRPC/HTTP server 构造、registry endpoint 解析、accept loop 与标准 server middleware chain。

## 子目录职责

| 目录 | 职责 |
| --- | --- |
| `accept/` | TCP accept 循环辅助 |
| `endpoint/` | 解析 registry endpoint、host、bind addr 与 query |
| `grpc/` | Kratos gRPC server 构造，注册服务，解析 TLS/registry endpoint |
| `http/` | Kratos HTTP server 构造，注册服务、CORS、metrics、health、swagger |
| `middleware/` | 标准 server chain 与 operation whitelist |

顶层 `Server` 接口只聚合 `Start/Stop` 生命周期和 `Endpoint()` 注册地址能力。

## 装配语义

- gRPC/HTTP server 都从 `corev1.Server_*` 读取 listen、timeout、TLS、registry 配置。
- TLS 构造统一调用 `security/tls`，协议子包共享 PEM 解析。
- registry endpoint 解析失败时在启动期 panic，防止注册错误地址。
- HTTP 额外负责 CORS、`/metrics`、`/healthz`、`/readyz` 与 swagger 注册。
- 服务 registrar 在 server 创建后执行。

## Middleware chain

`middleware.NewChainBuilder(l).Build()` 固定顺序：recovery、可选 tracing、logging、默认 ratelimit、proto validate、可选 metrics。

`middleware.NewChainBuilder(l).Build()` 返回 `[]middleware.Middleware`；调用方使用 `append(ms, additionalMiddleware...)` 追加中间件。

`whitelist` 匹配 operation 白名单，而非 IP 白名单或网络访问控制。

## 常见反模式

- 在 server 目录写 service handler 或领域逻辑。
- 将 whitelist 当作 IP allowlist。
- 在 HTTP/gRPC 子包分别实现 TLS 或 registry 解析。

## 测试

```bash
go test ./transport/server/...
```

修改 middleware 顺序时，同时检查 audit collector 的挂载位置和附加中间件的短路路径。
