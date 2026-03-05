# 分页查询实现说明（面向框架使用者）

## 1. 目标与范围

本文描述 `ListUsers` 分页链路在当前 Kratos + DDD 架构中的落地方式，重点覆盖：

- 协议契约：`PaginationRequest/PaginationResponse` 的模式设计
- 分层职责：`service -> biz -> data` 的边界
- 查询行为：默认值、排序、count、limit/offset
- 当前状态：page 模式已实现，cursor 模式为协议预留

---

## 2. 分层职责与调用路径

### 2.1 Proto（契约层）

分页协议定义在 `api/protos/pagination/v1/pagination.proto`：

- `PaginationRequest` 通过 `oneof mode` 承载两种模式：
  - `page`：`page` + `page_size`
  - `cursor`：`cursor` + `limit`
- 参数限制由 `buf.validate` 声明（`page_size`、`limit` 上限 100）

关键位置：

- `api/protos/pagination/v1/pagination.proto:7`
- `api/protos/pagination/v1/pagination.proto:23`
- `api/protos/pagination/v1/pagination.proto:41`

`User` 业务接口直接复用该协议：

- `ListUsersRequest.pagination`：`api/protos/user/service/v1/user.proto:51`
- `ListUsersResponse.pagination`：`api/protos/user/service/v1/user.proto:55`

### 2.2 Service（传输适配层）

`app/servora/service/internal/service/user.go:38`

`ListUsers` 仅负责：

1. 提取分页请求对象 `req.GetPagination()`
2. 调用 usecase `s.uc.ListUsers(...)`
3. 将领域对象映射为 pb 响应对象
4. 回填分页响应（空值兜底）

该层不承载分页算法，仅承载协议适配与响应组装。

### 2.3 Biz（用例编排层）

`app/servora/service/internal/biz/user.go:105`

`ListUsers` 负责分页语义编排：

- 默认值：`page=1`、`pageSize=20`
- 模式读取：当前仅消费 `pagination.GetPage()`
- 参数下沉：调用 repo `ListUsers(ctx, page, pageSize)`
- 响应封装：返回 `PaginationResponse_Page{Total, Page, PageSize}`

关键位置：

- `app/servora/service/internal/biz/user.go:106`
- `app/servora/service/internal/biz/user.go:110`
- `app/servora/service/internal/biz/user.go:120`
- `app/servora/service/internal/biz/user.go:125`

### 2.4 Data（持久化执行层）

`app/servora/service/internal/data/user.go:95`

`ListUsers` 负责将用例参数翻译为 Ent 查询：

1. 计算分页窗口：`offset := int((page - 1) * pageSize)`
2. 固定排序：`Order(user.ByID(sql.OrderDesc()))`
3. 总数统计：`query.Clone().Count(ctx)`
4. 当前页数据：`query.Offset(offset).Limit(limit).All(ctx)`

关键位置：

- `app/servora/service/internal/data/user.go:96`
- `app/servora/service/internal/data/user.go:99`
- `app/servora/service/internal/data/user.go:100`
- `app/servora/service/internal/data/user.go:105`

---

## 3. HTTP 入参与协议映射

OpenAPI 暴露了两种模式的 query 参数：

- `pagination.page.page`
- `pagination.page.pageSize`
- `pagination.cursor.cursor`
- `pagination.cursor.limit`

证据：`app/servora/service/openapi.yaml:214`。

这说明对外契约已具备 page/cursor 双模式表达能力。

---

## 4. 当前实现结论

### 4.1 已实现

- Page 模式完整链路可用：proto -> service -> biz -> data
- 查询包含稳定排序、总数统计与窗口查询
- 返回体包含 `total/page/page_size`

### 4.2 预留未落地

- Cursor 模式当前停留在契约层与参数层
- Biz/Data 查询逻辑尚未进入 `GetCursor()` 分支执行

可见证据：

- 协议定义 cursor：`api/protos/pagination/v1/pagination.proto:15`
- Biz 当前仅处理 page：`app/servora/service/internal/biz/user.go:110`

---

## 5. 作为框架使用者的落地要点

新增列表接口时，建议保持同构：

1. proto 层统一复用 `pagination.v1`，避免重复定义分页字段
2. service 层只做参数透传与响应映射，不写分页算法
3. biz 层统一维护默认值与模式分发（page/cursor）
4. data 层确保显式排序 + count + limit/offset（或 cursor where）

该约束能保证各业务列表接口在行为上保持一致，降低调用端心智负担。
