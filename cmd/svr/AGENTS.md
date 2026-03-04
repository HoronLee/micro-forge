# AGENTS.md - cmd/svr/ CLI 工具

<!-- Parent: ../../AGENTS.md -->
<!-- Generated: 2026-03-03 -->

## 目录概述

`cmd/svr/` 是 servora 项目的中心化 CLI 工具，基于 Cobra 构建，提供代码生成、脚手架等开发辅助命令。设计目标是将各服务中重复的开发工具逻辑抽象为统一入口，消除跨服务的代码重复。

**核心价值**：
- 统一的代码生成入口（替代各服务内的独立脚本）
- 可扩展的命令树（Cobra 命令分组 + 模块化注册）
- 美化的终端输出（lipgloss AdaptiveColor 浅色/深色主题自适应）
- 交互式操作（huh 多选/确认组件）

## 命令树

```bash
svr
├── gen
│   └── gorm <service-name...> [--dry-run]           # GORM GEN 代码生成
├── new
│   └── api <name> [--template dir] [--output dir]   # 创建 proto API 骨架
├── completion                                         # Shell 自动补全（Cobra 内置）
├── help                                               # 帮助信息
└── (未来扩展)
    ├── gen dao                                        # Ent 代码生成
    └── new svc                                        # 创建微服务骨架
```

## 目录结构

```
cmd/svr/
├── main.go                          # 入口文件
├── AGENTS.md                        # 本文件
└── internal/
    ├── root/
    │   └── root.go                  # 根命令（注册所有命令组）
    ├── cmd/
    │   ├── gen/
    │   │   ├── gen.go               # gen 命令组注册
    │   │   └── gorm.go              # gorm 子命令（批量生成 + 交互模式）
    │   └── new/
    │       ├── new.go               # new 命令组注册
    │       └── api.go               # new api 子命令（proto 脚手架生成）
    ├── discovery/
    │   └── config.go                # 服务发现与配置加载
    ├── generator/
    │   └── gorm.go                  # GORM GEN 生成器封装
    └── ux/
        └── output.go                # 终端输出美化（lipgloss 样式）
```

## 模块职责

### `internal/root/` - 根命令
- 定义 `svr` 根命令
- 注册所有命令组（`gen.Register(rootCmd)`、`new.Register(rootCmd)`）
- 暴露 `Execute()` 入口

### `internal/cmd/` - 命令实现
每个命令组一个子目录，每个子命令一个文件。

**gen/gen.go**：注册 `gen` 命令组
**gen/gorm.go**：实现 `svr gen gorm` 子命令
- 支持多服务名参数（`svr gen gorm servora sayhello`）
- 无参数时进入 huh 交互式多选
- `--dry-run` 预览模式
- 批量执行，失败不中断，最终汇总
- 4 种错误分类：`service-not-found` / `config-invalid` / `db-connect-failed` / `generation-failed`
- 退出码：全成功=0，存在失败=1

**new/new.go**：注册 `new` 命令组
**new/api.go**：实现 `svr new api` 子命令
- 输入格式：snake_case，支持点号层级（`test`、`say_hello`、`billing.invoice`）
- 二级模板查找：`--template` 标志 → `./api/protos/template/service/v1/`（项目根相对路径）
- 感知上下文的命名替换：`template` → snake、`Template` → PascalCase、`TEMPLATE` → UPPER
- proto package 行保留点分形式（`billing.invoice.service.v1`）符合 buf 目录映射规范
- 目标目录已存在时报错退出（exit code 1），适合 CI 使用
- 标志：`--template <dir>`、`--output <dir>`（默认 `./api/protos/`）
- 模板缺失时输出带 hint 的错误：提示从项目根目录执行或使用 `--template`

### `internal/discovery/` - 服务发现
- `LoadServiceConfig()` — 加载服务的 Bootstrap 配置（复用 `pkg/bootstrap/config/loader.LoadBootstrap()`）
- `ValidateServiceExists()` / `ValidateConfigExists()` / `ValidateDatabaseConfig()` — 验证函数
- `ListAvailableServices()` — 扫描 `app/*/service` 目录

### `internal/generator/` - 代码生成器
- `GormGenerator` 结构体 — 封装 GORM GEN 生成逻辑
- `connectDB()` — 支持 MySQL/PostgreSQL/SQLite（`strings.ToLower` 驱动名）
- `Generate()` — 配置并执行 GORM GEN（`WithDefaultQuery | WithQueryInterface`，`FieldNullable: true`）
- 输出路径：`{servicePath}/internal/data/gorm/{dao,po}`

### `internal/ux/` - 终端输出
- 8 种样式 token：`Title`、`Info`、`Success`、`Warn`、`Error`、`Path`、`Counter`、`Summary`
- 全部使用 `lipgloss.AdaptiveColor`（自适应浅色/深色终端主题）
- 输出函数：`PrintSuccess`、`PrintError`、`PrintInfo`、`PrintProgress`、`PrintSummary`、`PrintFailureDetail`、`PrintDryRun`、`PrintDBConnected`、`PrintGenerated`

## AI Agent 工作指南

### 添加新命令

**标准流程**（以添加 `svr gen dao` 为例）：

1. **创建命令文件**：
```go
// internal/cmd/gen/dao.go
package gen

import "github.com/spf13/cobra"

func NewDaoCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "dao [service-name...]",
        Short: "Generate Ent DAO code for services",
        RunE: func(cmd *cobra.Command, args []string) error {
            // 实现逻辑
            return nil
        },
    }
    return cmd
}
```

2. **注册到命令组**：
```go
// internal/cmd/gen/gen.go
func Register(parent *cobra.Command) {
    genCmd := &cobra.Command{
        Use:   "gen",
        Short: "Code generation commands",
    }
    genCmd.AddCommand(NewGormCmd())
    genCmd.AddCommand(NewDaoCmd())  // 新增
    parent.AddCommand(genCmd)
}
```

3. **如果需要新的命令组**（如 `svr new`）：
```go
// internal/cmd/new/new.go
package new

func Register(parent *cobra.Command) {
    newCmd := &cobra.Command{Use: "new", Short: "Create new resources"}
    newCmd.AddCommand(NewApiCmd())
    newCmd.AddCommand(NewSvcCmd())
    parent.AddCommand(newCmd)
}
```

然后在 `internal/root/root.go` 中注册：
```go
func init() {
    gen.Register(rootCmd)
    new.Register(rootCmd)  // 新增
}
```

### 添加新的发现/生成逻辑

- **服务发现逻辑** → `internal/discovery/`（如 Ent 能力判定）
- **代码生成逻辑** → `internal/generator/`（如 Ent 生成器封装）
- **模板渲染** → `internal/scaffold/`（未来脚手架功能）

### 使用 UX 模块

所有终端输出必须通过 `internal/ux/` 模块，确保样式一致：

```go
import "github.com/horonlee/servora/cmd/svr/internal/ux"

ux.PrintSuccess("service", "generated successfully")
ux.PrintError("service", "config not found")
ux.PrintProgress(1, 5, "generating servora")
ux.PrintSummary(4, 1)
ux.PrintFailureDetail("badservice", "service-not-found", "not found at app/badservice/service")
```

### 编译和测试

```bash
# 编译
go build ./cmd/svr/...

# gen gorm
go run ./cmd/svr gen gorm servora
go run ./cmd/svr gen gorm servora --dry-run
go run ./cmd/svr gen gorm  # 交互模式

# new api
go run ./cmd/svr new api user
go run ./cmd/svr new api say_hello
go run ./cmd/svr new api billing.invoice
go run ./cmd/svr new api user --output /custom/path
go run ./cmd/svr new api user --template /custom/templates

# 通过 make（在服务目录下）
cd app/servora/service && make gen.gorm
```

## 依赖关系

### 项目内依赖
- `pkg/bootstrap/config/` — 配置加载（`LoadBootstrap`）
- `api/gen/go/conf/v1` — 配置 proto 类型（`*conf.Bootstrap`、`*conf.Data_Database`）

### 外部依赖
- `github.com/spf13/cobra` — 命令行框架
- `github.com/charmbracelet/lipgloss` — 终端样式
- `github.com/charmbracelet/huh` — 交互式表单
- `gorm.io/gen` — GORM GEN 代码生成
- `gorm.io/gorm` + 驱动（mysql, postgres, sqlite）

## 设计约束

### 命令层不做业务逻辑
命令文件（`internal/cmd/`）只做参数校验和流程编排，具体逻辑下沉到 `discovery/`、`generator/`、`scaffold/` 等模块。

### 输出统一
所有面向用户的输出必须通过 `internal/ux/output.go`，不要在命令或生成器中直接 `fmt.Println`。

### 错误处理
- 返回 `error`，不要 `log.Fatal` 或 `panic`
- 错误输出到 `os.Stderr`（通过 `ux.PrintError`）
- 批量操作中失败不中断，最终汇总

### 汇总输出格式（可解析）
```
Summary: success=X failed=Y
- <service> [<error-type>] <message>
```

## 注意事项

- `svr` 必须从**项目根目录**运行（路径计算基于 `app/{name}/service`）
- 配置加载使用 `pkg/bootstrap/config/loader.LoadBootstrap()`，支持配置中心 + 环境变量覆盖
- 交互模式需要 TTY（管道或非终端环境下跳过交互）
- `--dry-run` 模式不连接数据库，仅输出路径
