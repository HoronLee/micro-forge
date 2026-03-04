## 上下文

`svr` CLI 基于 Cobra 构建，现有 `gen` 命令组（`gen gorm`）已建立了命令注册模式。新增 `new` 命令组遵循相同模式：在 `cmd/svr/internal/cmd/new/` 下实现，并在 `root.go` 中注册。

项目 proto 文件遵循 `api/protos/<service>/service/v1/<service>.proto` 结构，包名为 `<service>.service.v1`，服务名为 `<ServicePascal>Service`。模板文件使用 `template` 作为占位词，生成器负责转换为目标名称的各种大小写形式。

## 目标 / 非目标

**目标：**
- 实现 `svr new api <name>` 命令，生成符合 servora 规范的 proto 脚手架
- 支持点分层级输入（`test.test1` → `api/protos/test/test1/service/v1/`）
- 模板三级查找：`--template` 标志 → 本地 `api/protos/template/service/v1/` → embed 内嵌
- 命名感知替换（snake_case / PascalCase / 全大写三种模式）
- 目标目录已存在时报错退出（非零 exit code）
- 在 `AGENTS.md` 记录 `svr` 须在项目根目录执行的约定

**非目标：**
- 不自动运行 `buf generate`（生成 Go 代码）
- 不修改 `buf.yaml` 或其他 Buf 配置
- 不支持除 proto 以外的文件类型脚手架
- 不做项目根目录自动检测（调用方保证 cwd）

## 决策

### D1：模板三级查找（fallback 链）

```
--template <dir>
    ↓ (未指定)
./api/protos/template/service/v1/
    ↓ (不存在)
//go:embed template/protos/*
```

**理由**：embed 保证任意环境可用；本地模板允许项目定制；`--template` 支持 CI 场景下指定绝对路径。

**embed 文件位置**：
```
cmd/svr/internal/cmd/new/
└── template/
    └── protos/
        ├── template.proto
        └── template_doc.proto
```

### D2：输入格式约束为 snake_case（含点层级）

正则：`^[a-z][a-z0-9]*(_[a-z0-9]+)*(\.[a-z][a-z0-9]*(_[a-z0-9]+)*)*$`

示例：`test`、`say_hello`、`test.test1`

**理由**：proto package 名只允许小写字母、数字、下划线；禁止驼峰输入消除歧义；点号映射目录层级与 proto 规范一致。

### D3：命名替换策略（感知上下文，非暴力替换）

模板中使用 `template` 作为占位词，替换时按三种模式顺序处理（长串优先，防止部分匹配）：

| 模板中出现 | 替换为 | 示例（输入 `say_hello`） |
|------------|--------|--------------------------|
| `Template` (PascalCase) | PascalCase | `SayHello` |
| `TEMPLATE` (全大写) | 全大写 | `SAY_HELLO` |
| `template` (小写) | 原始 snake_case | `say_hello` |

转换逻辑：`say_hello` → 按 `_` 分割 → 每段首字母大写 → 拼接 → `SayHello`

替换顺序：先替换 `Template`（PascalCase），再替换 `TEMPLATE`，最后替换 `template`，避免小写替换污染大写模式。

### D4：目录结构生成规则

输入 `<name>`（支持点分层级）：

```
单词:  test
  目录: api/protos/test/service/v1/
  文件: test.proto, test_doc.proto
  包名: test.service.v1

层级:  test.test1
  目录: api/protos/test/test1/service/v1/
  文件: test_test1.proto, test_test1_doc.proto
  包名: test.test1.service.v1
```

文件名取点分各段用 `_` 拼接（`test.test1` → `test_test1`）。

### D5：冲突处理

目标目录已存在 → 打印错误信息 → `os.Exit(1)`

**理由**：CI 场景下非零退出码可被流水线捕获，防止意外覆盖。

## 风险 / 权衡

- **模板同步风险**：embed 模板与 `api/protos/template/` 演示模板可能不同步 → 两者均纳入版本控制，PR 时人工检查
- **替换误伤**：若模板内容中出现与占位词无关的 `template` 字符串（如注释中引用其他服务名），会被错误替换 → embed 模板由框架维护，内容严格控制，风险可控
- **根目录假设**：输出路径 `./api/protos/` 依赖调用方在项目根执行，无自动检测 → 通过 AGENTS.md 约定 + 错误提示缓解

## 迁移计划

纯新增，无破坏性变更，无需迁移步骤。发布后旧用户工作流不受影响。

## 开放问题

（无）
