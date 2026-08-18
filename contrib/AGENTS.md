# contrib

Servora 可选第三方生态接线空间，包含 database、cache、Kafka 与 Kubernetes runtime。

## 组织边界

- base package 负责官方 Client 构造、配置映射、生命周期、日志、tracing、健康检查与 provider-native helper。
- capability Adapter 位于对应能力路径，例如 Ent CRUD 位于 `contrib/db/entgo/crud`，Redis cache strategy 位于 `contrib/cache/redis`。
- `contrib/db/redis` 提供 Redis Client、lock 与 KV helper；`contrib/kafka` 提供 Kafka transport 接线。
- Provider error 使用稳定类型和 `errors.Is/As` 分类。

公共 Proto config 变更后运行 `just gen-fresh`、`just gen` 和 Proto lint。
