## 上下文

当前项目使用单一 `go.mod` 管理所有依赖，框架代码（`pkg/`, `cmd/svr/`）和业务服务（servora, sayhello）的依赖混在一起。同时，`api/gen/go/` 目录因为 `buf generate` 的 `clean: true` 无法放置 `go.mod`，导致生成代码无法作为独立模块。

采用 Go workspace 模式可以：
1. 将项目拆分为多个独立模块（根模块、api/gen、服务模块）
2. 通过 `go.work` 在本地开发时自动解析模块依赖
3. 为未来的 Git Submodule 拆分和独立 CI/CD 奠定基础
4. 保持所有 import 路径不变

关键设计决策：
- `go.mod` 放在 `api/gen/` 层级（而不是 `api/gen/go/`），避免被 `buf generate` 的 `clean: true` 删除
- 每个服务独立 `go.mod`，独立管理依赖
- 根 `go.mod` 精简，只保留框架依赖

## 目标 / 非目标

**目标：**
- 创建 4 个独立 Go 模块（根模块、api/gen、servora、sayhello）
- 创建 `go.work` 聚合所有模块，支持本地开发
- 精简根 `go.mod`，移除服务特定依赖
- 保持所有 import 路径不变
- 为未来的 Git Submodule 拆分奠定基础

**非目标：**
- 不拆分 Git 仓库（在 Phase 2 处理）
- 不修改生成代码的 import 路径
- 不改变服务的构建流程（对开发者透明）

## 决策

### 决策 1：go.mod 放置位置 - api/gen/ vs api/gen/go/

**选择：api/gen/go.mod**

**理由：**
- `buf generate` 的 `clean: true` 会删除 `api/gen/go/` 目录内容
- 如果 `go.mod` 在 `api/gen/go/`，每次生成都会被删除
- 放在 `api/gen/` 层级，`clean: true` 只清理 `go/` 子目录，`go.mod` 安全
- Import 路径完全不变：`github.com/horonlee/servora/api/gen/go/...` 仍然有效（Go 模块解析 = module path + 相对路径）

**替代方案：**
- 方案 A：`go.mod` 在 `api/gen/go/`
  - 缺点：会被 `buf generate` 删除
  - 缺点：需要关闭 `clean: true`，可能残留过期文件
- 方案 B：关闭 `clean: true`
  - 缺点：可能残留过期的生成文件，导致构建问题

### 决策 2：模块划分策略

**选择：4 个独立模块**

**模块列表：**
1. `github.com/horonlee/servora` - 根模块（pkg/, cmd/svr/）
2. `github.com/horonlee/servora/api/gen` - 生成代码模块
3. `github.com/horonlee/servora/app/servora/service` - servora 服务
4. `github.com/horonlee/servora/app/sayhello/service` - sayhello 服务

**理由：**
- 根模块：框架代码，可独立发布
- api/gen：生成代码，所有服务共享
- 服务模块：每个服务独立管理依赖，为未来独立仓库做准备

**替代方案：**
- 方案 A：只拆分 api/gen，服务仍在根模块
  - 缺点：服务无法独立管理依赖
  - 缺点：无法为未来的 Git Submodule 拆分做准备

### 决策 3：go.work 配置

**选择：提交 go.work 到 Git**

**理由：**
- 确保所有开发者和 CI 使用相同的 workspace 配置
- 避免本地开发和 CI 环境不一致
- Go 官方推荐做法（monorepo 场景）

**go.work 内容：**
```go
go 1.26.0
use (
    .                          // 根模块
    ./api/gen                  // 生成代码模块
    ./app/servora/service      // servora 服务
    ./app/sayhello/service     // sayhello 服务
)
```

**替代方案：**
- 方案 A：不提交 go.work，每个开发者自己创建
  - 缺点：容易出现配置不一致
  - 缺点：CI 需要额外配置

### 决策 4：根 go.mod 精简策略

**选择：只保留 pkg/ 和 cmd/svr/ 的依赖**

**移除的依赖：**
- `entgo.io/ent` - 只有 servora 使用
- `gorm.io/gorm` - 只有 servora 使用
- `gorm.io/driver/postgres` - 只有 servora 使用
- `github.com/lib/pq` - 只有 servora 使用
- 其他服务特定依赖

**保留的依赖：**
- `github.com/go-kratos/kratos/v2` - 框架核心
- `github.com/redis/go-redis/v9` - pkg/ 使用
- `github.com/spf13/cobra` - cmd/svr/ 使用
- 其他框架级依赖

## 风险 / 权衡

### 风险 1：依赖管理复杂度增加

**风险：** 从单一 `go.mod` 变为 4 个模块，依赖管理复杂度增加

**缓解措施：**
- `go.work` 自动解析本地依赖，开发者无感知
- 每个模块独立 `go mod tidy`，保持依赖清晰
- CI 中先执行 `make gen` 生成 `api/gen/go/` 代码

### 风险 2：CI/CD 需要调整

**风险：** CI 需要先生成代码，再构建服务

**缓解措施：**
- 更新 CI 配置，在构建前执行 `make gen`
- 文档中明确说明 CI 流程变更

### 风险 3：go.mod 位置非常规

**风险：** `go.mod` 在 `api/gen/` 而不是 `api/gen/go/`，稍显非常规

**缓解措施：**
- 这是合法的 Go 模块结构
- Import 路径完全不变，对使用者透明
- 文档中说明设计原因

## 迁移计划

### 阶段 1：Buf v2 迁移（提案 1）
- 完成 Buf v2 workspace 配置
- 为 proto 文件添加 `option go_package`

### 阶段 2：Go 模块拆分（提案 2）
1. 创建 `api/gen/go.mod`
2. 创建服务 `go.mod`（servora, sayhello）
3. 精简根 `go.mod`
4. 创建 `go.work`
5. 更新 `.gitignore`
6. 验证：`make build` 和 `make test`

### 阶段 3：Proto 重组（提案 3）
- 移动业务 proto 到服务目录
- 更新构建系统

### 回滚策略
如果迁移失败：
1. 删除所有新创建的 `go.mod` 和 `go.work`
2. 恢复根 `go.mod` 的完整依赖
3. 执行 `go mod tidy`
4. 验证构建

## 开放问题

无
