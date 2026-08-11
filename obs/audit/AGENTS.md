# obs/audit

CloudEvents Audit 运行时：`Auditor`、事件 helper、Kratos middleware 与 noop/stdout/log/kafka/multi 后端。

```go
type Auditor interface {
    Emit(context.Context, cloudevents.Event) error
}
```

`audit.Middleware` 根据生成规则在 handler 返回后发送通用 RPC event。发送失败写入日志，并返回原 handler 响应。

`NewEvent` 设置 source，并从 sampled span 添加 `traceparent` 和 `tracestate`。事件 data 保存领域字段，extension 保存路由字段。

`multi.New` 负责 fanout 与 `Close`/`Flush` 传播。

```bash
go test ./obs/audit/...
```
