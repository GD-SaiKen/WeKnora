# dsh-work 知识库换票 实现计划

> **面向实现者：** 按任务顺序做。每个任务先写失败测试，再写让该测试通过的最少代码。用 `- [ ]` 跟踪。完成一个任务并看到测试通过后再进入下一个。不要自行 git commit，除非执行者被明确要求提交。
>
> **规格：** [26-10-29 dsh-work 知识库换票](../规格文档/26-10-29%20dsh-work%20知识库换票.md)
>
> **目标：** dsh-work 服务用集成密钥换到已有用户的网页级访问令牌，或在本人登录后兑现一次性绑定码。
>
> **架构：** 查找用户与「是否开户」的规则放在现有目录身份之上，不在 handler 里写 SQL。签发令牌复用网页登录已经使用的令牌签发，不新造一种权限更低或更高的令牌。现有知识库 handler 不改。
>
> **技术栈：** Go、Gin。测试用 `go test`，放在被测包的 `*_test.go`。
>
> **不改：**
>
> - 知识库列表、检索、预览的权限判断
> - `/mcp` 与 `mcp_` 令牌
> - 登录页、注册档位、目录同步任务
> - OIDC 自动开户开关的现有语义

## 文件职责

| 路径 | 动作 | 职责 |
| --- | --- | --- |
| `internal/integration/dshwork/secret.go` | 创建 | 比对 `DSH_WORK_INTEGRATION_SECRET`；未配置与不匹配分开 |
| `internal/integration/dshwork/session.go` | 创建 | 按 `ai_hub_user_id`、`subject`、`user_id` 决定换票或拒绝；不创建用户 |
| `internal/integration/dshwork/bind.go` | 创建 | 校验 redirect 允许名单，签发并兑现 `bind_code` |
| `internal/handler/dsh_work_integration.go` | 创建 | 解析 JSON、状态码与规格中的 `error` 字符串 |
| `internal/router/router.go` 或现有 auth 路由文件 | 修改 | 注册三条路径 |
| `internal/application/service/user.go` | 修改 | 仅当已有「为指定用户签发网页访问令牌」的函数不可复用时，抽出一个不改登录语义的签发入口 |

领域包不返回 HTTP 状态码。handler 把「未开通 / 停用 / 重定向不允许 / 绑定码无效」映射成规格里的状态码与 `error`。

## 任务 1：密钥

- [ ] **步骤 1：写失败测试**

`internal/integration/dshwork/secret_test.go`

- 环境变量为空：未配置。
- 与配置不一致：拒绝。
- 一致：通过。比较必须是常数时间（调用现有常量时间比较；测试只断言结果，不度量时间）。

运行：

```bash
go test ./internal/integration/dshwork/ -run TestSecret -count=1
```

预期：FAIL，包不存在。

- [ ] **步骤 2：实现** `secret.go`。

## 任务 2：换票决策

- [ ] **步骤 1：写失败测试**

`internal/integration/dshwork/session_test.go`，使用内存中的用户与目录身份，不访问数据库。

- 仅 `ai_hub_user_id` 命中：返回该用户。
- 仅 `subject` 命中：返回该用户。
- 两者都传且指向不同用户：以 `ai_hub_user_id` 为准。
- 都未命中：未开通，且创建用户的次数为 0。
- 用户停用：停用。
- 仅 `user_id` 且用户存在：返回该用户。未知 `user_id`：未开通。
- 三个标识都空：拒绝，不查用户。

运行：

```bash
go test ./internal/integration/dshwork/ -run TestSession -count=1
```

- [ ] **步骤 2：实现** `session.go`。持久化查找接到现有目录身份仓库；单测用接口替身。

## 任务 3：绑定码

- [ ] **步骤 1：写失败测试**

`internal/integration/dshwork/bind_test.go`

- `redirect_uri` 不在允许名单：拒绝。
- 在名单内：发出的登录 URL 含一次性参数；10 分钟后兑现失败。
- 兑现一次后再次兑现失败。
- 登录未完成时不能兑换出 `user_id`。

运行：

```bash
go test ./internal/integration/dshwork/ -run TestBind -count=1
```

- [ ] **步骤 2：实现** `bind.go`。绑定码只存哈希。允许名单来自 `DSH_WORK_BIND_REDIRECT_ALLOWLIST`。

## 任务 4：HTTP

- [ ] **步骤 1：写失败测试**

`internal/handler/dsh_work_integration_test.go`，用 httptest 打三条路由。

- 未配置密钥：三条路径都是 404。
- 错误密钥：401。
- 换票成功的 JSON 含 `user_id`、`tenant_id`、`access_token`、`expires_in`，且不含 `refresh_token`。
- 未开通：404，`error` 为 `account_not_provisioned`。
- 停用：403，`error` 为 `account_disabled`。
- 兑现无效绑定码：400，`error` 为 `bind_code_invalid`。

运行：

```bash
go test ./internal/handler/ -run TestDshWorkIntegration -count=1
```

- [ ] **步骤 2：实现** handler 与路由。令牌签发复用网页登录的签发函数。`expires_in` 不超过 3600。

## 规格覆盖

| 规格 | 任务 |
| --- | --- |
| §4 密钥 | 任务 1、4 |
| §5 换票与不开户 | 任务 2、4 |
| §6 绑定码、允许名单、按 `user_id` 再换票 | 任务 2、3、4 |
| §3 不改知识库权限与 MCP | 不改清单 |

## 手工验收

用 curl 对本地 WeKnora：目录用户换票后，用返回的 `access_token` 调用 `GET /api/v1/knowledge-bases`，与同一用户网页登录后的列表一致。未在目录中的 subject 不增加用户行。
