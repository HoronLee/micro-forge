# obs/audit

CloudEvents Audit 运行时：`Auditor`、事件 helper、Kratos middleware 与 noop/stdout/log/kafka/multi 后端。

```go
type Auditor interface {
    Emit(context.Context, cloudevents.Event) error
}
```

`audit.Middleware` 在 handler 返回后按生成规则发送通用 RPC event。AuthN/AuthZ middleware 直接发送各自 typed event；每层只表达自己的结果。发送失败记录日志，不改变业务响应。

`NewEvent` 设置 source，并从 sampled span 补 `traceparent`/`tracestate`。稳定领域字段放 typed data，少量平台路由信息才放 extension。不要恢复旧 runtime detail、`authid` 或公开 extension 常量。

`multi.New` 负责 fanout 与 `Close`/`Flush` 传播。

```bash
go test ./obs/audit/...
```
