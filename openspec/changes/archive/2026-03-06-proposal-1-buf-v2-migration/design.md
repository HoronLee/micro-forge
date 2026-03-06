## 上下文

当前项目使用 Buf v1 配置，proto 文件集中在 `api/protos/`，通过 `buf.go.gen.yaml` 的 `go_package_prefix` 和 `override` 管理生成路径。随着项目演进，需要支持 proto 文件分散到各服务目录，但 Buf v1 workspace 无法优雅聚合多个分散的 proto 源。同时，`buf generate` 的 `clean: true` 会删除 `api/gen/go/` 目录内容，导致无法在其中放置 `go.mod` 文件。

迁移到 Buf v2 workspace 模式可以：
1. 通过 v2 workspace 的 `modules` 列表聚合多个 proto 源目录
2. 将 `go.mod` 放在 `api/gen/` 层级，避免被 `clean: true` 删除
3. 通过 proto 文件中的 `option go_package` 显式声明生成路径，简化配置

## 目标 / 非目标

**目标：**
- 迁移到 Buf v2 workspace 配置，支持聚合多个 proto 源目录
- 所有 proto 文件显式声明 `option go_package`
- 简化 `buf.go.gen.yaml`，删除 `go_package_prefix` 和 `override`
- 保持所有 import 路径不变（`github.com/horonlee/servora/api/gen/go/...`）
- 为后续 proto 目录重组和 Go 模块拆分奠定基础

**非目标：**
- 不移动 proto 文件位置（在提案 3 中处理）
- 不创建 Go 模块（在提案 2 中处理）
- 不修改生成代码的 import 路径

## 决策

### 决策 1：Buf 配置位置 - 根目录 vs api/ 目录

**选择：根目录**

**理由：**
- Proto 文件将分散在 `api/protos/` 和 `app/*/service/proto/`（提案 3）
- Buf v2 workspace 的 `modules` 路径是相对于 `buf.yaml` 的
- 如果 `buf.yaml` 在 `api/`，引用 `app/servora/service/proto/` 需要写 `../app/...`（不优雅）
- 根目录可以用清晰的相对路径：`api/protos`, `app/servora/service/proto`
- 符合 monorepo 的配置管理最佳实践（配置在根目录）

**替代方案：**
- 方案 A：`buf.yaml` 在 `api/` 目录
  - 缺点：引用服务 proto 需要 `../app/...`，路径不清晰
  - 缺点：不符合 monorepo 惯例

### 决策 2：go_package 管理方式 - Buf managed mode vs 显式声明

**选择：显式声明（在每个 proto 文件中添加 `option go_package`）**

**理由：**
- Buf managed mode 的 `go_package_prefix` + `override` 配置复杂，难以维护
- 显式声明更清晰，每个 proto 文件自包含生成路径信息
- 便于理解和调试（不需要查看 buf 配置就知道生成路径）
- 符合 protobuf 最佳实践

**替代方案：**
- 方案 A：继续使用 Buf managed mode
  - 缺点：配置复杂，需要为每个目录添加 `override`
  - 缺点：proto 文件不自包含，需要查看 buf 配置才知道生成路径

### 决策 3：Buf v2 workspace 结构

**选择：单一 `buf.yaml` 聚合所有 proto 源**

**理由：**
- Buf v2 workspace 通过 `modules` 列表聚合多个 proto 源目录
- 所有 proto 在同一 workspace 中，跨引用自动解析
- 简化配置，只需一个 `buf.yaml`

**配置示例：**
```yaml
version: v2
modules:
  - path: api/protos
  - path: app/servora/service/proto  # 提案 3 添加
  - path: app/sayhello/service/proto # 提案 3 添加
deps:
  - buf.build/googleapis/googleapis
  - buf.build/kratos/apis
  - buf.build/bufbuild/protovalidate
  - buf.build/gnostic/gnostic
```

### 决策 4：生成配置路径更新

**选择：更新 `out` 路径从 `gen/go` 到 `api/gen/go`**

**理由：**
- buf 配置从 `api/` 移到根目录
- 相对路径需要更新以保持生成位置不变
- `out: api/gen/go` 比 `out: gen/go` 更清晰（绝对路径）

## 风险 / 权衡

### 风险 1：Proto 文件批量修改

**风险：** 需要为所有 proto 文件添加 `option go_package`，涉及多个文件修改

**缓解措施：**
- 按照设计文档中的列表逐个添加
- 使用统一的格式：`option go_package = "github.com/horonlee/servora/api/gen/go/<path>;<alias>";`
- 验证：`make gen` 后检查生成代码路径是否正确

### 风险 2：Buf 配置路径变更影响开发者习惯

**风险：** 开发者习惯在 `api/` 目录执行 buf 命令，配置移到根目录后可能不适应

**缓解措施：**
- 更新 Makefile，`make gen` 命令在根目录执行
- 更新文档（README, AGENTS.md）说明新的配置位置
- CI/CD 脚本自动更新

### 风险 3：Import 路径兼容性

**风险：** 生成代码的 import 路径可能发生变化

**缓解措施：**
- 通过 `option go_package` 显式指定路径，确保与现有路径一致
- 验证：`make gen` 后检查生成代码的 package 声明
- 验证：`make build` 确保所有服务能正常构建

## 迁移计划

### 阶段 1：准备（提案 1）
1. 创建根目录 `buf.yaml` (v2 workspace)
2. 移动 `buf.*.gen.yaml` 到根目录
3. 为所有 proto 文件添加 `option go_package`
4. 删除 `api/buf.work.yaml`
5. 验证：`make gen` 和 `make build`

### 阶段 2：Go 模块拆分（提案 2）
- 在 `api/gen/` 创建 `go.mod`
- 创建服务 `go.mod`
- 创建 `go.work`

### 阶段 3：Proto 重组（提案 3）
- 移动业务 proto 到服务目录
- 更新 `buf.yaml` 的 `modules` 路径

### 回滚策略
如果迁移失败：
1. 恢复 `api/buf.work.yaml` 和 `api/buf.go.gen.yaml`
2. 删除根目录的 buf 配置
3. 移除 proto 文件中的 `option go_package`
4. 执行 `make gen` 恢复生成代码

## 开放问题

无
