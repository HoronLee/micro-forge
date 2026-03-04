## 1. Embed 模板文件

- [x] 1.1 创建 `cmd/svr/internal/cmd/new/template/protos/template.proto`，内容参照 `api/protos/sayhello/service/v1/sayhello.proto`，将 sayhello 相关命名替换为 template 占位词
- [x] 1.2 创建 `cmd/svr/internal/cmd/new/template/protos/template_doc.proto`，内容参照 `api/protos/sayhello/service/v1/sayhello_doc.proto`，将 sayhello 相关命名替换为 template 占位词
- [x] 1.3 创建 `api/protos/template/service/v1/template.proto`（与 embed 模板内容相同，作为项目级演示模板）
- [x] 1.4 创建 `api/protos/template/service/v1/template_doc.proto`（同上）

## 2. 命令结构

- [x] 2.1 创建 `cmd/svr/internal/cmd/new/new.go`，定义 `new` 命令组并暴露 `Register(parent)` 函数
- [x] 2.2 修改 `cmd/svr/internal/root/root.go`，导入并注册 `new` 命令组

## 3. 命名工具函数

- [x] 3.1 在 `cmd/svr/internal/cmd/new/api.go` 中实现输入验证函数，校验正则 `^[a-z][a-z0-9]*(_[a-z0-9]+)*(\.[a-z][a-z0-9]*(_[a-z0-9]+)*)*$`
- [x] 3.2 实现 `toSnake(name string) string`：点分输入各段保持原样，返回 snake_case 形式（点分时各段用 `_` 拼接作为文件名）
- [x] 3.3 实现 `toPascal(name string) string`：将 snake_case 段按 `_` 分割后每段首字母大写拼接（`say_hello` → `SayHello`，`test.test1` → `TestTest1`）
- [x] 3.4 实现 `toUpper(name string) string`：snake_case 转全大写（`say_hello` → `SAY_HELLO`）
- [x] 3.5 实现模板内容替换函数，按顺序替换 `Template`→Pascal、`TEMPLATE`→Upper、`template`→snake，避免二次替换

## 4. 模板查找逻辑

- [x] 4.1 实现模板加载函数，优先级：`--template` 标志 → `./api/protos/template/service/v1/` → embed FS
- [x] 4.2 `--template` 指定路径时，校验目录存在且包含 `template.proto` 和 `template_doc.proto`，否则报错退出
- [x] 4.3 本地路径查找时，静默跳过（不报错），fallback 到 embed

## 5. `svr new api` 子命令实现

- [x] 5.1 在 `api.go` 中实现 `NewApiCmd()` 返回 `*cobra.Command`，注册 `--template` 和 `--output` 标志
- [x] 5.2 实现目录结构计算：输入点分段 → 目录路径（`test.test1` → `test/test1/service/v1/`）
- [x] 5.3 实现目标目录冲突检测：目录已存在时打印错误并 `os.Exit(1)`
- [x] 5.4 实现文件生成主流程：加载模板 → 替换内容 → 创建目录 → 写出两个 proto 文件
- [x] 5.5 在 `new.go` 的 `Register` 中添加 `NewApiCmd()`
- [x] 5.6 打印成功信息，显示生成的文件路径

## 6. AGENTS.md 更新

- [x] 6.1 在根目录 `AGENTS.md` 的 `cmd/svr/` 章节补充：`svr` 命令设计为在项目根目录执行，`svr new api` 默认输出路径 `./api/protos/` 相对于当前工作目录解析，CI 中须先 `cd` 到项目根目录

## 7. 验证

- [x] 7.1 运行 `go build ./cmd/svr/...` 确认编译通过
- [x] 7.2 运行 `svr new api test` 验证生成 `api/protos/test/service/v1/test.proto` 和 `test_doc.proto`，内容中占位词已正确替换
- [x] 7.3 运行 `svr new api say_hello` 验证 PascalCase 转换正确（`SayHelloService`、`SayHelloRequest` 等）
- [x] 7.4 运行 `svr new api test.test1` 验证点分层级生成 `api/protos/test/test1/service/v1/`
- [x] 7.5 重复运行 `svr new api test` 验证冲突检测报错且 exit code 非零
- [x] 7.6 运行 `svr new api Test` 验证非法输入报错退出
- [x] 7.7 删除测试生成的目录，清理验证产物
