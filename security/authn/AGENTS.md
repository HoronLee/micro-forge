# security/authn

引擎中立的认证调度器；具体凭据解析由下游 Authenticator 实现。

## 合同

```go
type Authentication struct { Subject string }
type Scheme string
type Authenticator interface {
    Scheme() Scheme
    Authenticate(context.Context) (Authentication, error)
}
```

`Server([]Authenticator, ...Option)` 在构造期校验 nil、空/重复 Scheme 和规则引用。public RPC 直接通过；其余请求按装配顺序尝试允许的 Scheme，`ErrNoCredentials` 继续，凭据拒绝或 provider 错误立即停止，首个成功结果写入包私 context。

公开读取入口：`AuthenticationFrom`、`SubjectFrom`。不要恢复 `Multi`、`Named`、JWT/API Key 内置 Adapter 或可写 context helper。

错误通过 `ErrCredentialsRejected`、`ErrUnavailable` 和其他内部错误分类为 401/503/500；public error 保留服务端 cause，不向 wire 暴露原始错误。

`WithAuditor` 发送 typed `servora.authn.success.v1` / `servora.authn.failure.v1`；`WithLogger` 只记录 scheme/result/reason 等稳定字段，不记录 Subject 或 cause 文本。`Authentication.Subject` 必须是稳定且不可重放的身份标识，绝不能放 credential。

```bash
go test ./security/authn
```
