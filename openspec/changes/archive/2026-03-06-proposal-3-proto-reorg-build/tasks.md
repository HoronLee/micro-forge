## 1. 创建服务 proto 目录结构

- [x] 1.1 创建 `app/servora/service/api/protos/` 目录
- [x] 1.2 创建 `app/sayhello/service/api/protos/` 目录

## 2. 移动 servora 服务 proto 文件

- [x] 2.1 移动 `api/protos/auth/` 到 `app/servora/service/api/protos/auth/`
- [x] 2.2 移动 `api/protos/user/` 到 `app/servora/service/api/protos/user/`
- [x] 2.3 移动 `api/protos/test/` 到 `app/servora/service/api/protos/test/`
- [x] 2.4 移动 `api/protos/servora/` 到 `app/servora/service/api/protos/servora/`

## 3. 移动 sayhello 服务 proto 文件

- [x] 3.1 移动 `api/protos/sayhello/` 到 `app/sayhello/service/api/protos/sayhello/`

## 4. 更新根 buf.yaml

- [x] 4.1 在根 `buf.yaml` 的 `modules` 列表中添加 `path: app/servora/service/api/protos`
- [x] 4.2 在根 `buf.yaml` 的 `modules` 列表中添加 `path: app/sayhello/service/api/protos`
- [x] 4.3 验证 `buf.yaml` 格式正确（`buf lint`）- 配置正确，lint 警告为代码风格问题

## 5. 更新根 Makefile

- [x] 5.1 删除所有 `cd $(API_DIR) &&` 前缀（buf-update 目标已更新）
- [x] 5.2 更新 `api-go` 目标，使用 `buf generate --template buf.*.go.gen.yaml` 格式（已正确）
- [x] 5.3 更新 `api-ts` 目标，使用 `buf generate --template buf.*.typescript.gen.yaml` 格式（已正确）
- [x] 5.4 更新 `openapi` 目标，直接调用 `buf generate --template buf.*.openapi.gen.yaml`（已正确）
- [x] 5.5 更新 `clean` 目标，清理路径更新为 `api/gen/go`

## 6. 更新 app.mk

- [x] 6.1 更新服务级 `api` 目标，调用 `cd ../../.. && $(MAKE) api-go`
- [x] 6.2 更新服务级 `openapi` 目标，调用 `cd ../../.. && $(MAKE) openapi`

## 7. 验证 proto 生成和服务构建

- [x] 7.1 执行 `make gen` 验证所有 proto 文件能够正常生成代码
- [x] 7.2 验证生成的代码位于 `api/gen/go/` 目录
- [x] 7.3 验证 proto 跨引用解析正确（servora 引用 auth/user/test）
- [x] 7.4 执行 `make build` 验证所有服务能够正常构建
- [x] 7.5 验证不存在 import 路径错误
