# security/authz

引擎中立的资源授权合同与 Kratos middleware。

## 合同

```go
type Resource struct { Type, ID string }
type CheckRequest struct {
    Subject string
    Action string
    Resource Resource
    Attributes map[string]any
}
type Authorizer interface {
    Check(context.Context, CheckRequest) (bool, error)
}
```

`BatchAuthorizer` 与 `Lister` 是可选扩展。`Attributes` 只承载当前决策事实，不放凭据、持久化数据或 provider 专用 tuple。

`Server` 默认从 `authn.SubjectFrom` 读取主体。缺规则返回 500；缺主体返回 AuthN 401；资源解析失败返回 400；`(false, nil)` 返回 403；`ErrUnavailable` 返回 503；其他错误返回 500。incoming deadline 原样传递。

CHECK 规则由生成器校验 action、resource type 和 singular scalar ID 字段路径。根层负责 public error、typed Audit 与框架日志；日志不记录 Subject、Resource.ID 或 cause 文本，backend 不重复发送 decision Audit。`CheckRequest.Subject` 必须是稳定且不可重放的身份标识。

OpenFGA Client 位于 `contrib/openfga`，AuthZ Adapter 位于 `contrib/authz/openfga`。

```bash
go test ./security/authz ./cmd/protoc-gen-servora-authz
```
