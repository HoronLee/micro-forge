## 1. 创建 api/gen 模块

- [x] 1.1 在 `api/gen/` 创建 `go.mod`，模块路径为 `github.com/horonlee/servora/api/gen`
- [x] 1.2 添加必要依赖：google.golang.org/protobuf, google.golang.org/grpc, github.com/go-kratos/kratos/v2, github.com/envoyproxy/protoc-gen-validate
- [x] 1.3 执行 `cd api/gen && go mod tidy`

## 2. 创建 servora 服务模块

- [x] 2.1 在 `go/servora/service/` 创建 `go.mod`，模块路径为 `github.com/horonlee/servora/go/servora/service`
- [x] 2.2 添加依赖：github.com/horonlee/servora/api/gen, github.com/horonlee/servora
- [x] 2.3 添加服务特定依赖：entgo.io/ent, gorm.io/gorm, gorm.io/driver/postgres, github.com/lib/pq 等
- [x] 2.4 执行 `cd go/servora/service && go mod tidy`

## 3. 创建 sayhello 服务模块

- [x] 3.1 在 `app/sayhello/service/` 创建 `go.mod`，模块路径为 `github.com/horonlee/servora/app/sayhello/service`
- [x] 3.2 添加依赖：github.com/horonlee/servora/api/gen, github.com/horonlee/servora
- [x] 3.3 执行 `cd app/sayhello/service && go mod tidy`

## 4. 精简根 go.mod

- [x] 4.1 从根 `go.mod` 移除 entgo.io/ent（保留 gorm 相关依赖，因为 pkg/logger 和 cmd/svr 需要）
- [x] 4.2 从根 `go.mod` 移除 gorm.io/gorm（实际保留，框架代码需要）
- [x] 4.3 从根 `go.mod` 移除 gorm.io/driver/postgres（实际保留，框架代码需要）
- [x] 4.4 从根 `go.mod` 移除 github.com/lib/pq（已移除）
- [x] 4.5 从根 `go.mod` 移除其他服务特定依赖（已处理）
- [x] 4.6 执行 `go mod tidy` 清理根模块依赖（已通过 go work sync 处理）

## 5. 创建 go.work

- [x] 5.1 在根目录创建 `go.work`
- [x] 5.2 添加 `use .` (根模块)
- [x] 5.3 添加 `use ./api/gen` (生成代码模块)
- [x] 5.4 添加 `use ./app/servora/service` (servora 服务)
- [x] 5.5 添加 `use ./app/sayhello/service` (sayhello 服务)
- [x] 5.6 执行 `go work sync` 同步 workspace 依赖

## 6. 更新 .gitignore

- [x] 6.1 从 `.gitignore` 删除第 19 行的 `go.work` 忽略规则
- [x] 6.2 确保 `api/gen/go.mod` 不被 `api/gen/go/` 忽略规则影响（无相关忽略规则）

## 7. 验证模块构建

- [x] 7.1 执行 `make build` 验证所有服务能够正常构建（Go 构建成功，TypeScript 问题留待 proposal-3）
- [x] 7.2 验证 workspace 依赖解析正确（服务能自动解析到本地 api/gen 和根模块）
- [x] 7.3 验证不存在模块依赖解析错误
