## 新增需求

### 需求:Makefile 必须更新 buf 命令路径

根目录 Makefile 必须更新，buf 命令在根目录执行（因为 buf 配置已移到根目录）。

#### 场景:删除 cd 前缀

- **当** 更新根 Makefile
- **那么** 必须删除所有 `cd $(API_DIR) &&` 前缀
- **那么** buf 命令直接在根目录执行

#### 场景:api-go 目标更新

- **当** 更新 `api-go` 目标
- **那么** 必须使用 `buf generate --template buf.*.go.gen.yaml` 格式
- **那么** 自动扫描所有 `buf.*.go.gen.yaml` 模板文件

#### 场景:api-ts 目标更新

- **当** 更新 `api-ts` 目标
- **那么** 必须使用 `buf generate --template buf.*.typescript.gen.yaml` 格式
- **那么** 自动扫描所有 TypeScript 生成模板

### 需求:app.mk 必须更新调用路径

`app.mk` 中的服务级 API 生成命令必须调用根目录的 Makefile 目标。

#### 场景:服务级 api 目标

- **当** 在服务目录执行 `make api`
- **那么** 必须调用根目录的 `make api-go`
- **那么** 使用路径：`cd ../../.. && $(MAKE) api-go`

#### 场景:服务级 openapi 目标

- **当** 在服务目录执行 `make openapi`
- **那么** 必须调用根目录的 `make openapi`
- **那么** 使用路径：`cd ../../.. && $(MAKE) openapi`

### 需求:.gitignore 必须更新

`.gitignore` 必须更新，确保 `go.work` 和 `api/gen/go.mod` 不被忽略。

#### 场景:移除 go.work 忽略

- **当** 更新 `.gitignore`
- **那么** 必须删除第 19 行的 `go.work` 忽略规则
- **那么** `go.work` 必须提交到 Git

#### 场景:确保 go.mod 不被忽略

- **当** 更新 `.gitignore`
- **那么** 必须确保 `api/gen/go.mod` 不被 `api/gen/go/` 忽略规则影响
- **那么** 可以添加 `!api/gen/go.mod` 排除规则（如果需要）

### 需求:验证构建流程

构建系统更新后必须验证整个构建流程正常工作。

#### 场景:验证 make gen

- **当** 在根目录执行 `make gen`
- **那么** 必须能够正常生成所有代码（Go, TypeScript, OpenAPI）
- **那么** 不得出现路径错误

#### 场景:验证服务级 make api

- **当** 在服务目录（如 `app/servora/service/`）执行 `make api`
- **那么** 必须能够正常调用根目录的代码生成
- **那么** 生成的代码位于 `api/gen/go/`
