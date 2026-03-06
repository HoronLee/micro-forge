# service-proto-organization 规范

## 目的
待定 - 由归档变更 proposal-3-proto-reorg-build 创建。归档后请更新目的。
## 需求
### 需求:业务 proto 必须移到服务目录

所有业务相关的 proto 文件必须从 `api/protos/` 移动到对应服务的 `proto/` 目录，实现 proto 定义跟随服务。

#### 场景:servora 服务 proto 移动

- **当** 重组 proto 目录结构
- **那么** 必须将以下目录从 `api/protos/` 移动到 `app/servora/service/proto/`：
  - `auth/` → `app/servora/service/proto/auth/`
  - `user/` → `app/servora/service/proto/user/`
  - `test/` → `app/servora/service/proto/test/`
  - `servora/` → `app/servora/service/proto/servora/`

#### 场景:sayhello 服务 proto 移动

- **当** 重组 proto 目录结构
- **那么** 必须将 `api/protos/sayhello/` 移动到 `app/sayhello/service/proto/sayhello/`

#### 场景:框架保留公共 proto

- **当** 重组 proto 目录结构
- **那么** 以下目录必须保留在 `api/protos/`：
  - `conf/v1/` - 配置定义
  - `pagination/v1/` - 分页公共类型
  - `template/service/v1/` - 脚手架模板
  - `k8s/` - K8s 相关定义

### 需求:buf.yaml 必须聚合服务 proto

根目录的 `buf.yaml` 必须更新 `modules` 列表，包含所有服务的 proto 目录。

#### 场景:更新 modules 列表

- **当** 更新根 `buf.yaml`
- **那么** `modules` 列表必须包含：
  - `path: api/protos` (框架公共 proto)
  - `path: app/servora/service/proto` (servora 服务 proto)
  - `path: app/sayhello/service/proto` (sayhello 服务 proto)

#### 场景:proto 跨引用解析

- **当** servora proto 引用 auth/user/test proto（如 `import "auth/service/v1/auth.proto"`）
- **那么** Buf v2 workspace 必须能够自动解析引用
- **那么** 生成代码时不得出现 import 错误

### 需求:验证 proto 生成和服务构建

重组完成后必须验证 proto 能够正常生成代码，服务能够正常构建。

#### 场景:验证 proto 生成

- **当** 执行 `make gen`
- **那么** 所有 proto 文件必须能够正常生成代码到 `api/gen/go/`
- **那么** 不得出现 proto 引用解析错误

#### 场景:验证服务构建

- **当** 执行 `make build`
- **那么** 所有服务（servora, sayhello）必须能够正常构建
- **那么** 不得出现 import 路径错误

