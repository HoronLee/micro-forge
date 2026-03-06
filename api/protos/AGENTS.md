# AGENTS.md - 共享 Proto 模块

<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-09 | Updated: 2026-03-06 -->

## 当前定位

`api/protos/` 现在是 **共享 proto 模块**，不再承载全部服务协议。当前目录真实内容只有：
- `conf/`：配置结构 proto
- `pagination/`：分页相关公共 proto
- `template/`：`svr new api` 使用的默认脚手架模板

服务专属协议已拆到：
- `app/servora/service/api/protos/`
- `app/sayhello/service/api/protos/`

## 当前结构

```text
api/protos/
├── AGENTS.md
├── buf.yaml
├── buf.lock
├── conf/
├── pagination/
└── template/
```

## 目录说明

- `conf/v1/conf.proto`：服务配置映射
- `pagination/`：分页请求/响应的公共定义
- `template/service/v1/template.proto`：`svr new api` 主模板
- `template/service/v1/template_doc.proto`：`svr new api` 文档模板

## 生成与校验

在项目根目录执行：

```bash
make gen
```

只校验共享 proto 模块时：

```bash
cd api/protos && buf lint
cd api/protos && buf format -w
cd api/protos && buf mod update
```

## 脚手架约定

`svr new api <name>` 默认把骨架写到 `api/protos/<name>/service/v1/`：

```bash
svr new api user
svr new api say_hello
svr new api billing.invoice
```

规则：
- 目录：点号分层映射为多级目录
- 文件：点号转下划线，生成 `<name>.proto` 与 `<name>_doc.proto`
- 包名：保留点分形式，如 `billing.invoice.service.v1`

## 维护提示

- 这里不再列出 `auth/`、`user/`、`servora/`、`sayhello/` 等业务目录，那些协议已不在本模块
- `template/` 只是脚手架源，不参与业务依赖设计
- 若要更新 `servora` 或 `sayhello` 的实际接口，请改对应服务目录下的 `api/protos/`
