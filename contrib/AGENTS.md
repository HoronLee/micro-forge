# contrib

可选第三方系统集成。基础包负责官方 Client 构造；capability Adapter 按框架能力分目录。

## OpenFGA

- `contrib/openfga`：generated config 到官方 `*fgaclient.OpenFgaClient`。
- `contrib/authz/openfga`：官方 Client 到 `authz.Authorizer`、`BatchAuthorizer`、`Lister`。
- 两个包互不依赖；不恢复 `security/authz/openfga` 或 `contrib/openfga/authz`。
- 基础 Client 包不镜像 SDK data plane，也不拥有 Redis cache、Audit 或业务语义。
- 关系变更 Audit 由发起变更的业务 Use Case 负责；技术变化使用 SDK `ReadChanges`。

Provider error 使用稳定类型和 `errors.Is/As` 分类，不匹配错误文本。业务 repository、缓存 key、失效策略和 Audit schema 留在所属领域。

公共 Proto config 变更后运行 `just gen-fresh`、`just gen` 和 Proto lint。
