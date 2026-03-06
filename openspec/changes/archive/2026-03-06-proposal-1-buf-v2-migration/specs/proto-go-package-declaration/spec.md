## 新增需求

### 需求:所有 proto 文件必须显式声明 go_package

每个 proto 文件必须包含 `option go_package` 声明，明确指定生成的 Go 代码的 package 路径和别名。

#### 场景:添加 go_package 声明

- **当** proto 文件缺少 `option go_package` 声明
- **那么** 必须添加格式为 `option go_package = "github.com/horonlee/servora/api/gen/go/<path>;<alias>";` 的声明

#### 场景:go_package 路径格式

- **当** 为 proto 文件添加 `option go_package`
- **那么** 路径必须遵循格式：`github.com/horonlee/servora/api/gen/go/<proto_path>;<package_alias>`
- **那么** `<proto_path>` 必须与 proto 文件的目录结构对应（如 `auth/service/v1`）
- **那么** `<package_alias>` 必须使用服务名 + `pb` 后缀（如 `authpb`, `userpb`, `servorapb`）

#### 场景:需要添加 go_package 的文件列表

- **当** 执行迁移
- **那么** 必须为以下 proto 文件添加 `option go_package`：
  - `api/protos/auth/service/v1/auth.proto` → `authpb`
  - `api/protos/user/service/v1/user.proto` → `userpb`
  - `api/protos/test/service/v1/test.proto` → `testpb`
  - `api/protos/servora/service/v1/i_auth.proto` → `servorapb`
  - `api/protos/servora/service/v1/i_user.proto` → `servorapb`
  - `api/protos/servora/service/v1/i_test.proto` → `servorapb`
  - `api/protos/servora/service/v1/servora_doc.proto` → `servorapb`
  - `api/protos/sayhello/service/v1/sayhello.proto` → `sayhellopb`
  - `api/protos/sayhello/service/v1/sayhello_doc.proto` → `sayhellopb`
  - `api/protos/pagination/v1/pagination.proto` → `paginationpb`

### 需求:buf.go.gen.yaml 必须简化配置

`buf.go.gen.yaml` 必须删除 `go_package_prefix` 和所有 `override` 配置，依赖 proto 文件中的 `option go_package` 声明。

#### 场景:删除 managed mode 配置

- **当** 更新 `buf.go.gen.yaml`
- **那么** 必须删除 `managed.go_package_prefix` 配置
- **那么** 必须删除所有 `managed.override` 配置

#### 场景:保留必要的 managed 配置

- **当** 更新 `buf.go.gen.yaml`
- **那么** 必须保留 `managed.enabled: true`
- **那么** 必须保留 `managed.disable` 列表（禁用外部依赖的 managed mode）

#### 场景:更新输出路径

- **当** `buf.go.gen.yaml` 从 `api/` 移到根目录
- **那么** 所有插件的 `out` 路径必须从 `gen/go` 更新为 `api/gen/go`

### 需求:验证生成代码正确性

迁移完成后必须验证生成的代码路径和 package 声明正确。

#### 场景:验证生成代码路径

- **当** 执行 `make gen` 生成代码
- **那么** 生成的文件必须位于 `api/gen/go/<path>/` 目录
- **那么** 生成的 Go 文件的 package 声明必须与 `option go_package` 中的别名一致

#### 场景:验证服务构建

- **当** 执行 `make build`
- **那么** 所有服务（servora, sayhello）必须能够正常构建
- **那么** 不得出现 import 路径错误
