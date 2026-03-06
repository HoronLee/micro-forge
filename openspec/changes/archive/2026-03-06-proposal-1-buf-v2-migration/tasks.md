## 1. 创建 Buf v2 Workspace 配置

- [x] 1.1 在根目录创建 `buf.yaml` (v2 workspace)，配置 `modules` 列表包含 `api/protos`
- [x] 1.2 配置 `deps` 列表，包含 googleapis, kratos/apis, protovalidate, gnostic
- [x] 1.3 验证 `buf.yaml` 格式正确（`buf lint`）

## 2. 移动 Buf 生成配置到根目录

- [x] 2.1 移动 `api/buf.go.gen.yaml` 到根目录 `buf.go.gen.yaml`
- [x] 2.2 移动 `api/buf.typescript.gen.yaml` 到根目录 `buf.typescript.gen.yaml`
- [x] 2.3 移动 `api/buf.servora.openapi.gen.yaml` 到根目录 `buf.servora.openapi.gen.yaml`
- [x] 2.4 移动 `api/buf.sayhello.openapi.gen.yaml` 到根目录 `buf.sayhello.openapi.gen.yaml`
- [x] 2.5 删除 `api/buf.work.yaml`

## 3. 更新 buf.go.gen.yaml 配置

- [x] 3.1 删除 `managed.go_package_prefix` 配置
- [x] 3.2 删除所有 `managed.override` 配置
- [x] 3.3 保留 `managed.enabled: true` 和 `managed.disable` 列表
- [x] 3.4 更新所有插件的 `out` 路径从 `gen/go` 到 `api/gen/go`

## 4. 为 Proto 文件添加 option go_package

- [x] 4.1 为 `api/protos/auth/service/v1/auth.proto` 添加 `option go_package = "github.com/horonlee/servora/api/gen/go/auth/service/v1;authpb";`
- [x] 4.2 为 `api/protos/user/service/v1/user.proto` 添加 `option go_package = "github.com/horonlee/servora/api/gen/go/user/service/v1;userpb";`
- [x] 4.3 为 `api/protos/test/service/v1/test.proto` 添加 `option go_package = "github.com/horonlee/servora/api/gen/go/test/service/v1;testpb";`
- [x] 4.4 为 `api/protos/servora/service/v1/i_auth.proto` 添加 `option go_package = "github.com/horonlee/servora/api/gen/go/servora/service/v1;servorapb";`
- [x] 4.5 为 `api/protos/servora/service/v1/i_user.proto` 添加 `option go_package = "github.com/horonlee/servora/api/gen/go/servora/service/v1;servorapb";`
- [x] 4.6 为 `api/protos/servora/service/v1/i_test.proto` 添加 `option go_package = "github.com/horonlee/servora/api/gen/go/servora/service/v1;servorapb";`
- [x] 4.7 为 `api/protos/servora/service/v1/servora_doc.proto` 添加 `option go_package = "github.com/horonlee/servora/api/gen/go/servora/service/v1;servorapb";`
- [x] 4.8 为 `api/protos/sayhello/service/v1/sayhello.proto` 添加 `option go_package = "github.com/horonlee/servora/api/gen/go/sayhello/service/v1;sayhellopb";`
- [x] 4.9 为 `api/protos/sayhello/service/v1/sayhello_doc.proto` 添加 `option go_package = "github.com/horonlee/servora/api/gen/go/sayhello/service/v1;sayhellopb";`
- [x] 4.10 为 `api/protos/pagination/v1/pagination.proto` 添加 `option go_package = "github.com/horonlee/servora/api/gen/go/pagination/v1;paginationpb";`

## 5. 验证生成代码

- [x] 5.1 执行 `make gen` 生成代码
- [x] 5.2 验证生成的文件位于 `api/gen/go/<path>/` 目录
- [x] 5.3 验证生成的 Go 文件的 package 声明与 `option go_package` 中的别名一致
- [x] 5.4 执行 `make build` 验证所有服务能够正常构建
- [x] 5.5 验证不存在 import 路径错误
