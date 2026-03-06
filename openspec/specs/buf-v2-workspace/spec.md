# buf-v2-workspace 规范

## 目的
待定 - 由归档变更 proposal-1-buf-v2-migration 创建。归档后请更新目的。
## 需求
### 需求:Buf v2 workspace 必须聚合所有 proto 源目录

系统必须使用 Buf v2 workspace 配置，通过 `modules` 列表聚合框架和服务的所有 proto 源目录，确保跨目录的 proto 引用能够正确解析。

#### 场景:聚合多个 proto 源目录

- **当** 在根目录创建 `buf.yaml` 并配置 `modules` 列表包含 `api/protos`
- **那么** buf 命令能够识别并处理 `api/protos/` 下的所有 proto 文件

#### 场景:proto 跨引用解析

- **当** proto 文件引用其他目录的 proto（如 `import "pagination/v1/pagination.proto"`）
- **那么** buf 必须能够自动解析引用，无需额外配置

#### 场景:外部依赖解析

- **当** proto 文件引用外部依赖（如 `import "google/protobuf/timestamp.proto"`）
- **那么** buf 必须通过 `deps` 列表正确解析 BSR 依赖（googleapis, kratos/apis, protovalidate, gnostic）

### 需求:Buf 配置文件必须位于根目录

所有 buf 配置文件（`buf.yaml`, `buf.*.gen.yaml`）必须位于项目根目录，以便统一管理和清晰的路径引用。

#### 场景:buf.yaml 在根目录

- **当** 在根目录创建 `buf.yaml`
- **那么** 可以使用清晰的相对路径引用所有 proto 源（如 `api/protos`, `app/servora/service/proto`）

#### 场景:生成配置在根目录

- **当** `buf.go.gen.yaml` 在根目录
- **那么** 输出路径必须使用 `out: api/gen/go`（而不是 `out: gen/go`）

#### 场景:删除旧配置

- **当** 完成迁移
- **那么** 必须删除 `api/buf.work.yaml`（v1 workspace 配置）

### 需求:生成代码路径必须保持不变

迁移后生成的 Go 代码 import 路径必须与迁移前完全一致，确保现有代码无需修改。

#### 场景:import 路径不变

- **当** 执行 `buf generate` 生成 Go 代码
- **那么** 生成的代码 package 路径必须为 `github.com/horonlee/servora/api/gen/go/<path>`

#### 场景:现有代码无需修改

- **当** 迁移完成后执行 `make build`
- **那么** 所有服务必须能够正常构建，无需修改任何 import 语句

