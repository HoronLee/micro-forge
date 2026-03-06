## 新增需求

### 需求:每个服务必须有独立的 go.mod

每个服务（servora, sayhello）必须在其目录下创建独立的 `go.mod` 文件，独立管理服务特定的依赖。

#### 场景:servora 服务模块

- **当** 创建 servora 服务模块
- **那么** 必须在 `app/servora/service/go.mod` 创建模块文件
- **那么** 模块路径必须为 `module github.com/horonlee/servora/app/servora/service`
- **那么** 必须包含服务特定依赖（entgo.io/ent, gorm.io/gorm, 数据库驱动等）

#### 场景:sayhello 服务模块

- **当** 创建 sayhello 服务模块
- **那么** 必须在 `app/sayhello/service/go.mod` 创建模块文件
- **那么** 模块路径必须为 `module github.com/horonlee/servora/app/sayhello/service`

#### 场景:服务模块引用框架和生成代码

- **当** 服务 `go.mod` 需要引用框架或生成代码
- **那么** 必须在 `require` 中声明依赖：
  - `github.com/horonlee/servora/api/gen v0.0.0`
  - `github.com/horonlee/servora v0.0.0`
- **那么** 在有 `go.work` 的情况下，不需要 `replace` 指令（workspace 自动解析）

### 需求:根 go.mod 必须精简

根模块的 `go.mod` 必须移除服务特定依赖，只保留框架级依赖。

#### 场景:移除服务特定依赖

- **当** 精简根 `go.mod`
- **那么** 必须移除以下依赖：
  - `entgo.io/ent`
  - `gorm.io/gorm`
  - `gorm.io/driver/postgres`
  - `github.com/lib/pq`
  - 其他仅服务使用的依赖

#### 场景:保留框架依赖

- **当** 精简根 `go.mod`
- **那么** 必须保留以下依赖：
  - `github.com/go-kratos/kratos/v2`
  - `github.com/redis/go-redis/v9`
  - `github.com/spf13/cobra`
  - 其他 pkg/ 和 cmd/svr/ 使用的依赖

### 需求:验证模块构建

模块拆分完成后必须验证所有服务能够正常构建。

#### 场景:验证服务构建

- **当** 执行 `make build`
- **那么** 所有服务（servora, sayhello）必须能够正常构建
- **那么** 不得出现模块依赖解析错误

#### 场景:验证 workspace 依赖解析

- **当** 在 `go.work` 环境中构建
- **那么** 服务模块必须能够自动解析到本地的 `api/gen` 和根模块
- **那么** 不需要手动配置 `replace` 指令
