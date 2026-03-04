## 新增需求

### 需求:输入名称格式验证
命令必须只接受符合 proto 包名规范的输入，格式为 snake_case 单词，支持点号分隔的多层级。
输入必须匹配正则 `^[a-z][a-z0-9]*(_[a-z0-9]+)*(\.[a-z][a-z0-9]*(_[a-z0-9]+)*)*$`。
不合法输入必须打印错误信息并以非零 exit code 退出。

#### 场景:合法单词输入
- **当** 用户运行 `svr new api test`
- **那么** 命令正常执行，生成 `api/protos/test/service/v1/` 下的 proto 文件

#### 场景:合法点分层级输入
- **当** 用户运行 `svr new api test.test1`
- **那么** 命令正常执行，生成 `api/protos/test/test1/service/v1/` 下的 proto 文件

#### 场景:合法 snake_case 输入
- **当** 用户运行 `svr new api say_hello`
- **那么** 命令正常执行，生成 `api/protos/say_hello/service/v1/` 下的 proto 文件

#### 场景:非法大写输入
- **当** 用户运行 `svr new api Test` 或 `svr new api sayHello`
- **那么** 命令打印格式错误信息，以非零 exit code 退出，不创建任何文件

#### 场景:非法特殊字符输入
- **当** 用户运行 `svr new api say-hello` 或 `svr new api _test` 或 `svr new api test_`
- **那么** 命令打印格式错误信息，以非零 exit code 退出，不创建任何文件

### 需求:目录结构生成
命令必须根据输入名称生成正确的目录结构和文件名。
点分层级中各段必须成为独立目录层级，文件名必须将各段用下划线拼接。

#### 场景:单词生成目录结构
- **当** 输入为 `test`
- **那么** 创建目录 `api/protos/test/service/v1/`，生成文件 `test.proto` 和 `test_doc.proto`

#### 场景:点分层级生成嵌套目录
- **当** 输入为 `test.test1`
- **那么** 创建目录 `api/protos/test/test1/service/v1/`，生成文件 `test_test1.proto` 和 `test_test1_doc.proto`

### 需求:Proto 文件内容的命名替换
生成的 proto 文件内容必须将模板占位词 `template` 替换为目标名称的对应大小写形式。
替换必须感知上下文，按三种模式处理，禁止暴力全局替换相同字符串。

#### 场景:小写替换（包名、文件引用）
- **当** 模板中出现 `template`（全小写）
- **那么** 替换为输入的 snake_case 形式（如 `say_hello`）

#### 场景:PascalCase 替换（服务名、消息名）
- **当** 模板中出现 `Template`（首字母大写）
- **那么** 替换为输入的 PascalCase 形式（如 `SayHello`）

#### 场景:全大写替换
- **当** 模板中出现 `TEMPLATE`（全大写）
- **那么** 替换为输入的全大写形式（如 `SAY_HELLO`）

#### 场景:多段点分名称的 PascalCase 转换
- **当** 输入为 `test.test1`（点分多段）
- **那么** PascalCase 形式为 `TestTest1`（各段首字母大写后拼接）

### 需求:模板三级查找
命令必须按优先级查找模板：`--template` 标志 > 本地项目模板 > embed 内嵌模板。
只有在高优先级来源不可用时才 fallback 到下一级。

#### 场景:使用 --template 标志指定模板
- **当** 用户运行 `svr new api test --template /path/to/dir`，且该目录包含 `template.proto` 和 `template_doc.proto`
- **那么** 使用指定目录的模板文件

#### 场景:--template 指定路径不存在
- **当** 用户运行 `svr new api test --template /nonexistent/`
- **那么** 命令报错退出，提示模板目录不存在，不创建任何文件

#### 场景:使用本地项目模板
- **当** 未指定 `--template`，且 `./api/protos/template/service/v1/` 目录存在并包含模板文件
- **那么** 使用本地项目模板

#### 场景:fallback 到 embed 模板
- **当** 未指定 `--template`，且本地项目模板目录不存在
- **那么** 使用 embed 内嵌的默认模板

### 需求:目标目录冲突检测
命令必须在生成文件前检查目标目录是否已存在。
目标目录已存在时，命令必须打印错误信息并以非零 exit code 退出，禁止覆盖已有文件。

#### 场景:目标目录已存在
- **当** 用户运行 `svr new api test`，但 `api/protos/test/service/v1/` 已存在
- **那么** 命令打印冲突错误信息，以非零 exit code 退出，不修改任何现有文件

#### 场景:目标目录不存在
- **当** 用户运行 `svr new api test`，且 `api/protos/test/service/v1/` 不存在
- **那么** 命令创建目录及文件，打印成功信息

### 需求:输出路径可配置
命令必须支持 `--output` 标志覆盖默认输出根目录。
未指定时默认使用 `./api/protos/`（相对于当前工作目录）。

#### 场景:默认输出路径
- **当** 用户未指定 `--output`，在项目根目录运行 `svr new api test`
- **那么** 文件写入 `./api/protos/test/service/v1/`

#### 场景:指定输出路径
- **当** 用户运行 `svr new api test --output /custom/path`
- **那么** 文件写入 `/custom/path/test/service/v1/`

## 修改需求

## 移除需求
