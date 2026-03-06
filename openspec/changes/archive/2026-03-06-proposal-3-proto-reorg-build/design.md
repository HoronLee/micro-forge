## 上下文

当前所有 proto 文件集中在 `api/protos/` 目录，框架公共 proto（conf, pagination）和业务 proto（auth, user, servora, sayhello）混在一起。这导致：
1. 业务 API 变更需要在框架仓库操作
2. 服务无法独立管理自己的 API 定义
3. 框架仓库包含业务代码，不够纯净

在完成 Buf v2 迁移（提案 1）和 Go 模块拆分（提案 2）后，现在可以将业务 proto 移到各服务目录，实现 proto 定义跟随服务。同时需要更新构建系统（Makefile, app.mk）以适应新的目录结构。

## 目标 / 非目标

**目标：**
- 将业务 proto 移到服务目录（auth, user, test, servora → servora/service/api/protos/；sayhello → sayhello/service/api/protos/）
- 框架保留公共 proto（conf, pagination, template, k8s）
- 更新根 `buf.yaml` 的 `modules` 路径，聚合服务 proto
- 更新构建系统（Makefile, app.mk）
- 更新 `.gitignore`，确保 `api/gen/go.mod` 不被忽略
- 保持所有 import 路径不变

**非目标：**
- 不修改 proto 文件内容（只移动位置）
- 不修改生成代码的 import 路径
- 不拆分 Git 仓库（在 Phase 2 处理）

## 决策

### 决策 1：Proto 组织策略 - 集中 vs 分散

**选择：Proto 跟随服务**

**理由：**
- 业务 proto 是服务的 API 定义，应该跟随服务
- 服务独立管理自己的 API，便于版本控制和独立发布
- 框架仓库更纯净，只保留公共 proto
- 为未来的 Git Submodule 拆分做准备

**Proto 分配：**
- **框架保留**（`api/protos/`）：
  - `conf/v1/` - 配置定义（所有服务共用）
  - `pagination/v1/` - 分页公共类型
  - `template/service/v1/` - svr new api 脚手架模板
  - `k8s/` - K8s 相关定义

- **servora 服务**（`app/servora/service/api/protos/`）：
  - `auth/service/v1/` - 认证服务 API
  - `user/service/v1/` - 用户服务 API
  - `test/service/v1/` - 测试服务 API
  - `servora/service/v1/` - servora 主服务 API

- **sayhello 服务**（`app/sayhello/service/api/protos/`）：
  - `sayhello/service/v1/` - sayhello 示例服务 API

**替代方案：**
- 方案 A：所有 proto 保持在 `api/protos/`
  - 缺点：服务无法独立管理 API
  - 缺点：框架仓库包含业务代码

### 决策 2：Proto 跨引用解析

**选择：通过 Buf v2 workspace 自动解析**

**理由：**
- Buf v2 workspace 的 `modules` 列表聚合所有 proto 源
- 跨 module 的 proto 引用自动解析
- 无需额外配置

**示例：**
```
servora/service/v1/i_auth.proto (在 app/servora/service/api/protos/)
  ↓ import "auth/service/v1/auth.proto"
  ↓
auth/service/v1/auth.proto (在 app/servora/service/api/protos/)
  ↑
  Buf v2 workspace 自动解析
```

### 决策 3：构建系统更新策略

**选择：最小化变更，保持开发者体验一致**

**变更点：**
1. **根 Makefile**：
   - buf 命令在根目录执行（配置已在根目录）
   - 删除 `cd $(API_DIR)` 前缀

2. **app.mk**：
   - 服务级 `make api` 调用根目录的 `make api-go`
   - 路径更新：`cd ../../.. && $(MAKE) api-go`

3. **.gitignore**：
   - 移除 `go.work` 忽略规则（第 19 行）
   - 确保 `api/gen/go.mod` 不被忽略

**开发者体验：**
- `make gen` 命令保持不变
- `make build` 命令保持不变
- 对开发者透明

## 风险 / 权衡

### 风险 1：Proto 文件路径变更影响 IDE 跳转

**风险：** Proto 文件从 `api/protos/` 移到 `app/*/service/api/protos/`，影响 IDE 的文件跳转和引用查找

**缓解措施：**
- 更新文档（README, AGENTS.md）说明新的 proto 位置
- IDE 的 proto 引用通过 import 语句解析，不受文件位置影响

### 风险 2：Proto 跨引用解析失败

**风险：** servora 引用 auth/user/test 的跨引用可能解析失败

**缓解措施：**
- Buf v2 workspace 通过 `modules` 列表聚合所有 proto 源
- 验证：`make gen` 后检查生成代码是否正确
- 测试：servora 服务能否正常构建

### 风险 3：构建流程变更影响 CI/CD

**风险：** Makefile 变更可能影响 CI/CD 脚本

**缓解措施：**
- `make gen` 和 `make build` 命令保持不变
- CI/CD 脚本无需修改
- 文档中说明内部实现变更

## 迁移计划

### 阶段 1：Buf v2 迁移（提案 1）✓
- 完成 Buf v2 workspace 配置
- 为 proto 文件添加 `option go_package`

### 阶段 2：Go 模块拆分（提案 2）✓
- 创建独立 Go 模块
- 创建 `go.work`

### 阶段 3：Proto 重组（提案 3）
1. 创建服务 proto 目录
2. 移动业务 proto 文件
3. 更新根 `buf.yaml` 的 `modules` 路径
4. 更新 Makefile 和 app.mk
5. 更新 `.gitignore`
6. 验证：`make gen` 和 `make build`

### 回滚策略
如果迁移失败：
1. 将 proto 文件移回 `api/protos/`
2. 恢复根 `buf.yaml` 的 `modules` 配置
3. 恢复 Makefile 和 app.mk
4. 执行 `make gen` 重新生成代码

## 开放问题

无
