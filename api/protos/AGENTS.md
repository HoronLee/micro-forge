# AGENTS.md - API 定义 (Protobuf)

<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-09 | Updated: 2026-02-09 -->

## 目录概述

此目录包含 servora 项目中所有服务的 Protobuf 定义文件。我们使用 [Buf](https://buf.build/) 作为 Protobuf 构建工具链。

### 服务子目录说明

| 目录 | 说明 | 关键文件 |
| :--- | :--- | :--- |
| `auth/` | 身份验证服务协议定义 | `auth.proto` (gRPC) |
| `user/` | 用户管理服务协议定义 | `user.proto` (gRPC) |
| `servora/` | 主服务入口，主要包含 HTTP API 定义 | `i_*.proto` (HTTP) |
| `sayhello/` | 独立微服务示例协议定义 | `sayhello.proto` |
| `test/` | 测试与演示服务协议定义 | `test.proto` (gRPC) |
| `conf/` | 服务内部配置结构定义 | `conf.proto` |
| `template/` | `svr new api` 脚手架模板（可自定义覆盖） | `template.proto`, `template_doc.proto` |

## Proto 文件编写规范

为了保持代码生成的一致性和 API 的清晰度，请遵循以下规范：

### 1. 协议分类与命名
- **HTTP API (BFF/Gateway)**: 文件名必须以 `i_` 开头（如 `i_auth.proto`），生成 HTTP 路由代码。统一使用包名 `servora.service.v1`。
- **gRPC 服务**: 位于 `{service}/service/v1/` 目录下，文件名为 `{service}.proto`。每个服务拥有独立的 package 空间。
- **配置定义**: 位于 `conf/v1/`，用于定义 `config.yaml` 映射的 Go 结构体。

### 2. 核心原则
- **按域划分**: Proto 文件应按业务域（Domain）组织，而不是按数据库表组织。
- **双协议支持**: Kratos 支持同时开启 gRPC 和 HTTP。HTTP 接口通过 `google.api.http` 注解实现。
- **错误定义**: 使用 `errors.proto` 定义业务错误码，以便生成可复用的 Go 错误处理代码。

### 3. 代码生成流程
修改任何 `.proto` 文件后，必须在项目根目录执行：
```bash
make gen
```
这会触发 `buf generate` 并更新 `api/gen/go/` 下的生成代码。

## 常用工具配置
- **buf.yaml**: 维护 BSR (Buf Schema Registry) 依赖。
- **buf.lock**: 锁定依赖版本。
- **buf.gen.yaml**: (位于父目录 `api/`) 定义生成插件（go, go-grpc, go-http, go-errors, openapi）。

## 脚手架快速创建

使用 `svr new api` 可以一键生成符合 servora proto 规范的骨架文件：

```bash
# 在项目根目录执行
svr new api user
svr new api say_hello
svr new api billing.invoice   # 生成嵌套目录 billing/invoice/service/v1/
```

生成规则：
- 目录：`api/protos/<name>/service/v1/`（点分段映射为多级目录）
- 文件：`<name>.proto` + `<name>_doc.proto`（点分段以 `_` 拼接为文件名）
- package：`<name>.service.v1`（保留点分，与目录结构对应）

**自定义模板**：将 `template/service/v1/template.proto` 和 `template_doc.proto` 修改为项目偏好的样板，`svr new api` 会直接使用此目录作为模板源。也可通过 `--template` 标志指定任意目录。

## 注意事项
- **不可手动修改生成代码**: `api/gen/go/` 下的所有内容都是自动生成的。
- **版本控制**: 遵循 `v1`, `v2` 等版本化路径，确保 API 的向后兼容性。
- **文档**: 使用 `*_doc.proto` 或在 message/service 上方编写详细注释，以便自动生成高质量的 OpenAPI 文档。
- **模板目录**: `template/service/v1/` 是脚手架模板，不参与 buf 正式代码生成，不应被引用为业务 proto 依赖。
